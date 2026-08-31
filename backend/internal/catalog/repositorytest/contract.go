package repositorytest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shxntanu/aether/backend/internal/domain"
)

type Factory func(t *testing.T) domain.Repository

func Run(t *testing.T, factory Factory) {
	t.Helper()

	t.Run("persists and updates a document", func(t *testing.T) {
		repository := factory(t)
		ctx := context.Background()
		member := memberFixture("member-document", "document@example.com")
		mustCreateMember(t, ctx, repository, member)

		document := documentFixture("document-1", member.ID)
		if err := repository.CreateDocument(ctx, document); err != nil {
			t.Fatalf("CreateDocument() error = %v", err)
		}

		got, err := repository.GetDocument(ctx, document.ID)
		if err != nil {
			t.Fatalf("GetDocument() error = %v", err)
		}
		if got.Title != "Electricity bill" || got.Version != 1 {
			t.Fatalf("GetDocument() = %#v, want title and version from fixture", got)
		}

		got.Title = "Electricity bill — August"
		updated, err := repository.UpdateDocument(ctx, got, 1)
		if err != nil {
			t.Fatalf("UpdateDocument() error = %v", err)
		}
		if updated.Title != "Electricity bill — August" || updated.Version != 2 {
			t.Fatalf("UpdateDocument() = %#v, want changed title and version 2", updated)
		}
	})

	t.Run("rejects stale document versions without changing data", func(t *testing.T) {
		repository := factory(t)
		ctx := context.Background()
		member := memberFixture("member-conflict", "conflict@example.com")
		mustCreateMember(t, ctx, repository, member)
		document := documentFixture("document-conflict", member.ID)
		if err := repository.CreateDocument(ctx, document); err != nil {
			t.Fatalf("CreateDocument() error = %v", err)
		}

		document.Title = "Stale title"
		_, err := repository.UpdateDocument(ctx, document, 0)
		if !errors.Is(err, domain.ErrConflict) {
			t.Fatalf("UpdateDocument() error = %v, want ErrConflict", err)
		}

		got, err := repository.GetDocument(ctx, document.ID)
		if err != nil {
			t.Fatalf("GetDocument() error = %v", err)
		}
		if got.Title != "Electricity bill" || got.Version != 1 {
			t.Fatalf("stale update changed document: %#v", got)
		}
	})

	t.Run("persists reusable tags and document relationships", func(t *testing.T) {
		repository := factory(t)
		ctx := context.Background()
		member := memberFixture("member-tags", "tags@example.com")
		mustCreateMember(t, ctx, repository, member)
		document := documentFixture("document-tags", member.ID)
		if err := repository.CreateDocument(ctx, document); err != nil {
			t.Fatalf("CreateDocument() error = %v", err)
		}
		tag, err := domain.NewTag("tag-utilities", "Utility Bills")
		if err != nil {
			t.Fatalf("NewTag() error = %v", err)
		}
		if err := repository.CreateTag(ctx, tag); err != nil {
			t.Fatalf("CreateTag() error = %v", err)
		}
		if err := repository.AttachTag(ctx, document.ID, tag.ID); err != nil {
			t.Fatalf("AttachTag() error = %v", err)
		}

		got, err := repository.GetTagByNormalizedName(ctx, "utility bills")
		if err != nil {
			t.Fatalf("GetTagByNormalizedName() error = %v", err)
		}
		if got != tag {
			t.Fatalf("GetTagByNormalizedName() = %#v, want %#v", got, tag)
		}

		tags, err := repository.ListDocumentTags(ctx, document.ID)
		if err != nil {
			t.Fatalf("ListDocumentTags() error = %v", err)
		}
		if len(tags) != 1 || tags[0] != tag {
			t.Fatalf("ListDocumentTags() = %#v, want [%#v]", tags, tag)
		}
	})

	t.Run("persists members and audit events", func(t *testing.T) {
		repository := factory(t)
		ctx := context.Background()
		member := memberFixture("member-audit", "AUDIT@Example.com")
		mustCreateMember(t, ctx, repository, member)

		gotMember, err := repository.GetMemberByEmail(ctx, "audit@example.com")
		if err != nil {
			t.Fatalf("GetMemberByEmail() error = %v", err)
		}
		if gotMember.ID != member.ID {
			t.Fatalf("GetMemberByEmail().ID = %q, want %q", gotMember.ID, member.ID)
		}

		event := domain.AuditEvent{
			ID:         "audit-1",
			ActorID:    &member.ID,
			Action:     "document.upload",
			ObjectType: "document",
			ObjectID:   "document-audit",
			Outcome:    domain.AuditOutcomeSucceeded,
			OccurredAt: time.Date(2026, time.August, 31, 2, 0, 0, 0, time.UTC),
		}
		if err := repository.AppendAuditEvent(ctx, event); err != nil {
			t.Fatalf("AppendAuditEvent() error = %v", err)
		}
		events, err := repository.ListAuditEvents(ctx, "document", "document-audit")
		if err != nil {
			t.Fatalf("ListAuditEvents() error = %v", err)
		}
		if len(events) != 1 || events[0].ID != event.ID || events[0].ActorID == nil || *events[0].ActorID != member.ID {
			t.Fatalf("ListAuditEvents() = %#v, want persisted event", events)
		}
	})

	t.Run("rolls back a failed transaction", func(t *testing.T) {
		repository := factory(t)
		ctx := context.Background()
		member := memberFixture("member-rollback", "rollback@example.com")
		operationError := errors.New("stop transaction")

		err := repository.WithinTransaction(ctx, func(transaction domain.Repository) error {
			if err := transaction.CreateMember(ctx, member); err != nil {
				return err
			}
			return operationError
		})
		if !errors.Is(err, operationError) {
			t.Fatalf("WithinTransaction() error = %v, want operation error", err)
		}

		_, err = repository.GetMemberByEmail(ctx, member.Email)
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("GetMemberByEmail() after rollback error = %v, want ErrNotFound", err)
		}
	})
}

func documentFixture(id domain.DocumentID, uploaderID domain.MemberID) domain.Document {
	createdAt := time.Date(2026, time.August, 31, 1, 30, 0, 0, time.UTC)
	return domain.Document{
		ID:               id,
		Title:            "Electricity bill",
		OriginalFilename: "scan.pdf",
		MediaType:        "application/pdf",
		SizeBytes:        2048,
		SHA256:           "b6d81b360a5672d80c27430f39153e2c55bfeb8a5c4c9052f244b0e98c97de2b",
		StorageKey:       "documents/document-1/original",
		Status:           domain.DocumentStatusReady,
		IndexStatus:      domain.IndexStatusNotScheduled,
		UploaderID:       uploaderID,
		Version:          1,
		CreatedAt:        createdAt,
		UpdatedAt:        createdAt,
	}
}

func memberFixture(id domain.MemberID, email string) domain.Member {
	createdAt := time.Date(2026, time.August, 31, 1, 0, 0, 0, time.UTC)
	return domain.Member{
		ID:          id,
		Email:       email,
		DisplayName: "Family Member",
		Role:        domain.MemberRoleMember,
		Status:      domain.MemberStatusActive,
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
	}
}

func mustCreateMember(t *testing.T, ctx context.Context, repository domain.Repository, member domain.Member) {
	t.Helper()
	if err := repository.CreateMember(ctx, member); err != nil {
		t.Fatalf("CreateMember() error = %v", err)
	}
}
