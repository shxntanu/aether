# Public pilot boundary

This document defines the public surface for the HTTPS pilot. The landing page
is intentionally a self-contained public route. It does not identify a family,
display vault contents, or claim that the product is available.

## Public routes

The following routes are reachable without an Aether session:

| Route | Purpose | Boundary |
| --- | --- | --- |
| `/` | Public constellation landing page | Uses only authored synthetic nodes. It performs no API fetch and displays no vault or member state. |
| `/api/v1/health` | Process health check | Returns health status only; it does not authenticate or reveal vault data. |
| `GET /auth/google/start` | Start Google login | Starts the protected OAuth flow and redirects to Google. OAuth state, nonce, and PKCE protect the callback. |
| `GET /auth/google/callback` | Complete Google login | Validates the OAuth flow and verified Google identity, then creates a session only for an active allowlisted member. |

The landing page makes no analytics or third-party requests. Its constellation
nodes are authored synthetic visual elements, and its interaction supports
keyboard focus/activation and pointer/touch input. Reduced-motion preferences
disable the custom animation loop and make selection transitions instantaneous.

## Private routes and authorization

Private API routes require the opaque `aether_session` cookie. The server
re-checks the current member and status on every request. For every state-
changing API request, the client must also send the synchronizer token in
`X-CSRF-Token`; the cookie, header, and server-side session digest must agree.
Safe `GET` requests do not require the CSRF header.

| Operation | Routes | Requirement |
| --- | --- | --- |
| Session | `GET /api/v1/session` | Authenticated active member; no CSRF header. |
| Logout | `POST /api/v1/logout` | Authenticated member and valid CSRF token; invalidates the server-side session and expires cookies. |
| Documents | `POST /api/v1/documents`, `GET /api/v1/documents`, `GET /api/v1/documents/{id}`, `PATCH /api/v1/documents/{id}`, `GET /api/v1/documents/{id}/content` | Authenticated active member. `POST`/`PATCH` require CSRF; `GET` operations do not. Uploads retain the existing 50 MiB and supported-format constraints. |
| Soft delete | `DELETE /api/v1/documents/{id}` | Authenticated active member and valid CSRF token. Idempotently trashes the original and manifest and starts the 30-day retention period. |
| Restore | `POST /api/v1/documents/{id}/restore` | Authenticated administrator and valid CSRF token. Restores both stored objects before returning the document to `ready`. |
| Purge | `DELETE /api/v1/documents/{id}/purge` | Authenticated administrator and valid CSRF token. Irreversibly removes a document only after its 30-day retention period. |
| Tags | `GET /api/v1/tags` | Authenticated active member; no CSRF header. |
| Create tag | `POST /api/v1/tags` | Authenticated active member and valid CSRF token. |
| Member administration | `GET /api/v1/admin/members`, `POST /api/v1/admin/members`, `PATCH /api/v1/admin/members/{id}` | Authenticated administrator. `POST` and `PATCH` require a valid CSRF token; the list `GET` does not. |

Google Drive object links are never returned as part of the public landing
surface. Credentials, OAuth secrets, session tokens, CSRF tokens, and raw
provider credentials are never exposed in page content, URLs, logs, or audit
records.

## Pilot operational limitations

- Rate limits are in-memory and per process: login start/callback is limited to
  10 requests per 10 minutes per IP; authenticated API traffic is limited to
  120 requests per minute per member and 60 per minute per IP. A multi-process
  deployment does not share these counters.
- Purge is an in-process, cancellable hourly loop that scans and attempts due
  documents in bounded batches of 100. It is temporary pilot maintenance;
  durable background jobs are planned for Checkpoint 8.
- Audit records intentionally omit document content, filenames, titles, tag
  values, request bodies, query strings, session tokens, and raw IP addresses.
  Audit failures are not a reason to disclose those values for diagnostics.
- Sessions created before the CSRF migration have no usable CSRF digest and
  must sign in again before unsafe requests can succeed.
- The security regression suite is still missing. Pilot acceptance is blocked
  until the later test checkpoint exercises these observable security
  contracts and all repository checks pass; this document does not declare
  Checkpoint 6 accepted.

## Rollback and incident boundary

The public route can be withdrawn at the edge or by stopping the application
without changing stored documents. Do not roll back the CSRF migration by
removing its column: old sessions are intentionally invalidated for unsafe
requests. If the pilot is paused, preserve the database and storage volume,
then resume after the security regression suite and operational review are
complete.
