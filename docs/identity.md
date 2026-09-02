# Identity and membership

Aether uses Google OpenID Connect to identify users. Google authentication is
not authorization: an account must also exist in Aether's `members_tbl` table and
have `active` status. Members with the `admin` role can manage that allowlist.

## Configuration

Create a Google OAuth web client and register exactly one callback URI ending
in `/auth/google/callback`. Configure all of these variables together:

```dotenv
AETHER_GOOGLE_CLIENT_ID=...
AETHER_GOOGLE_CLIENT_SECRET=...
AETHER_GOOGLE_REDIRECT_URL=https://vault.example/auth/google/callback
AETHER_BOOTSTRAP_ADMIN_EMAIL=admin@example.com
AETHER_SECURE_COOKIES=true
```

The process refuses partial identity configuration and callback URLs containing
a query or fragment. On startup it creates the bootstrap email as an active
administrator if absent. If the email already exists, it is promoted to admin;
an explicitly disabled account remains disabled.

## Login and sessions

`GET /auth/google/start` creates a ten-minute, single-use login record in
PostgreSQL and redirects to Google with random state, nonce, and S256 PKCE
values. The callback consumes that record atomically before exchanging the
authorization code. The ID token must have a valid Google signature, issuer,
audience, expiry, matching nonce, and verified email.

Successful login sets an `HttpOnly`, `SameSite=Lax` cookie. In HTTPS deployments
`AETHER_SECURE_COOKIES` must be `true`. The browser receives a random opaque
token; PostgreSQL stores only its SHA-256 digest. Sessions expire after seven
days. Authorization reads current member state on every request, so disabling a
member takes effect immediately.

## API

- `GET /api/v1/session` returns the current public member profile.
- `POST /api/v1/logout` invalidates the server-side session and expires the cookie.
- `GET /api/v1/admin/members` lists members (admin only).
- `POST /api/v1/admin/members` adds an active member with `member` or `admin` role.
- `PATCH /api/v1/admin/members/{id}` changes role and `active`/`disabled` status.

The internal Google subject is never serialized in member API responses. The
administrator UI is intentionally deferred; these endpoints are its backend
contract.
