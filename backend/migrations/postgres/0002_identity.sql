ALTER TABLE members_tbl ADD COLUMN oidc_subject TEXT;
CREATE UNIQUE INDEX members_oidc_subject_idx ON members_tbl(oidc_subject) WHERE oidc_subject IS NOT NULL;

CREATE TABLE sessions_tbl (
    id TEXT PRIMARY KEY,
    token_hash TEXT NOT NULL UNIQUE,
    member_id TEXT NOT NULL REFERENCES members_tbl(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX sessions_member_idx ON sessions_tbl(member_id);
CREATE INDEX sessions_expiry_idx ON sessions_tbl(expires_at);

CREATE TABLE auth_flows_tbl (
    state_hash TEXT PRIMARY KEY,
    nonce TEXT NOT NULL,
    pkce_verifier TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX auth_flows_expiry_idx ON auth_flows_tbl(expires_at);
