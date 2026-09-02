# Aether Family Document Vault Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` or `superpowers:executing-plans` to implement this plan task-by-task. The project owner intends to implement each checkpoint personally and request review before advancing.

**Goal:** Build a private family document vault that stores original files in Google Drive and progressively adds reusable tags, content indexing, OCR, and evidence-backed natural-language retrieval.

**Architecture:** Aether is a modular Go monolith with a React/TypeScript responsive web client. Google Drive is the reference object store, PostgreSQL holds the live catalog in local and hosted environments, and versioned manifests make stored documents portable and recoverable.

**Tech Stack:** Go 1.27, React with TypeScript and Vite, PostgreSQL with pgvector, Google OAuth/OIDC, Google Drive API, OCRmyPDF/Tesseract, Docker Compose, and Caddy.

**Spec:** This file incorporates the approved product design and staged implementation roadmap.

## Global Constraints

- Keep backend and frontend dependency ecosystems under `backend/` and `frontend/`.
- Use Google Drive through a provider-neutral object-storage interface.
- Use one designated Google account and an app-managed Drive folder; do not support direct Drive edits or bidirectional synchronization.
- Store PDF, JPEG, PNG, and WebP files up to 50 MB without modifying the original bytes.
- Use Google login and an administrator-managed allowlist for one shared family library.
- Make tags optional, reusable, and case-insensitive while retaining a display spelling.
- Treat printed English as the pilot OCR language.
- Default OCR and model processing to local providers; require explicit configuration for cloud providers.
- Keep document storage within existing Google Drive capacity and do not attach a billing account for Drive API overage.
- Develop every behavioral change test-first and review each checkpoint before starting the next.

---

## Repository Structure

```text
aether/
├── backend/
│   ├── cmd/aether/            # Go process entrypoint
│   ├── internal/              # Private domain and adapter packages
│   ├── migrations/            # PostgreSQL schema migrations
│   ├── go.mod
│   └── go.sum
├── frontend/
│   ├── src/                   # React application
│   ├── public/                # Static public assets
│   ├── package.json
│   └── pnpm-lock.yaml
├── contracts/
│   └── openapi.yaml           # Browser/backend HTTP contract
├── deploy/
│   ├── Dockerfile
│   └── compose.yaml
├── docs/                      # Setup, operations, and architecture notes
├── plans/                     # Approved implementation plans
├── Makefile                   # Repository-level development commands
├── README.md
└── .gitignore
```

During development, Vite runs on port 5173 and proxies `/api` and `/auth` to Go on port 8080. The production container builds the frontend and packages its static output with the Go service, preserving a single deployable application container.

## Stable Interfaces and Data Model

### Object storage

The domain depends on an `ObjectStore` interface with operations equivalent to:

```go
type ObjectStore interface {
	Put(ctx context.Context, key string, body io.Reader, options PutOptions) (ObjectInfo, error)
	OpenRange(ctx context.Context, key string, byteRange *ByteRange) (io.ReadCloser, ObjectInfo, error)
	Stat(ctx context.Context, key string) (ObjectInfo, error)
	Trash(ctx context.Context, key string) error
	Restore(ctx context.Context, key string) error
	Delete(ctx context.Context, key string) error
}
```

Implement local-filesystem and Google Drive adapters. Browsers never receive provider credentials or public Drive URLs.

### Document lifecycle

- Document states: `uploading`, `ready`, `failed`, `deleted`.
- Index states: `not_scheduled`, `queued`, `extracting`, `enriching`, `indexed`, `failed`.
- Store UUID, title, original filename, detected MIME type, size, SHA-256 checksum, storage key, uploader, timestamps, deletion deadline, and integer version.
- Keep document content immutable. Metadata edits use the integer version for optimistic concurrency.
- Store tags separately with a normalized value and display value; connect them through a document-tag join table.
- Record membership, sessions, storage records, background jobs, page text, content chunks, suggestions, and append-only audit events in the catalog.

### HTTP API

- `GET /api/v1/health`
- `GET /auth/google/start` and `GET /auth/google/callback`
- `POST /api/v1/logout` and `GET /api/v1/session`
- `GET/POST /api/v1/documents`
- `GET/PATCH/DELETE /api/v1/documents/{id}`
- `GET /api/v1/documents/{id}/content`
- `POST /api/v1/documents/{id}/restore`
- `DELETE /api/v1/documents/{id}/purge`
- `GET/POST /api/v1/tags`
- `GET /api/v1/search`
- `GET/POST/PATCH /api/v1/admin/members`
- `GET /api/v1/admin/jobs` and `POST /api/v1/admin/jobs/{id}/retry` when processing is introduced.

Uploads use multipart streaming, `Idempotency-Key`, signature-based MIME validation, and SHA-256 calculation. Metadata responses include an ETag/version. Search responses contain ranked documents and typed evidence such as matched tags, fields, or page snippets.

## Implementation Checkpoints

### 1. Monorepo and Tested Go Service

- [x] Move the Go module to `backend/` using module path `github.com/shxntanu/aether/backend`.
- [x] Write a failing handler test for `GET /api/v1/health`.
- [x] Implement the minimal router and executable required to pass the test.
- [x] Add graceful shutdown and configuration loading through tested functions.
- [x] Scaffold the Vite React/TypeScript application under `frontend/`.
- [x] Configure the Vite development proxy and prove the browser can read the Go health endpoint.
- [x] Add root `Makefile` commands for formatting, testing, and running both applications.

Acceptance: backend and frontend start independently, the health test passes, and the browser reaches the API through the development proxy.

### 2. Catalog and Domain Foundations

- [x] Define document, tag, member, and audit domain types without database-specific fields.
- [x] Add explicit repository interfaces and PostgreSQL migrations.
- [x] Implement repository contract tests against containerized PostgreSQL.
- [x] Add lifecycle-transition and optimistic-concurrency tests before implementing document persistence.
- [x] Add PostgreSQL configuration profiles for local containers and hosted providers.

Acceptance: the domain and repository contract tests pass against containerized PostgreSQL, including conflict and rollback cases.

### 3. Identity and Membership

- [x] Implement Google OIDC using state, nonce, PKCE, exact redirect URIs, and secure server-side sessions.
- [x] Seed the first administrator from a configured bootstrap email.
- [x] Implement active/disabled membership and `member`/`admin` roles.
- [ ] Add administrator member-management endpoints and UI.
- [x] Enforce authorization in Go and test unauthenticated, disabled, member, and administrator cases.

Acceptance: only allowlisted active accounts enter the vault, and only administrators can manage membership.

### 4. Local Tagged Vault Vertical Slice

- [ ] Implement the local-filesystem `ObjectStore` adapter and its contract tests.
- [x] Stream uploads through Go while enforcing type/size limits and calculating SHA-256.
- [x] Derive the initial title from the filename and accept optional reusable tags.
- [x] Implement listing, tag autocomplete, all/any tag filtering, metadata edits, preview, and range downloads.
- [ ] Build the corresponding responsive React screens and browser tests.
- [x] Add inline versioned manifests and surface manifest-write failures for repair.

Acceptance: an allowlisted user can upload an optionally tagged document, retrieve it by tags, edit its metadata, preview it, and download identical bytes.

### 5. Google Drive Storage

- [x] Create a separate vault-owner OAuth authorization flow using only the `drive.file` scope.
- [x] Store the refresh token as a deployment secret, never in browser storage or logs.
- [x] Implement the Drive adapter with resumable uploads, range downloads, metadata lookup, trash, restore, and permanent deletion.
- [ ] Run the shared object-store contract suite against a fake Drive server.
- [ ] Provide an opt-in integration test and setup guide for a real app-managed Drive folder.
- [ ] Re-run the tagged-vault acceptance flow with Drive selected through configuration only.

Acceptance: switching from local storage to Drive requires configuration changes but no domain or HTTP changes.

### 6. Deletion, Audit, and Public Hardening

- [x] Add 30-day soft deletion, administrator restore, and eventual purge.
- [x] Audit uploads, edits, downloads, deletion, restoration, membership changes, and rejected authorization without recording document text or query text.
- [x] Add CSRF protection, account/IP rate limits, request timeouts, `nosniff`, restrictive content policies, and safe content disposition.
- [ ] Add a public landing page that never exposes family data.
- [ ] Test interrupted uploads, duplicate idempotency keys, stale edits, failed manifest writes, and deletion recovery.

Acceptance: the tagged Drive vault is suitable for a public HTTPS pilot behind authentication.

### 7. Deployment and Recovery

- [ ] Create a multi-stage image that builds React, compiles Go, and produces one application image.
- [ ] Add Caddy, PostgreSQL with pgvector, web, and worker services to `deploy/compose.yaml`.
- [ ] Document Supabase Postgres as an alternative connection string, not an application dependency.
- [ ] Add database migrations as an explicit pre-deployment command.
- [ ] Encrypt nightly catalog backups before writing them to Drive.
- [ ] Add health reporting for process, database, Drive connectivity, and maintenance failures.
- [ ] Perform and document a restore drill from a database backup and stored manifests.

Acceptance: a fresh VPS can deploy the tagged vault through documented commands and recover it from backups.

### 8. Digital-PDF Text Indexing

- [ ] Add a database-backed job/outbox system with leasing, idempotency, retries, and dead-letter state.
- [ ] Run web and worker roles from the same binary, with combined mode available locally.
- [ ] Extract embedded page text from text-native PDFs without modifying originals.
- [ ] Store page provenance, chunks, extractor name, and extractor version.
- [ ] Add PostgreSQL full-text and trigram search.
- [ ] Return highlighted page snippets and provide administrator retry controls.
- [ ] Move manifest synchronization, reconciliation, cleanup, and purge onto background jobs.

Acceptance: phrases, names, and reference numbers inside digital PDFs are searchable, and failed work can be retried without duplication.

### 9. Printed-English OCR

- [ ] Implement an `Extractor` adapter around OCRmyPDF/Tesseract.
- [ ] OCR scan-only PDF pages and JPEG/PNG/WebP documents with the English language pack.
- [ ] Store extraction method and confidence by page and visibly mark low-confidence results.
- [ ] Apply subprocess memory/time limits and handle corrupt, encrypted, or oversized documents.
- [ ] Add administrator reprocessing and extractor-version migration controls.

Acceptance: representative printed-English scans become searchable, while unreadable pages and failures remain visible.

### 10. Hybrid Natural-Language Retrieval

- [ ] Add an `EmbeddingProvider` with local Ollama and explicitly configured cloud-compatible adapters.
- [ ] Chunk text by page and paragraph with overlap while preserving provenance.
- [ ] Store model name, vector dimension, and version with every embedding.
- [ ] Use pgvector for vector storage and cosine ranking.
- [ ] Fuse metadata/full-text, fuzzy, and semantic candidates using reciprocal-rank fusion.
- [ ] Add an `EnrichmentProvider` that suggests title, type, issuer, people, dates, reference numbers, and tags without overwriting confirmed metadata.
- [ ] Return “matched because” evidence rather than generated factual answers.
- [ ] Build a versioned fixture with at least 50 representative documents and 30 natural-language queries.

Acceptance: the expected document appears in the top five for at least 85% of benchmark queries, and exact identifiers rank first.

## Verification Strategy

- Use Go unit tests for permissions, lifecycle transitions, tag normalization, retries, and ranking fusion.
- Run repository contracts against containerized PostgreSQL.
- Run object-store contracts against local storage and a fake Drive HTTP server; keep real Drive tests opt-in.
- Use React Testing Library for components and Playwright for login, upload, tag search, preview, editing, deletion, restoration, and administration flows.
- Inject failures around storage/database boundaries, worker crashes, duplicated jobs, OAuth rejection, stale edits, and reconciliation.
- Keep the relevance fixture versioned and run its metrics whenever extraction, embedding, enrichment, or ranking changes.
- Require `go test ./...`, frontend tests, production builds, and applicable integration tests to pass at every checkpoint.

## Deferred Work

- Handwriting recognition and Indian-language OCR.
- In-app camera scanning and image correction.
- Public or short document links.
- Generated answers and answer-with-citations/RAG.
- Per-document visibility and sharing.
- Bidirectional Google Drive synchronization.
- End-to-end encryption, which conflicts with server-side content indexing and search.
- A SQLite catalog adapter; add it only if a concrete offline deployment requires it.
