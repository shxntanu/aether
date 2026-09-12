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

## Deployment configuration

The reverse proxy is expected to terminate HTTPS and forward requests from its
loopback address. A production profile should include the following values:

```dotenv
AETHER_PUBLIC_URL=https://vault.example
AETHER_TRUSTED_PROXY_CIDRS=127.0.0.1/32,::1/128
AETHER_REQUEST_TIMEOUT=30s
AETHER_UPLOAD_TIMEOUT=10m
```

Only the explicitly listed loopback Caddy proxy CIDRs may supply
`X-Forwarded-For`. The application compares unsafe-request origins to the
normalized `AETHER_PUBLIC_URL`; an origin mismatch or
`Sec-Fetch-Site: cross-site` request is rejected before CSRF token validation.

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
| Documents | `POST /api/v1/documents`, `GET /api/v1/documents`, `GET /api/v1/documents/{id}`, `PATCH /api/v1/documents/{id}`, `GET /api/v1/documents/{id}/content` | Authenticated active member. `POST`/`PATCH` require CSRF; `GET` operations do not. Uploads retain the existing 50 MiB and supported-format constraints; successful uploads receive an implicit IST date tag, and `PATCH` accepts an editable `YYYY-MM-DD` date. |
| Search | `GET /api/v1/search` | Authenticated active member; supports fuzzy `q` plus repeated exact `tag` filters and `match=all` or `match=any`. |
| Soft delete | `DELETE /api/v1/documents/{id}` | Authenticated active member and valid CSRF token. Idempotently trashes the original and manifest and starts the 30-day retention period. |
| Restore | `POST /api/v1/documents/{id}/restore` | Authenticated administrator and valid CSRF token. Restores both stored objects before returning the document to `ready`. |
| Purge | `DELETE /api/v1/documents/{id}/purge` | Authenticated administrator and valid CSRF token. Irreversibly removes a document only after its 30-day retention period. |
| Tags | `GET /api/v1/tags` | Authenticated active member; no CSRF header. |
| Create tag | `POST /api/v1/tags` | Authenticated active member and valid CSRF token. |
| Member administration | `GET /api/v1/admin/members`, `POST /api/v1/admin/members`, `PATCH /api/v1/admin/members/{id}` | Authenticated administrator. `POST` and `PATCH` require a valid CSRF token; the list `GET` does not. |

Unsafe API requests from a rejected origin or with a missing, stale, or
mismatched CSRF token return `403` with `{"error":"csrf_required"}`. OAuth
callbacks remain `GET` routes and use OAuth state, nonce, and PKCE instead of
CSRF token validation.

## Rate limits

Rate limiting is process-local and returns `429` with a JSON body of
`{"error":"rate_limited"}` and a `Retry-After` header containing the number of
seconds until retry. Login start and callback requests are limited to 10 per
10 minutes per client IP. Authenticated API requests are limited to 120 per
minute per member and 60 per minute per client IP. The health endpoint is not
limited by account, though it remains covered by the conservative IP limit.

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
- Administrators can manually restore or purge retained documents with the
  restore and purge endpoints listed above; purge rejects documents while the
  30-day retention period is active.
- The security regression suite is still missing. Pilot acceptance is blocked
  until the later test checkpoint exercises these observable security
  contracts and all repository checks pass; this document does not declare
  Checkpoint 6 accepted.

## Rollback and incident boundary

Application rollback is safe because migration `0004` is additive: deploy an
older application binary while preserving the database and storage volume.
Do not drop the `sessions_tbl.csrf_hash` column or its index until all older and
newer binaries have been retired. Old sessions are intentionally invalidated
for unsafe requests and users must sign in again after the hardened binary is
restored. The public route can also be withdrawn at the edge or by stopping
the application without changing stored documents.
