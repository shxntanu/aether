ALTER TABLE documents_tbl
ADD COLUMN deletion_status TEXT NOT NULL DEFAULT '',
ADD COLUMN deletion_error TEXT NOT NULL DEFAULT '';

UPDATE documents_tbl
SET deletion_status = 'complete'
WHERE status = 'deleted';

CREATE INDEX documents_deletion_work_idx
ON documents_tbl (updated_at, id)
WHERE status = 'deleted'
    AND deletion_status IN ('queued', 'processing', 'failed');
