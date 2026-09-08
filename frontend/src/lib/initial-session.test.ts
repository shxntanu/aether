import { describe, expect, test, vi } from "vitest";

import { ApiError } from "./api";
import { loadInitialSession } from "./initial-session";

const immediateSleep = vi.fn(() => Promise.resolve());

describe("initial session cold-start recovery", () => {
  test("retries a transient failure and reports that the vault is waking", async () => {
    const request = vi
      .fn<() => Promise<string>>()
      .mockRejectedValueOnce(new ApiError(502, "backend_unavailable", "asleep"))
      .mockResolvedValue("session");
    const onTransientFailure = vi.fn();

    await expect(
      loadInitialSession(request, { sleep: immediateSleep, onTransientFailure }),
    ).resolves.toBe("session");

    expect(request).toHaveBeenCalledTimes(2);
    expect(immediateSleep).toHaveBeenCalledWith(5_000, undefined);
    expect(onTransientFailure).toHaveBeenCalledOnce();
  });

  test.each([401, 403])("does not retry HTTP %d", async (status) => {
    const request = vi.fn(() =>
      Promise.reject(new ApiError(status, "authentication_required", "denied")),
    );
    const sleep = vi.fn(() => Promise.resolve());

    await expect(loadInitialSession(request, { sleep })).rejects.toMatchObject({
      status,
    });

    expect(request).toHaveBeenCalledOnce();
    expect(sleep).not.toHaveBeenCalled();
  });

  test("does not retry a malformed authentication response", async () => {
    const failure = new ApiError(401, "unexpected_response", "not JSON");
    const request = vi.fn(() => Promise.reject(failure));
    const sleep = vi.fn(() => Promise.resolve());

    await expect(loadInitialSession(request, { sleep })).rejects.toBe(failure);

    expect(request).toHaveBeenCalledOnce();
    expect(sleep).not.toHaveBeenCalled();
  });

  test("retries malformed upstream responses but stops at the time budget", async () => {
    const failure = new ApiError(200, "unexpected_response", "not JSON");
    const request = vi.fn(() => Promise.reject(failure));
    let elapsedMs = 0;
    const sleep = vi.fn((delayMs: number) => {
      elapsedMs += delayMs;
      return Promise.resolve();
    });

    await expect(
      loadInitialSession(request, {
        sleep,
        retryDelayMs: 5_000,
        maxElapsedMs: 10_000,
        now: () => elapsedMs,
      }),
    ).rejects.toBe(failure);

    expect(request).toHaveBeenCalledTimes(3);
    expect(sleep).toHaveBeenCalledTimes(2);
  });

  test.each([400, 404, 500])(
    "does not retry malformed permanent HTTP %d responses",
    async (status) => {
      const failure = new ApiError(status, "unexpected_response", "not JSON");
      const request = vi.fn(() => Promise.reject(failure));
      const sleep = vi.fn(() => Promise.resolve());

      await expect(loadInitialSession(request, { sleep })).rejects.toBe(failure);

      expect(request).toHaveBeenCalledOnce();
      expect(sleep).not.toHaveBeenCalled();
    },
  );

  test("counts request latency toward the retry deadline", async () => {
    let elapsedMs = 0;
    const failure = new ApiError(503, "backend_unavailable", "asleep");
    const request = vi.fn(() => {
      elapsedMs += 45_000;
      return Promise.reject(failure);
    });
    const sleep = vi.fn((delayMs: number) => {
      elapsedMs += delayMs;
      return Promise.resolve();
    });

    await expect(
      loadInitialSession(request, { sleep, now: () => elapsedMs }),
    ).rejects.toBe(failure);

    expect(request).toHaveBeenCalledTimes(2);
    expect(sleep).toHaveBeenCalledOnce();
  });

  test("cancels a hanging session request at the deadline", async () => {
    vi.useFakeTimers();
    try {
      const request = vi.fn((signal?: AbortSignal) =>
        new Promise<never>((_resolve, reject) => {
          signal?.addEventListener("abort", () => reject(signal.reason), {
            once: true,
          });
        }),
      );
      const result = loadInitialSession(request);
      const rejection = expect(result).rejects.toMatchObject({
        name: "TimeoutError",
      });

      await vi.advanceTimersByTimeAsync(90_000);

      await rejection;
      expect(request).toHaveBeenCalledOnce();
    } finally {
      vi.useRealTimers();
    }
  });

  test("does not add retries to mutating API failures", async () => {
    const fetch = vi.fn(() => Promise.reject(new Error("offline")));
    vi.stubGlobal("fetch", fetch);
    const { api } = await import("./api");

    await expect(api.logout()).rejects.toMatchObject({ status: 0 });

    expect(fetch).toHaveBeenCalledOnce();
    vi.unstubAllGlobals();
  });
});
