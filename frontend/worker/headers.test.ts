import { readFile } from "node:fs/promises";

import { expect, test } from "vitest";

test("static assets retain the backend browser security policy", async () => {
  const headers = await readFile(
    new URL("../public/_headers", import.meta.url),
    "utf8",
  );

  for (const policy of [
    "X-Content-Type-Options: nosniff",
    "Referrer-Policy: no-referrer",
    "Permissions-Policy: camera=(), microphone=(), geolocation=()",
    "Cross-Origin-Opener-Policy: same-origin",
    "frame-ancestors 'none'",
    "base-uri 'none'",
    "form-action 'self'",
  ]) {
    expect(headers).toContain(policy);
  }
});
