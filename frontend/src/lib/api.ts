type MemberRole = "member" | "admin";
type MemberStatus = "active" | "disabled";
type DocumentStatus = "uploading" | "ready" | "failed" | "deleted";
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
  | "catalog_error"
  | "csrf_required"
  | "document_too_large"
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
  /** manifestError reports a failed manifest synchronization for repair. */
  manifestError?: string;
};

/** DocumentRecord combines catalog metadata with its reusable tags. */
export type DocumentRecord = {
  /** document contains browser-visible catalog metadata for the immutable original. */
  document: Document;
  /** tags contains the reusable tags currently attached to the document. */
  tags: Tag[];
};

/** Tag is the reusable tag record attached to documents. */
export type Tag = {
  /** id uniquely identifies the reusable tag. */
  id: string;
  /** displayName preserves the first accepted user-facing spelling. */
  displayName: string;
  /** normalizedName is the case-insensitive identity used by filters. */
  normalizedName: string;
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

/** UploadState describes deterministic upload phases exposed by fetch-based uploads. */
export type UploadState = {
  /** state describes the deterministic phase visible to callers. */
  state: "uploading" | "complete" | "failed";
  /** loaded is null while the browser transfer amount is unavailable. */
  loaded: number | null;
  /** total is the selected file size when available. */
  total: number | null;
  /** percent is null for indeterminate upload progress. */
  percent: number | null;
};

const knownApiErrorCodes: ReadonlySet<string> = new Set<KnownApiErrorCode>([
  "administrator_required",
  "already_exists",
  "authentication_required",
  "catalog_error",
  "csrf_required",
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
  /** getSession loads the authenticated session and captures its CSRF token. */
  async getSession(): Promise<ApiResponse<Session>> {
    csrfToken = null;
    const response = await request<Session>("/session");
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

  /** listDocuments returns ready documents filtered by supported tag options. */
  listDocuments(options?: {
    tags?: string[];
    match?: "all" | "any";
  }): Promise<ApiResponse<{ documents: DocumentRecord[] }>> {
    const params = new URLSearchParams();
    for (const tag of options?.tags ?? []) {
      const trimmedTag = tag.trim();
      if (trimmedTag !== "") params.append("tag", trimmedTag);
    }
    if (params.has("tag") && options?.match) params.set("match", options.match);
    return request(`/documents${querySuffix(params)}`);
  },

  /** getDocument loads one ready document record by id. */
  getDocument(id: string): Promise<ApiResponse<DocumentRecord>> {
    return request(`/documents/${encodeURIComponent(id)}`);
  },

  /** updateMetadata replaces document title and tags using optimistic concurrency. */
  updateMetadata(
    id: string,
    payload: { title: string; tags: string[]; version: number },
  ): Promise<ApiResponse<DocumentRecord>> {
    return request(`/documents/${encodeURIComponent(id)}`, {
      method: "PATCH",
      body: JSON.stringify(payload),
      headers: { "Content-Type": "application/json" },
    });
  },

  /** deleteDocument soft-deletes one document and preserves the empty response status. */
  deleteDocument(id: string): Promise<ApiResponse<null>> {
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

    metadata.onProgress?.({
      state: "uploading",
      loaded: null,
      total: file.size,
      percent: null,
    });
    try {
      const response = await request<DocumentRecord>("/documents", {
        method: "POST",
        headers: { "Idempotency-Key": idempotencyKey },
        body,
      });
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
