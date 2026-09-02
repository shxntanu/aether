package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

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
			"TRUNCATE auth_flows_tbl, audit_events_tbl, document_tags_tbl, "+
				"documents_tbl, tags_tbl, members_tbl CASCADE",
		); err != nil {
			_ = store.Close()
			t.Fatalf("truncate test catalog: %v", err)
		}
		t.Cleanup(func() { _ = store.Close() })
		return store
	})
}

func TestDocumentFiltersPurgeAndUploadClaims(t *testing.T) {
	dataSourceName := os.Getenv("AETHER_TEST_POSTGRES_URL")
	if dataSourceName == "" {
		t.Skip("AETHER_TEST_POSTGRES_URL is not set")
	}

	store, err := Open(context.Background(), dataSourceName)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()
	if _, err := store.db.ExecContext(
		context.Background(),
		"TRUNCATE auth_flows_tbl, audit_events_tbl, document_tags_tbl, "+
			"documents_tbl, tags_tbl, members_tbl CASCADE",
	); err != nil {
		t.Fatalf("truncate test catalog: %v", err)
	}

	ctx := context.Background()
	member := postgresMemberFixture("member-filters", "filters@example.com")
	if err := store.CreateMember(ctx, member); err != nil {
		t.Fatalf("CreateMember() error = %v", err)
	}
	ready := postgresDocumentFixture("document-list-ready", member.ID)
	ready.StorageKey = "documents/document-list-ready/original"
	deleted := postgresDocumentFixture("document-list-deleted", member.ID)
	deleted.StorageKey = "documents/document-list-deleted/original"
	deleted.Status = domain.DocumentStatusDeleted
	deletedAt := time.Date(2026, time.August, 31, 2, 0, 0, 0, time.UTC)
	deleted.DeletedAt = &deletedAt
	purgeAfter := deletedAt.Add(time.Hour)
	deleted.PurgeAfter = &purgeAfter
	for _, document := range []domain.Document{ready, deleted} {
		if err := store.CreateDocument(ctx, document); err != nil {
			t.Fatalf("CreateDocument(%q) error = %v", document.ID, err)
		}
	}

	tag, err := domain.NewTag("tag-filter", "Filter")
	if err != nil {
		t.Fatalf("NewTag() error = %v", err)
	}
	if err := store.CreateTag(ctx, tag); err != nil {
		t.Fatalf("CreateTag() error = %v", err)
	}
	if err := store.ReplaceDocumentTags(ctx, ready.ID, []domain.TagID{tag.ID}); err != nil {
		t.Fatalf("ReplaceDocumentTags() error = %v", err)
	}

	matching, err := store.ListDocuments(ctx, domain.DocumentListOptions{
		NormalizedTags: []string{tag.NormalizedName},
	})
	if err != nil {
		t.Fatalf("ListDocuments() error = %v", err)
	}
	if len(matching) != 1 || matching[0].ID != ready.ID {
		t.Fatalf("ListDocuments() = %#v, want ready tagged document", matching)
	}

	dueBefore := purgeAfter.Add(time.Minute)
	due, err := store.ListDocuments(ctx, domain.DocumentListOptions{
		Statuses:       []domain.DocumentStatus{domain.DocumentStatusDeleted},
		PurgeDueBefore: &dueBefore,
	})
	if err != nil {
		t.Fatalf("ListDocuments() due error = %v", err)
	}
	if len(due) != 1 || due[0].ID != deleted.ID {
		t.Fatalf("ListDocuments() due = %#v, want deleted document", due)
	}

	claimed, first, err := store.ClaimUpload(ctx, member.ID, "key-hash", ready.ID, time.Now())
	if err != nil || !first || claimed != ready.ID {
		t.Fatalf("ClaimUpload() first = %q, %t, %v", claimed, first, err)
	}
	claimed, first, err = store.ClaimUpload(ctx, member.ID, "key-hash", deleted.ID, time.Now())
	if err != nil || first || claimed != ready.ID {
		t.Fatalf("ClaimUpload() duplicate = %q, %t, %v", claimed, first, err)
	}

	if err := store.PurgeDocument(ctx, deleted.ID); err != nil {
		t.Fatalf("PurgeDocument() error = %v", err)
	}
	if _, err := store.GetDocument(ctx, deleted.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("GetDocument() after purge error = %v, want ErrNotFound", err)
	}
	if err := store.PurgeDocument(ctx, ready.ID); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("PurgeDocument() ready error = %v, want ErrConflict", err)
	}
}

func postgresDocumentFixture(id domain.DocumentID, uploaderID domain.MemberID) domain.Document {
	createdAt := time.Date(2026, time.August, 31, 1, 30, 0, 0, time.UTC)
	return domain.Document{
		ID:               id,
		Title:            "Test document",
		OriginalFilename: "test.pdf",
		MediaType:        "application/pdf",
		SizeBytes:        2048,
		SHA256:           string(id) + "-sha256",
		StorageKey:       "documents/" + string(id) + "/original",
		Status:           domain.DocumentStatusReady,
		IndexStatus:      domain.IndexStatusNotScheduled,
		UploaderID:       uploaderID,
		Version:          1,
		CreatedAt:        createdAt,
		UpdatedAt:        createdAt,
	}
}

func postgresMemberFixture(id domain.MemberID, email string) domain.Member {
	createdAt := time.Date(2026, time.August, 31, 1, 0, 0, 0, time.UTC)
	return domain.Member{
		ID:          id,
		Email:       email,
		DisplayName: "Test Member",
		Role:        domain.MemberRoleMember,
		Status:      domain.MemberStatusActive,
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
	}
}
