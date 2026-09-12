type MemberRole = "member" | "admin";
type MemberStatus = "active" | "disabled";
type DocumentStatus = "uploading" | "ready" | "failed" | "deleted";
type DeletionStatus = "queued" | "processing" | "complete" | "failed";
type IndexStatus =
  | "not_scheduled"
  | "queued"
  | "extracting"
  | "enriching"
  | "indexed"
  | "failed";
type KnownApiErrorCode =
  | "administrator_required"
  | "already_exists"
  | "authentication_required"
  | "backend_unavailable"
  | "catalog_error"
  | "csrf_required"
  | "document_too_large"
  | "deletion_in_progress"
  | "file_required"
  | "idempotency_key_required"
  | "invalid_login"
  | "invalid_range"
  | "invalid_request"
  | "invalid_upload"
  | "login_unavailable"
  | "logout_failed"
  | "membership_disabled"
  | "network_error"
  | "not_found"
  | "rate_limited"
  | "retention_active"
  | "storage_usage_unavailable"
  | "unexpected_response"
  | "unsupported_media_type"
  | "upload_in_progress"
  | "vault_error"
  | "version_conflict";

/** ApiErrorCode is the known backend/client error union plus an unknown-code fallback. */
export type ApiErrorCode =
  | KnownApiErrorCode
  | {
      /** kind identifies backend error codes not yet modeled by this boundary. */
      kind: "unknown_backend_code";
      /** code preserves the raw backend error string for diagnostics and UI fallbacks. */
      code: string;
    };

/** ApiResponse preserves HTTP metadata alongside a typed success body. */
export type ApiResponse<T> = {
  /** status is the numeric HTTP response status. */
  status: number;
  /** headers contains the response headers returned by fetch. */
  headers: Headers;
  /** data is the parsed response body, or null for empty success bodies. */
  data: T;
};

/** Session is the authenticated browser session returned by GET /api/v1/session. */
export type Session = {
  /** member is the authenticated user associated with the current session cookie. */
  member: Member;
  /** csrfToken is captured and sent as X-CSRF-Token on later mutating requests. */
  csrfToken?: string;
};

/** Member is the browser-visible identity record for an allowlisted user. */
export type Member = {
  /** id uniquely identifies the member in admin and audit workflows. */
  id: string;
  /** email is the allowlisted sign-in address. */
  email: string;
  /** displayName is the human-readable name supplied by identity providers. */
  displayName: string;
  /** role controls whether admin-only routes are available. */
  role: MemberRole;
  /** status controls whether the member can authenticate. */
  status: MemberStatus;
  /** createdAt is the server timestamp for allowlist creation. */
  createdAt: string;
  /** updatedAt is the server timestamp for the latest member change. */
  updatedAt: string;
};

/** Document is the browser-visible catalog metadata for an immutable original. */
export type Document = {
  /** id uniquely identifies the catalog document. */
  id: string;
  /** title is the user-editable display title. */
  title: string;
  /** originalFilename is the sanitized upload filename. */
  originalFilename: string;
  /** mediaType is the server-detected content type. */
  mediaType: string;
  /** sizeBytes is the original document size in bytes. */
  sizeBytes: number;
  /** sha256 is the lowercase hexadecimal digest of the original bytes. */
  sha256: string;
  /** status is the document storage lifecycle state. */
  status: DocumentStatus;
  /** indexStatus is the document content-processing lifecycle state. */
  indexStatus: IndexStatus;
  /** uploaderId identifies the member who uploaded the document. */
  uploaderId: string;
  /** version is required for optimistic metadata updates. */
  version: number;
  /** createdAt is the server timestamp for catalog creation. */
  createdAt: string;
  /** updatedAt is the server timestamp for the latest metadata change. */
  updatedAt: string;
  /** deletedAt is present when the document has been soft-deleted. */
  deletedAt?: string;
  /** purgeAfter is the earliest permanent-deletion timestamp when applicable. */
  purgeAfter?: string;
  /** deletionStatus reports asynchronous object-storage trashing progress. */
  deletionStatus?: DeletionStatus;
  /** manifestError reports a failed manifest synchronization for repair. */
  manifestError?: string;
};

/** DocumentRecord combines catalog metadata with every attached tag. */
export type DocumentRecord = {
  /** document contains browser-visible catalog metadata for the immutable original. */
  document: Document;
  /** tags contains reusable tags and the implicit document date tag. */
  tags: Tag[];
  /** uploaderName is the human-readable contributor name for the document. */
  uploaderName: string;
};

/** SearchFilters combines selected exact tags with fuzzy metadata search. */
export type SearchFilters = {
  /** tags are case-insensitive tag names that must match the result. */
  tags?: string[];
  /** match controls whether every or any selected tag is required. */
  match?: "all" | "any";
};

/** SearchMatchEvidence identifies one visible metadata field matched by search. */
export type SearchMatchEvidence = {
  /** field names the matched title, original filename, or attached tag. */
  field: "title" | "filename" | "tag";
  /** value preserves the user-facing spelling of the matched metadata. */
  value: string;
};

/** DocumentSearchResult is a ranked document with visible match evidence. */
export type DocumentSearchResult = DocumentRecord & {
  /** evidence explains why a non-empty query matched the document. */
  evidence: SearchMatchEvidence[];
};

/** Tag is a reusable or vault-managed date tag attached to documents. */
export type Tag = {
  /** id uniquely identifies the tag. */
  id: string;
  /** displayName contains the current user-facing spelling. */
  displayName: string;
  /** normalizedName is the case-insensitive identity used by filters. */
  normalizedName: string;
  /** implicit marks a date tag managed through its document metadata. */
  implicit?: boolean;
};

/** StorageUsage is the host storage account capacity reported by the backend. */
export type StorageUsage = {
  /** usedBytes is the account's current storage consumption. */
  usedBytes: number;
  /** limitBytes is null when the provider grants unlimited storage. */
  limitBytes: number | null;
  /** remainingBytes is null when the account has no reported storage limit. */
  remainingBytes: number | null;
};

/** ApiError carries typed failure details for HTTP and network errors. */
export class ApiError extends Error {
  /** status is the HTTP status, or 0 when the browser could not send the request. */
  readonly status: number;
  /** code is the backend error code or a client-side boundary code. */
  readonly code: ApiErrorCode;
  /** body is the parsed response body when one was available. */
  readonly body: unknown;

  constructor(
    status: number,
    code: ApiErrorCode,
    message: string,
    body?: unknown,
  ) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.body = body;
  }
}

/** UploadResult is the typed success response returned after a multipart upload. */
export type UploadResult = ApiResponse<DocumentRecord>;

/** UploadState describes measured transfer and server-finalization phases. */
export type UploadState = {
  /** state describes the deterministic phase visible to callers. */
  state: "uploading" | "processing" | "complete" | "failed";
  /** loaded is the measured request bytes, or null when unavailable. */
  loaded: number | null;
  /** total is the browser-reported multipart request size when available. */
  total: number | null;
  /** percent is null for indeterminate upload progress. */
  percent: number | null;
};

const knownApiErrorCodes: ReadonlySet<string> = new Set<KnownApiErrorCode>([
  "administrator_required",
  "already_exists",
  "authentication_required",
  "backend_unavailable",
  "catalog_error",
  "csrf_required",
  "deletion_in_progress",
  "document_too_large",
  "file_required",
  "idempotency_key_required",
  "invalid_login",
  "invalid_range",
  "invalid_request",
  "invalid_upload",
  "login_unavailable",
  "logout_failed",
  "membership_disabled",
  "network_error",
  "not_found",
  "rate_limited",
  "retention_active",
  "storage_usage_unavailable",
  "unexpected_response",
  "unsupported_media_type",
  "upload_in_progress",
  "vault_error",
  "version_conflict",
]);

let csrfToken: string | null = null;

const apiBase = "/api/v1";
const emptyResponse = 204;

/** api is the typed browser boundary for the current /api/v1 backend contract. */
export const api = {
  /** getSession loads the session and supports bounded startup cancellation. */
  async getSession(signal?: AbortSignal): Promise<ApiResponse<Session>> {
    csrfToken = null;
    const response = await request<Session>("/session", { signal });
    if (
      typeof response.data.csrfToken === "string" &&
      response.data.csrfToken !== ""
    ) {
      csrfToken = response.data.csrfToken;
    }
    return response;
  },

  /** logout invalidates the current session and clears the stored CSRF token. */
  async logout(): Promise<ApiResponse<null>> {
    try {
      return await request<null>("/logout", { method: "POST" });
    } finally {
      csrfToken = null;
    }
  },

  /** getStorageUsage returns capacity for the configured host storage account. */
  getStorageUsage(): Promise<ApiResponse<StorageUsage>> {
    return request("/storage/usage");
  },

  /** listDocuments returns documents filtered by supported status and tag options. */
  listDocuments(options?: {
    tags?: string[];
    match?: "all" | "any";
    status?: "ready" | "deleted";
  }): Promise<ApiResponse<{ documents: DocumentRecord[] }>> {
    const params = new URLSearchParams();
    for (const tag of options?.tags ?? []) {
      const trimmedTag = tag.trim();
      if (trimmedTag !== "") params.append("tag", trimmedTag);
    }
    if (params.has("tag") && options?.match) params.set("match", options.match);
    if (options?.status) params.set("status", options.status);
    return request(`/documents${querySuffix(params)}`);
  },

  /** searchDocuments returns recent or fuzzy-matched ready documents. */
  searchDocuments(
    query: string,
    limit = 10,
    signal?: AbortSignal,
  ): Promise<ApiResponse<{ results: DocumentSearchResult[] }>> {
    return this.searchDocumentsWithFilters(query, {}, limit, signal);
  },

  /** searchDocumentsWithFilters combines fuzzy search with selected tags. */
  searchDocumentsWithFilters(
    query: string,
    filters: SearchFilters,
    limit = 10,
    signal?: AbortSignal,
  ): Promise<ApiResponse<{ results: DocumentSearchResult[] }>> {
    const params = new URLSearchParams({
      q: query,
      limit: String(limit),
    });
    for (const tag of filters.tags ?? []) {
      const trimmedTag = tag.trim();
      if (trimmedTag !== "") params.append("tag", trimmedTag);
    }
    if (params.has("tag") && filters.match) params.set("match", filters.match);
    return request(`/search${querySuffix(params)}`, { signal });
  },

  /** getDocument loads one ready document record by id. */
  getDocument(id: string): Promise<ApiResponse<DocumentRecord>> {
    return request(`/documents/${encodeURIComponent(id)}`);
  },

  /** updateMetadata replaces title, tags, and date using optimistic concurrency. */
  updateMetadata(
    id: string,
    payload: {
      title: string;
      tags: string[];
      date?: string;
      version: number;
    },
  ): Promise<ApiResponse<DocumentRecord>> {
    return request(`/documents/${encodeURIComponent(id)}`, {
      method: "PATCH",
      body: JSON.stringify(payload),
      headers: { "Content-Type": "application/json" },
    });
  },

  /** deleteDocument queues storage trashing and returns the current server record. */
  deleteDocument(id: string): Promise<ApiResponse<DocumentRecord>> {
    return request(`/documents/${encodeURIComponent(id)}`, {
      method: "DELETE",
    });
  },

  /** restoreDocument returns a deleted document to the ready lifecycle state. */
  restoreDocument(id: string): Promise<ApiResponse<DocumentRecord>> {
    return request(`/documents/${encodeURIComponent(id)}/restore`, {
      method: "POST",
    });
  },

  /** purgeDocument permanently deletes a retained document when backend policy allows it. */
  purgeDocument(id: string): Promise<ApiResponse<null>> {
    return request(`/documents/${encodeURIComponent(id)}/purge`, {
      method: "DELETE",
    });
  },

  /** getDocumentContentUrl builds a same-origin URL for inline or download content. */
  getDocumentContentUrl(id: string, options?: { download?: boolean }): string {
    const params = new URLSearchParams();
    if (options?.download === true) params.set("download", "true");
    return `${apiBase}/documents/${encodeURIComponent(id)}/content${querySuffix(params)}`;
  },

  /** uploadDocument creates a document from multipart form data. */
  async uploadDocument(
    file: File,
    metadata: {
      title?: string;
      tags?: string[];
      onProgress?: (state: UploadState) => void;
    },
    idempotencyKey: string,
  ): Promise<UploadResult> {
    const body = new FormData();
    body.append("file", file);
    if (metadata.title !== undefined) body.append("title", metadata.title);
    for (const tag of metadata.tags ?? []) body.append("tags", tag);

    try {
      const response = await uploadRequest(
        body,
        metadata.onProgress,
        idempotencyKey,
      );
      metadata.onProgress?.({
        state: "complete",
        loaded: file.size,
        total: file.size,
        percent: 100,
      });
      return response;
    } catch (error) {
      metadata.onProgress?.({
        state: "failed",
        loaded: null,
        total: file.size,
        percent: null,
      });
      throw error;
    }
  },

  /** listTags returns reusable tags for autocomplete. */
  listTags(
    query?: string,
    limit?: number,
  ): Promise<ApiResponse<{ tags: Tag[] }>> {
    const params = new URLSearchParams();
    if (query !== undefined) params.set("q", query);
    if (limit !== undefined) params.set("limit", String(limit));
    return request(`/tags${querySuffix(params)}`);
  },

  /** createTag creates or returns a reusable tag. */
  createTag(name: string): Promise<ApiResponse<Tag>> {
    return request("/tags", {
      method: "POST",
      body: JSON.stringify({ name }),
      headers: { "Content-Type": "application/json" },
    });
  },

  /** updateTag renames a reusable tag without changing its identity. */
  updateTag(id: string, name: string): Promise<ApiResponse<Tag>> {
    return request(`/tags/${encodeURIComponent(id)}`, {
      method: "PATCH",
      body: JSON.stringify({ name }),
      headers: { "Content-Type": "application/json" },
    });
  },

  /** deleteTag removes a reusable tag from the vault and attached documents. */
  deleteTag(id: string): Promise<ApiResponse<null>> {
    return request(`/tags/${encodeURIComponent(id)}`, {
      method: "DELETE",
    });
  },

  /** listMembers returns all allowlisted members for an administrator. */
  listMembers(): Promise<ApiResponse<{ members: Member[] }>> {
    return request("/admin/members");
  },

  /** createMember creates an active allowlisted member. */
  createMember(
    email: string,
    role: "member" | "admin",
  ): Promise<ApiResponse<Member>> {
    return request("/admin/members", {
      method: "POST",
      body: JSON.stringify({ email, role }),
      headers: { "Content-Type": "application/json" },
    });
  },

  /** updateMember changes a member's role and active or disabled status. */
  updateMember(
    id: string,
    role: "member" | "admin",
    status: "active" | "disabled",
  ): Promise<ApiResponse<Member>> {
    return request(`/admin/members/${encodeURIComponent(id)}`, {
      method: "PATCH",
      body: JSON.stringify({ role, status }),
      headers: { "Content-Type": "application/json" },
    });
  },
} as const;

async function request<T>(
  path: string,
  init: RequestInit = {},
): Promise<ApiResponse<T>> {
  const headers = new Headers(init.headers);
  const method = (init.method ?? "GET").toUpperCase();
  if (isMutatingMethod(method) && csrfToken)
    headers.set("X-CSRF-Token", csrfToken);

  let response: Response;
  try {
    response = await fetch(`${apiBase}${path}`, {
      ...init,
      method,
      credentials: "same-origin",
      headers,
    });
  } catch (error) {
    const message =
      error instanceof Error ? error.message : "The request could not be sent.";
    throw new ApiError(0, "network_error", message);
  }

  const body = await parseBody(response);
  if (!response.ok) {
    throw apiErrorFromResponse(response, body);
  }

  return {
    status: response.status,
    headers: response.headers,
    data: response.status === emptyResponse ? (null as T) : (body as T),
  };
}

function uploadRequest(
  body: FormData,
  onProgress: ((state: UploadState) => void) | undefined,
  idempotencyKey: string,
): Promise<UploadResult> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open("POST", `${apiBase}/documents`);
    xhr.withCredentials = true;
    xhr.setRequestHeader("Idempotency-Key", idempotencyKey);
    if (csrfToken) xhr.setRequestHeader("X-CSRF-Token", csrfToken);

    onProgress?.({ state: "uploading", loaded: 0, total: null, percent: null });
    xhr.upload.onprogress = (event) => {
      const measurable = event.lengthComputable && event.total > 0;
      onProgress?.({
        state: "uploading",
        loaded: event.loaded,
        total: measurable ? event.total : null,
        percent: measurable
          ? Math.min(100, Math.round((event.loaded / event.total) * 100))
          : null,
      });
    };
    xhr.upload.onload = () => {
      onProgress?.({
        state: "processing",
        loaded: null,
        total: null,
        percent: 100,
      });
    };
    xhr.onerror = () =>
      reject(
        new ApiError(0, "network_error", "The request could not be sent."),
      );
    xhr.onabort = () =>
      reject(new ApiError(0, "network_error", "The upload was canceled."));
    xhr.onload = () => {
      const headers = responseHeaders(xhr.getAllResponseHeaders());
      const response = new Response(xhr.responseText, {
        status: xhr.status,
        headers,
      });
      void parseBody(response)
        .then((parsed) => {
          if (xhr.status < 200 || xhr.status >= 300) {
            reject(apiErrorFromResponse(response, parsed));
            return;
          }
          resolve({
            status: xhr.status,
            headers,
            data: parsed as DocumentRecord,
          });
        })
        .catch(reject);
    };
    xhr.send(body);
  });
}

function responseHeaders(rawHeaders: string): Headers {
  const headers = new Headers();
  for (const line of rawHeaders.trim().split(/[\r\n]+/)) {
    if (line === "") continue;
    const separator = line.indexOf(":");
    if (separator > 0)
      headers.append(
        line.slice(0, separator),
        line.slice(separator + 1).trim(),
      );
  }
  return headers;
}

async function parseBody(response: Response): Promise<unknown> {
  const text = await response.text();
  if (text === "") return null;
  try {
    return JSON.parse(text) as unknown;
  } catch {
    throw new ApiError(
      response.status,
      "unexpected_response",
      "The server returned a response that was not valid JSON.",
      text,
    );
  }
}

function apiErrorFromResponse(response: Response, body: unknown): ApiError {
  const code = errorCodeFromBody(body);
  return new ApiError(
    response.status,
    code,
    errorMessage(response.status, code),
    body,
  );
}

function errorCodeFromBody(body: unknown): ApiErrorCode {
  if (
    typeof body === "object" &&
    body !== null &&
    "error" in body &&
    typeof body.error === "string" &&
    body.error !== ""
  ) {
    if (isKnownApiErrorCode(body.error)) return body.error;
    return { kind: "unknown_backend_code", code: body.error };
  }
  return "unexpected_response";
}

function errorMessage(status: number, code: ApiErrorCode): string {
  if (status === 0) return "The request could not be sent.";
  return `API request failed with ${status} (${formatErrorCode(code)}).`;
}

function isKnownApiErrorCode(code: string): code is KnownApiErrorCode {
  return knownApiErrorCodes.has(code);
}

function formatErrorCode(code: ApiErrorCode): string {
  return typeof code === "string" ? code : code.code;
}

function isMutatingMethod(method: string): boolean {
  return method !== "GET" && method !== "HEAD" && method !== "OPTIONS";
}

function querySuffix(params: URLSearchParams): string {
  const query = params.toString();
  return query === "" ? "" : `?${query}`;
}
