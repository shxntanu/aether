ALTER TABLE tags_tbl
ADD COLUMN IF NOT EXISTS is_implicit BOOLEAN NOT NULL DEFAULT FALSE;

WITH document_dates AS (
    SELECT DISTINCT to_char(created_at AT TIME ZONE 'Asia/Kolkata', 'YYYY-MM-DD') AS date_name
    FROM documents_tbl
)
INSERT INTO tags_tbl (id, display_name, normalized_name, is_implicit)
SELECT 'implicit-date-' || md5(date_name), date_name, date_name, TRUE
FROM document_dates
ON CONFLICT (normalized_name) DO UPDATE
SET is_implicit = tags_tbl.is_implicit OR EXCLUDED.is_implicit;

INSERT INTO document_tags_tbl (document_id, tag_id)
SELECT d.id, t.id
FROM documents_tbl AS d
JOIN tags_tbl AS t
  ON t.normalized_name = to_char(d.created_at AT TIME ZONE 'Asia/Kolkata', 'YYYY-MM-DD')
WHERE t.is_implicit
ON CONFLICT (document_id, tag_id) DO NOTHING;

-- Rollback: remove only the date associations and tags created by this migration,
-- then retain the additive tag column for compatibility with newer binaries.
