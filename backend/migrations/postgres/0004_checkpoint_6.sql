CREATE INDEX documents_deleted_purge_idx
ON documents_tbl (purge_after, id)
WHERE status = 'deleted';

ALTER TABLE sessions_tbl
ADD COLUMN csrf_hash TEXT NOT NULL DEFAULT '';
