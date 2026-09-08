package migrations

import (
	"strings"
	"testing"
)

func TestSupabaseLockdownCoversEveryPublicAetherTable(t *testing.T) {
	migrationList, err := PostgreSQL()
	if err != nil {
		t.Fatalf("PostgreSQL() error = %v", err)
	}
	var lockdown string
	for _, migration := range migrationList {
		if migration.Version == "0008_supabase_data_api_lockdown.sql" {
			lockdown = migration.SQL
			break
		}
	}
	if lockdown == "" {
		t.Fatal("Supabase lockdown migration is missing")
	}

	for _, table := range []string{
		"schema_migrations_tbl",
		"members_tbl",
		"documents_tbl",
		"tags_tbl",
		"document_tags_tbl",
		"audit_events_tbl",
		"sessions_tbl",
		"auth_flows_tbl",
		"upload_requests_tbl",
	} {
		statement := "ALTER TABLE public." + table + " ENABLE ROW LEVEL SECURITY"
		if !strings.Contains(lockdown, statement) {
			t.Errorf("lockdown does not enable RLS on %s", table)
		}
	}
	for _, role := range []string{"anon", "authenticated"} {
		if !strings.Contains(lockdown, "rolname = '"+role+"'") {
			t.Errorf("lockdown does not handle the %s role", role)
		}
	}
}
