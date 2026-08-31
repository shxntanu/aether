ALTER TABLE documents
ADD COLUMN manifest_error TEXT NOT NULL DEFAULT '';

CREATE INDEX documents_ready_created_idx
ON documents (created_at DESC, id DESC)
WHERE status = 'ready';

CREATE INDEX document_tags_tag_document_idx
ON document_tags (tag_id, document_id);

CREATE TABLE upload_requests (
    member_id TEXT NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    key_hash TEXT NOT NULL,
    document_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (member_id, key_hash)
);
