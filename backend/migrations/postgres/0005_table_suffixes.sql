DO $$
BEGIN
    IF to_regclass('public.members') IS NOT NULL
        AND to_regclass('public.members_tbl') IS NULL THEN
        ALTER TABLE members RENAME TO members_tbl;
    END IF;
    IF to_regclass('public.documents') IS NOT NULL
        AND to_regclass('public.documents_tbl') IS NULL THEN
        ALTER TABLE documents RENAME TO documents_tbl;
    END IF;
    IF to_regclass('public.tags') IS NOT NULL
        AND to_regclass('public.tags_tbl') IS NULL THEN
        ALTER TABLE tags RENAME TO tags_tbl;
    END IF;
    IF to_regclass('public.document_tags') IS NOT NULL
        AND to_regclass('public.document_tags_tbl') IS NULL THEN
        ALTER TABLE document_tags RENAME TO document_tags_tbl;
    END IF;
    IF to_regclass('public.audit_events') IS NOT NULL
        AND to_regclass('public.audit_events_tbl') IS NULL THEN
        ALTER TABLE audit_events RENAME TO audit_events_tbl;
    END IF;
    IF to_regclass('public.sessions') IS NOT NULL
        AND to_regclass('public.sessions_tbl') IS NULL THEN
        ALTER TABLE sessions RENAME TO sessions_tbl;
    END IF;
    IF to_regclass('public.auth_flows') IS NOT NULL
        AND to_regclass('public.auth_flows_tbl') IS NULL THEN
        ALTER TABLE auth_flows RENAME TO auth_flows_tbl;
    END IF;
    IF to_regclass('public.upload_requests') IS NOT NULL
        AND to_regclass('public.upload_requests_tbl') IS NULL THEN
        ALTER TABLE upload_requests RENAME TO upload_requests_tbl;
    END IF;
END $$;
