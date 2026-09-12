// @vitest-environment jsdom

import "@testing-library/jest-dom/vitest";
import { render, screen } from "@testing-library/react";
import { expect, test, vi } from "vitest";

import { DocumentRow } from "./DocumentRow";

test("renders the contributor name instead of the uploader id", () => {
  const item = {
    document: {
      id: "document-1",
      title: "Barclays Logo",
      originalFilename: "barclays-logo.png",
      mediaType: "image/png",
      sizeBytes: 1024,
      sha256: "checksum",
      status: "ready" as const,
      indexStatus: "not_scheduled" as const,
      uploaderId: "opaque-member-id",
      version: 1,
      createdAt: "2026-09-12T12:00:00Z",
      updatedAt: "2026-09-12T12:00:00Z",
    },
    tags: [],
    uploaderName: "Shantanu Wable",
  };

  render(<DocumentRow item={item} selected={false} onSelect={vi.fn()} />);

  expect(screen.getByText("Shantanu Wable")).toBeInTheDocument();
  expect(screen.queryByText("opaque-member-id")).not.toBeInTheDocument();
});
