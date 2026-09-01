CREATE INDEX documents_deleted_purge_idx
ON documents (purge_after, id)
WHERE status = 'deleted';

ALTER TABLE sessions
ADD COLUMN csrf_hash TEXT NOT NULL DEFAULT '';
