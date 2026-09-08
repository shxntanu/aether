# Render backend and Cloudflare frontend deployment

This profile keeps Aether as one Go service on Render, PostgreSQL on Supabase,
and immutable originals in the existing Google Drive vault. Cloudflare serves
the Vite build and proxies only `/api/*` and `/auth/*`, so browsers retain the
same-origin API, cookie, CSRF, upload, and download contracts.

No database rows or Drive objects are copied by this deployment.

## 1. Fix the public origin

Authenticate Wrangler and inspect the account before configuring Google or
Render:

```bash
cd frontend
pnpm exec wrangler login
pnpm exec wrangler whoami
```

The permanent origin is
`https://aether-vault.<workers-subdomain>.workers.dev`. Use that exact origin
for every value below. Changing it later requires updating Render and both
Google OAuth registrations.

Register these exact authorized redirect URIs in Google Cloud:

```text
https://aether-vault.<workers-subdomain>.workers.dev/auth/google/callback
https://aether-vault.<workers-subdomain>.workers.dev/auth/gdrive/callback
```

The Drive callback is retained as configuration compatibility for the existing
vault-owner credential. Normal deployments reuse the existing refresh token;
the loopback helper in `docs/google-drive-storage.md` remains the way to mint a
replacement token.

## 2. Verify and lock down Supabase

Use the Supabase SQL editor as a privileged project administrator. This query
shows whether Aether tables have RLS enabled and whether the Data API roles
hold table privileges:

```sql
SELECT
    c.relname AS table_name,
    c.relrowsecurity AS rls_enabled,
    has_table_privilege('anon', c.oid, 'SELECT,INSERT,UPDATE,DELETE')
        AS anon_has_dml,
    has_table_privilege('authenticated', c.oid, 'SELECT,INSERT,UPDATE,DELETE')
        AS authenticated_has_dml
FROM pg_class AS c
JOIN pg_namespace AS n ON n.oid = c.relnamespace
WHERE n.nspname = 'public'
  AND c.relkind = 'r'
  AND c.relname = ANY (ARRAY[
      'schema_migrations_tbl', 'members_tbl', 'documents_tbl', 'tags_tbl',
      'document_tags_tbl', 'audit_events_tbl', 'sessions_tbl',
      'auth_flows_tbl', 'upload_requests_tbl'
  ])
ORDER BY c.relname;
```

All `rls_enabled` values must be true and both privilege columns must be false.
Migration `0008_supabase_data_api_lockdown.sql` establishes that state without
creating browser policies. It preserves all rows. Aether must connect through
the Supabase session pooler on port `5432` as the privileged backend database
role; do not use an `anon` or `authenticated` JWT role.

The Render connection string must include TLS, for example:

```text
postgresql://<backend-role>:<password>@<project>.pooler.supabase.com:5432/postgres?sslmode=require
```

Keep this URL only in Render. No Supabase password or privileged key belongs in
Vite or Wrangler configuration.

## 3. Create the Render service

Create a Blueprint from the repository's `render.yaml`. Supply every value
marked `sync: false` in the Render dashboard. In particular:

- `AETHER_DATABASE_URL`: the TLS session-pooler URL described above.
- `AETHER_PUBLIC_URL`: the permanent Worker origin.
- `AETHER_GATEWAY_SECRET`: at least 32 random bytes, stored without quotes.
- `AETHER_STORAGE_PROVIDER`: already fixed to `gdrive` by the Blueprint.
- All existing Drive client, folder, and refresh-token values.
- All existing Google OIDC client values and bootstrap administrator email.
- `AETHER_GDRIVE_REDIRECT_URL` and `AETHER_GOOGLE_REDIRECT_URL`: the exact
  Worker callback URLs from step 1.

Generate the shared secret locally without printing or committing it anywhere
except the two platform secret prompts:

```bash
openssl rand -base64 48
```

Render starts Aether with normal embedded migrations. Its filesystem is not
used for vault storage. After deployment, verify the boundary:

```bash
curl -i https://<render-service>.onrender.com/api/v1/health
curl -i https://<render-service>.onrender.com/api/v1/session
```

The health request must return `200`; the session request without the gateway
headers must return `404`.

## 4. Configure and deploy Cloudflare

Store both Worker bindings as secrets. `RENDER_ORIGIN` is the Render origin
without a trailing path, and `AETHER_GATEWAY_SECRET` must be byte-for-byte
identical to the Render value:

```bash
cd frontend
pnpm exec wrangler secret put RENDER_ORIGIN
pnpm exec wrangler secret put AETHER_GATEWAY_SECRET
pnpm deploy
```

`pnpm deploy` builds Vite first. Static navigation stays in Cloudflare Assets;
API and auth bodies are streamed through the Worker without buffering. For
local Worker validation, use `pnpm worker:dev` and provide development secrets
through Wrangler's local secret mechanism, never a `VITE_*` variable.

## 5. Smoke test and rollback

Through the Worker origin, verify Google login, session cookies, upload and
progress, range download, search, metadata edits, delete, restore, purge, and
member administration. Restart Render while a deletion is queued and confirm
the persisted job resumes.

If a smoke test fails, roll Render back to its previous deployment and use
Cloudflare Workers & Pages deployment history to roll the Worker back to its
previous version. Database rows and Drive objects require no rollback. The RLS
lockdown should remain in place; restoring browser Data API access requires a
separate reviewed migration with explicit policies and least-privilege grants.
