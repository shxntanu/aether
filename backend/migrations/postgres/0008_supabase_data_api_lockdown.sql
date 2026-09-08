-- Aether is server-only: browser-facing Supabase roles must not read or mutate
-- catalog, identity, audit, or migration state through the Data API.
ALTER TABLE public.schema_migrations_tbl ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.members_tbl ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.documents_tbl ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.tags_tbl ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.document_tags_tbl ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.audit_events_tbl ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.sessions_tbl ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.auth_flows_tbl ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.upload_requests_tbl ENABLE ROW LEVEL SECURITY;

-- Local PostgreSQL installations need not define Supabase's browser roles, so
-- apply the defense-in-depth privilege revocation only when each role exists.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'anon') THEN
        REVOKE ALL PRIVILEGES ON TABLE
            public.schema_migrations_tbl,
            public.members_tbl,
            public.documents_tbl,
            public.tags_tbl,
            public.document_tags_tbl,
            public.audit_events_tbl,
            public.sessions_tbl,
            public.auth_flows_tbl,
            public.upload_requests_tbl
        FROM anon;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'authenticated') THEN
        REVOKE ALL PRIVILEGES ON TABLE
            public.schema_migrations_tbl,
            public.members_tbl,
            public.documents_tbl,
            public.tags_tbl,
            public.document_tags_tbl,
            public.audit_events_tbl,
            public.sessions_tbl,
            public.auth_flows_tbl,
            public.upload_requests_tbl
        FROM authenticated;
    END IF;
END $$;

-- Rollback, if browser access is ever intentionally introduced, requires an
-- explicit policy migration plus least-privilege GRANT statements. Disabling
-- RLS alone is deliberately not an acceptable rollback for these tables.
