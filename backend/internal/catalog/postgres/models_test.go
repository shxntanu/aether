package postgres

import "testing"

func TestPersistenceModelsUseExpectedTableNames(t *testing.T) {
	tableNames := map[string]string{
		"schema migration": schemaMigrationsTable,
		"document":         documentModel{}.TableName(),
		"tag":              tagModel{}.TableName(),
		"document tag":     documentTagModel{}.TableName(),
		"upload":           uploadRequestModel{}.TableName(),
		"member":           memberModel{}.TableName(),
		"session":          sessionModel{}.TableName(),
		"auth flow":        authFlowModel{}.TableName(),
		"audit event":      auditEventModel{}.TableName(),
	}
	expecteds := map[string]string{
		"schema migration": schemaMigrationsTable,
		"document":         "documents_tbl",
		"tag":              "tags_tbl",
		"document tag":     "document_tags_tbl",
		"upload":           "upload_requests_tbl",
		"member":           "members_tbl",
		"session":          "sessions_tbl",
		"auth flow":        "auth_flows_tbl",
		"audit event":      "audit_events_tbl",
	}

	for name, expected := range expecteds {
		if tableNames[name] != expected {
			t.Errorf("%s table name = %q, want %q", name, tableNames[name], expected)
		}
	}
}
