import { describe, expect, test, vi } from "vitest";

import { handleRequest, type Environment } from "./index";

function environment(assetFetch = vi.fn(() => Promise.resolve(new Response("asset")))) {
  return {
    ASSETS: { fetch: assetFetch },
    RENDER_ORIGIN: "https://aether-api.example.com",
    AETHER_GATEWAY_SECRET: "gateway-secret",
  } satisfies Environment;
}

describe("Cloudflare frontend gateway", () => {
  test("serves assets without contacting Render", async () => {
    const assetFetch = vi.fn(() => Promise.resolve(new Response("asset")));
    const upstreamFetch = vi.fn();

    const response = await handleRequest(
      new Request("https://vault.example/library"),
      environment(assetFetch),
      upstreamFetch,
    );

    expect(await response.text()).toBe("asset");
    expect(assetFetch).toHaveBeenCalledOnce();
    expect(upstreamFetch).not.toHaveBeenCalled();
  });

  test.each(["/api/v1/documents?tag=tax", "/auth/google/start?next=%2F"])(
    "proxies %s while preserving the browser contract",
    async (path) => {
      const upstreamFetch = vi.fn(async (request: Request) => {
        expect(request.url).toBe(`https://aether-api.example.com${path}`);
        expect(request.method).toBe("POST");
        expect(request.redirect).toBe("manual");
        expect(request.headers.get("Cookie")).toBe("aether_session=session-token");
        expect(await request.text()).toBe("request-body");
        return new Response("response-body", {
          status: 201,
          headers: { "Set-Cookie": "aether_session=new-token; Secure; HttpOnly" },
        });
      });

      const response = await handleRequest(
        new Request(`https://vault.example${path}`, {
          method: "POST",
          body: "request-body",
          headers: { Cookie: "aether_session=session-token" },
        }),
        environment(),
        upstreamFetch,
      );

      expect(response.status).toBe(201);
      expect(response.headers.get("Set-Cookie")).toBe(
        "aether_session=new-token; Secure; HttpOnly",
      );
      expect(await response.text()).toBe("response-body");
    },
  );

  test("replaces spoofed private and forwarding headers", async () => {
    const upstreamFetch = vi.fn(async (request: Request) => {
      expect(request.headers.get("X-Aether-Gateway-Token")).toBe("gateway-secret");
      expect(request.headers.get("X-Aether-Client-IP")).toBe("203.0.113.9");
      expect(request.headers.get("X-Forwarded-For")).toBeNull();
      expect(request.headers.get("Forwarded")).toBeNull();
      expect(request.headers.get("CF-Connecting-IP")).toBeNull();
      return new Response(null, { status: 204 });
    });
    const request = new Request("https://vault.example/api/v1/session", {
      headers: {
        "CF-Connecting-IP": "203.0.113.9",
        "X-Aether-Gateway-Token": "spoofed",
        "X-Aether-Client-IP": "198.51.100.1",
        "X-Forwarded-For": "198.51.100.2",
        Forwarded: "for=198.51.100.3",
      },
    });

    await handleRequest(request, environment(), upstreamFetch);

    expect(upstreamFetch).toHaveBeenCalledOnce();
  });

  test("maps an upstream network failure to backend_unavailable", async () => {
    const response = await handleRequest(
      new Request("https://vault.example/api/v1/session"),
      environment(),
      vi.fn(() => Promise.reject(new Error("socket closed"))),
    );

    expect(response.status).toBe(502);
    expect(await response.json()).toEqual({ error: "backend_unavailable" });
  });

  test("passes request and response bodies as streams", async () => {
    const requestBody = new ReadableStream({
      start(controller) {
        controller.enqueue(new TextEncoder().encode("streamed upload"));
        controller.close();
      },
    });
    const responseBody = new ReadableStream({
      start(controller) {
        controller.enqueue(new TextEncoder().encode("streamed download"));
        controller.close();
      },
    });
    const upstreamFetch = vi.fn((request: Request) => {
      expect(request.body).toBeInstanceOf(ReadableStream);
      return Promise.resolve(new Response(responseBody));
    });

    const response = await handleRequest(
      new Request("https://vault.example/api/v1/documents", {
        method: "POST",
        body: requestBody,
        duplex: "half",
      }),
      environment(),
      upstreamFetch,
    );

    expect(response.body).toBe(responseBody);
    expect(await response.text()).toBe("streamed download");
  });
});
