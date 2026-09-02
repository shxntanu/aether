CREATE TABLE IF NOT EXISTS schema_migrations_tbl (
    version TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE members_tbl (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('member', 'admin')),
    status TEXT NOT NULL CHECK (status IN ('active', 'disabled')),
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE documents_tbl (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    original_filename TEXT NOT NULL,
    media_type TEXT NOT NULL,
    size_bytes BIGINT NOT NULL CHECK (size_bytes >= 0),
    sha256 TEXT NOT NULL,
    storage_key TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL CHECK (status IN ('uploading', 'ready', 'failed', 'deleted')),
    index_status TEXT NOT NULL CHECK (index_status IN ('not_scheduled', 'queued', 'extracting', 'enriching', 'indexed', 'failed')),
    uploader_id TEXT NOT NULL REFERENCES members_tbl(id),
    version BIGINT NOT NULL CHECK (version > 0),
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    deleted_at TIMESTAMPTZ,
    purge_after TIMESTAMPTZ
);

CREATE TABLE tags_tbl (
    id TEXT PRIMARY KEY,
    display_name TEXT NOT NULL,
    normalized_name TEXT NOT NULL UNIQUE
);

CREATE TABLE document_tags_tbl (
    document_id TEXT NOT NULL REFERENCES documents_tbl(id) ON DELETE CASCADE,
    tag_id TEXT NOT NULL REFERENCES tags_tbl(id) ON DELETE CASCADE,
    PRIMARY KEY (document_id, tag_id)
);

CREATE TABLE audit_events_tbl (
    id TEXT PRIMARY KEY,
    actor_id TEXT REFERENCES members_tbl(id),
    action TEXT NOT NULL,
    object_type TEXT NOT NULL,
    object_id TEXT NOT NULL,
    outcome TEXT NOT NULL CHECK (outcome IN ('succeeded', 'rejected', 'failed')),
    occurred_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX audit_events_object_idx ON audit_events_tbl(object_type, object_id, occurred_at, id);
