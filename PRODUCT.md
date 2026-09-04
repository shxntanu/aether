# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Stack

Go modular monolith backend with a React, TypeScript, and Vite web client; PostgreSQL catalog; Google Drive object storage; Google OAuth/OIDC.

## Users

Primary users are members of a small team or organization who need a private shared library for important documents. An administrator manages the allowlist and membership; active members retrieve and work with the shared collection.

The vault is not family-specific, but the same private-library model can support a family or household.

## Product Purpose

Aether is a private document vault for storing original files, organizing them with reusable tags, and making them easier to retrieve over time. Its success is that an allowlisted group can safely preserve important documents and find the right source file again without exposing the library publicly.

## Positioning

Aether combines private group access, durable original-file storage, reusable metadata, and progressively richer indexing into one vault. Search is evidence-backed: results should identify matched tags, metadata fields, or source-page snippets rather than fabricate answers. The vault can support teams or organizations without requiring direct edits in the underlying storage provider.

## Operating Context

Administrators maintain membership and the shared library. Members upload, tag, browse, search, preview, edit metadata, and download documents through the web application. Google Drive is managed as the vault's object store; authenticated view and download requests redirect to Drive without exposing provider credentials. Processing may add extracted text, OCR, and suggestions while preserving the original bytes.

## Capabilities and Constraints

Confirmed capabilities and roadmap:

- Google login with secure server-side sessions, an administrator-managed allowlist, and member or administrator roles.
- A shared library with document upload, reusable optional tags, metadata editing, preview, download, and search.
- Original PDF, JPEG, PNG, and WebP files up to 50 MB are stored without modifying their bytes.
- The storage layer remains provider-neutral; Google Drive is the planned reference object store and direct Drive edits or bidirectional synchronization are not supported.
- PostgreSQL stores the catalog, metadata, membership, processing state, and audit records.
- Local providers are the default for OCR and model processing; cloud providers require explicit configuration.
- Printed English is the pilot OCR language.
- Uploads must stream through the server, validate file type and size at the trust boundary, calculate SHA-256 checksums, and support idempotency protection.
- Document content is immutable; metadata edits use optimistic concurrency. Tags are case-insensitive while retaining display spelling.
- Search results return ranked documents with typed evidence such as matched tags, fields, or page snippets. Generated factual answers are out of scope.
- Storage stays within existing Google Drive capacity; no Drive API overage billing account is required.

The implementation plan is staged. Several capabilities, including the complete document vault workflow, administration UI, local object storage, Drive integration, OCR, indexing, and deployment hardening, remain planned rather than currently available in the repository.

## Brand Commitments

The product name is Aether. No additional voice, visual identity, logo, or terminology commitments were confirmed during init.

## Evidence on Hand

- `plans/aether-implementation-plan.md` contains the approved product goal, architecture, constraints, staged roadmap, and verification strategy.
- `frontend/src/App.tsx` currently presents Aether as a private document-vault concept and checks `GET /api/v1/health`.
- `backend/internal/identity/` and `backend/internal/httpapi/` contain the implemented Google OIDC, session, membership, and health-route foundations.
- `backend/internal/catalog/` and `backend/migrations/` contain the catalog domain and PostgreSQL foundations.
- No production document fixtures, testimonials, customer proof, or public-facing brand assets were found or confirmed. Future work must not fabricate them.

## Product Principles

1. Keep original documents durable and recoverable.
2. Restrict access to explicitly allowlisted active members.
3. Make retrieval useful through reusable metadata and source-grounded evidence.
4. Preserve provider neutrality so storage can change without changing the product contract.
5. Prefer local processing and require explicit consent for cloud processing.
