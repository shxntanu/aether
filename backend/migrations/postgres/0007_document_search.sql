CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX IF NOT EXISTS documents_title_trgm_idx
ON documents_tbl USING GIN (lower(title) gin_trgm_ops);

CREATE INDEX IF NOT EXISTS documents_original_filename_trgm_idx
ON documents_tbl USING GIN (lower(original_filename) gin_trgm_ops);

CREATE INDEX IF NOT EXISTS tags_normalized_name_trgm_idx
ON tags_tbl USING GIN (lower(normalized_name) gin_trgm_ops);

-- Rollback intentionally drops only the three indexes above. pg_trgm is
-- retained because the extension may be shared with other database features.
