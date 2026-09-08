/** Bindings supplied to the Cloudflare frontend gateway. */
export interface Environment {
  /** Static asset service generated from the Vite distribution directory. */
  ASSETS: { fetch(request: Request): Promise<Response> };
  /** Absolute HTTPS origin of the Render Go service. */
  RENDER_ORIGIN: string;
  /** Shared secret also configured as AETHER_GATEWAY_SECRET on Render. */
  AETHER_GATEWAY_SECRET: string;
}

/** Injectable fetch shape used to test the proxy without making network calls. */
export type UpstreamFetch = (request: Request) => Promise<Response>;

const strippedForwardingHeaders = [
  "CF-Connecting-IP",
  "CF-Connecting-IPv6",
  "Forwarded",
  "True-Client-IP",
  "X-Forwarded-For",
  "X-Forwarded-Host",
  "X-Forwarded-Proto",
  "X-Real-IP",
] as const;

/** Routes one request to static assets or streams it to the Render backend. */
export async function handleRequest(
  request: Request,
  environment: Environment,
  upstreamFetch: UpstreamFetch = fetch,
): Promise<Response> {
  const browserURL = new URL(request.url);
  if (!isBackendPath(browserURL.pathname)) {
    return environment.ASSETS.fetch(request);
  }

  const upstreamURL = new URL(environment.RENDER_ORIGIN);
  upstreamURL.pathname = browserURL.pathname;
  upstreamURL.search = browserURL.search;

  const clientIP = request.headers.get("CF-Connecting-IP") ?? "";
  const routedRequest = new Request(upstreamURL, request);
  const upstreamRequest = new Request(routedRequest, { redirect: "manual" });
  const headers = upstreamRequest.headers;
  headers.delete("X-Aether-Gateway-Token");
  headers.delete("X-Aether-Client-IP");
  for (const header of strippedForwardingHeaders) headers.delete(header);
  headers.set("X-Aether-Gateway-Token", environment.AETHER_GATEWAY_SECRET);
  headers.set("X-Aether-Client-IP", clientIP);

  try {
    return await upstreamFetch(upstreamRequest);
  } catch {
    return Response.json(
      { error: "backend_unavailable" },
      { status: 502, headers: { "Cache-Control": "no-store" } },
    );
  }
}

function isBackendPath(pathname: string): boolean {
  return pathname === "/api" || pathname.startsWith("/api/") ||
    pathname === "/auth" || pathname.startsWith("/auth/");
}

/** Cloudflare module Worker entrypoint. */
const worker = {
  fetch(request: Request, environment: Environment): Promise<Response> {
    return handleRequest(request, environment);
  },
} satisfies ExportedHandler<Environment>;

export default worker;
