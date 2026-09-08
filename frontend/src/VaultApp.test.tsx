// @vitest-environment jsdom

import "@testing-library/jest-dom/vitest";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";

import VaultApp from "./VaultApp";

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

test("shows the waking state after the initial transient session failure", async () => {
  vi.stubGlobal(
    "fetch",
    vi.fn(() =>
      Promise.resolve(
        Response.json({ error: "backend_unavailable" }, { status: 502 }),
      ),
    ),
  );

  render(<VaultApp />);

  expect(await screen.findByRole("status")).toHaveTextContent("Waking vault…");
});
