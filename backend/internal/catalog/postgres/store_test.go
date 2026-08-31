package postgres

import (
	"context"
	"os"
	"testing"

	"github.com/shxntanu/aether/backend/internal/catalog/repositorytest"
	"github.com/shxntanu/aether/backend/internal/domain"
)

func TestRepositoryContract(t *testing.T) {
	dataSourceName := os.Getenv("AETHER_TEST_POSTGRES_URL")
	if dataSourceName == "" {
		t.Skip("AETHER_TEST_POSTGRES_URL is not set")
	}

	repositorytest.Run(t, func(t *testing.T) domain.Repository {
		t.Helper()
		store, err := Open(context.Background(), dataSourceName)
		if err != nil {
			t.Fatalf("Open() error = %v", err)
		}
		if _, err := store.db.ExecContext(
			context.Background(),
			"TRUNCATE auth_flows, audit_events, document_tags, documents, tags, members CASCADE",
		); err != nil {
			_ = store.Close()
			t.Fatalf("truncate test catalog: %v", err)
		}
		t.Cleanup(func() { _ = store.Close() })
		return store
	})
}
