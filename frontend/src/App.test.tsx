// @vitest-environment jsdom

import "@testing-library/jest-dom/vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import { cleanup } from "@testing-library/react";

import App from "./App";

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

test("loads the health and session probes on startup", async () => {
  const fetch = vi.fn((input: RequestInfo | URL) => {
    if (String(input) === "/api/v1/health") return Promise.resolve(jsonResponse({ status: "ok" }));
    return Promise.resolve(jsonResponse({ error: "authentication_required" }, 401));
  });
  vi.stubGlobal("fetch", fetch);

  render(<App />);

  expect(screen.getByRole("heading", { name: "API bench" })).toBeVisible();
  await waitFor(() => expect(fetch).toHaveBeenCalledTimes(2));
  expect(screen.getByText(/"status": "ok"/)).toBeVisible();
});

test("sends a document list request from its practical controls", async () => {
  const fetch = vi.fn(() => Promise.resolve(jsonResponse({ documents: [] })));
  vi.stubGlobal("fetch", fetch);

  render(<App />);
  await waitFor(() => expect(fetch).toHaveBeenCalledTimes(2));

  fireEvent.change(screen.getByLabelText("Tag filter"), { target: { value: "tax" } });
  fireEvent.change(screen.getByLabelText("Match"), { target: { value: "any" } });
  fireEvent.click(screen.getByRole("button", { name: "Run list request" }));

  await waitFor(() => expect(fetch).toHaveBeenCalledTimes(3));
  expect(fetch).toHaveBeenLastCalledWith("/api/v1/documents?tag=tax&match=any", { credentials: "same-origin" });
});
