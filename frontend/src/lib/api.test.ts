// @vitest-environment jsdom

import { afterEach, expect, test, vi } from "vitest";

import { api, type UploadState } from "./api";

afterEach(() => vi.unstubAllGlobals());

test("reports measured upload progress before server finalization", async () => {
  const states: UploadState[] = [];
  vi.stubGlobal("XMLHttpRequest", ProgressXMLHttpRequest);

  await api.uploadDocument(
    new File(["content"], "document.pdf", { type: "application/pdf" }),
    { onProgress: (state) => states.push(state) },
    "upload-key",
  );

  expect(states.map(({ state, percent }) => [state, percent])).toEqual([
    ["uploading", null],
    ["uploading", 50],
    ["processing", 100],
    ["complete", 100],
  ]);
});

class ProgressXMLHttpRequest {
  readonly upload: {
    onprogress: ((event: ProgressEvent) => void) | null;
    onload: ((event: ProgressEvent) => void) | null;
  } = { onprogress: null, onload: null };
  status = 201;
  responseText = JSON.stringify({ document: { id: "document-1" }, tags: [] });
  withCredentials = false;
  onabort: (() => void) | null = null;
  onerror: (() => void) | null = null;
  onload: (() => void) | null = null;

  open() {}
  setRequestHeader() {}
  getAllResponseHeaders() {
    return "Content-Type: application/json";
  }
  send() {
    this.upload.onprogress?.(
      new ProgressEvent("progress", { lengthComputable: true, loaded: 5, total: 10 }),
    );
    this.upload.onload?.(new ProgressEvent("load"));
    this.onload?.();
  }
}
