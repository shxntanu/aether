# Local tagged vault

Checkpoint 4 stores immutable document originals through the provider-neutral
object-store interface. The `local` provider keeps objects beneath a private
filesystem root; later providers such as Google Drive live in their own package
and implement the same interface.

## Configuration

Configure PostgreSQL and Google identity as described in `identity.md`, then
select local storage:

```dotenv
AETHER_STORAGE_PROVIDER=local
AETHER_LOCAL_STORAGE_PATH=./data/vault
```

The process creates the storage root with owner-only permissions. Production
deployments must place it on a persistent volume and exclude it from backups
that do not meet the vault's privacy requirements.

## API

- `POST /api/v1/documents` requires an `Idempotency-Key` header and accepts
  multipart fields `file`, optional `title`, and repeated or comma-separated
  `tags` values. Reusing the key as the same member returns the first document.
- `GET /api/v1/documents?tag=tax&tag=2026&match=all` lists ready documents.
  Set `match=any` to match at least one tag, or set `status=deleted` to load
  recoverable Trash records and their asynchronous deletion status.
- `GET /api/v1/documents/{id}` returns metadata and tags.
- `PATCH /api/v1/documents/{id}` accepts `title`, `tags`, and `version`.
- `DELETE /api/v1/documents/{id}` returns `202 Accepted` with the complete
  updated document record. The catalog transition is immediate; a persisted
  background worker moves the original and manifest into provider trash and
  updates `document.deletionStatus` from `queued` to `processing`, `complete`,
  or retryable `failed`.
- `GET /api/v1/documents/{id}/content` previews content and supports one
  explicit `Range: bytes=start-end` range. Add `?download=true` for attachment
  disposition. With Google Drive storage and no range, the route redirects to
  Drive's viewer or original-content link; local storage streams through Aether.
- `GET /api/v1/tags?q=tax&limit=20` provides tag autocomplete.
- `POST /api/v1/tags` accepts `{"name":"Tax"}`.

Uploads accept PDF, JPEG, PNG, and WebP signatures up to 50 MiB. The catalog
stores a SHA-256 digest, and every successful metadata state is written as a
versioned JSON manifest next to the original object. A failed manifest write is
returned as `document.manifestError`; the original remains retrievable and can
be repaired by the maintenance workflow introduced in a later checkpoint.

Migration `0006_async_deletions.sql` marks existing deleted rows complete and
adds only backward-compatible columns. Rolling application code back is safe
because older versions ignore those columns; apply the migration before
deploying code that starts the deletion worker.
