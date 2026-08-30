package domain

import (
	"errors"
	"testing"
	"time"
)

func TestDocumentTransitionMovesUploadingDocumentToReady(t *testing.T) {
	now := time.Date(2026, time.August, 31, 1, 0, 0, 0, time.UTC)
	document := Document{Status: DocumentStatusUploading, UpdatedAt: now.Add(-time.Minute)}

	err := document.TransitionTo(DocumentStatusReady, now)

	if err != nil {
		t.Fatalf("TransitionTo() error = %v", err)
	}
	if document.Status != DocumentStatusReady {
		t.Fatalf("Status = %q, want %q", document.Status, DocumentStatusReady)
	}
	if !document.UpdatedAt.Equal(now) {
		t.Fatalf("UpdatedAt = %s, want %s", document.UpdatedAt, now)
	}
}

func TestDocumentTransitionRejectsReadyToUploading(t *testing.T) {
	document := Document{Status: DocumentStatusReady}

	err := document.TransitionTo(DocumentStatusUploading, time.Now())

	if !errors.Is(err, ErrInvalidDocumentTransition) {
		t.Fatalf("TransitionTo() error = %v, want ErrInvalidDocumentTransition", err)
	}
	if document.Status != DocumentStatusReady {
		t.Fatalf("Status = %q, want unchanged status %q", document.Status, DocumentStatusReady)
	}
}

func TestDocumentTransitionRestoresDeletedDocument(t *testing.T) {
	deletedAt := time.Date(2026, time.August, 30, 1, 0, 0, 0, time.UTC)
	purgeAfter := deletedAt.Add(30 * 24 * time.Hour)
	document := Document{
		Status:     DocumentStatusDeleted,
		DeletedAt:  &deletedAt,
		PurgeAfter: &purgeAfter,
	}

	err := document.TransitionTo(DocumentStatusReady, deletedAt.Add(time.Hour))

	if err != nil {
		t.Fatalf("TransitionTo() error = %v", err)
	}
	if document.DeletedAt != nil || document.PurgeAfter != nil {
		t.Fatalf("restore retained deletion timestamps: DeletedAt=%v PurgeAfter=%v", document.DeletedAt, document.PurgeAfter)
	}
}

func TestDocumentSoftDeleteSetsRetentionDeadline(t *testing.T) {
	now := time.Date(2026, time.August, 31, 1, 0, 0, 0, time.UTC)
	document := Document{Status: DocumentStatusReady}

	err := document.SoftDelete(now, 30*24*time.Hour)

	if err != nil {
		t.Fatalf("SoftDelete() error = %v", err)
	}
	if document.Status != DocumentStatusDeleted {
		t.Fatalf("Status = %q, want %q", document.Status, DocumentStatusDeleted)
	}
	if document.DeletedAt == nil || !document.DeletedAt.Equal(now) {
		t.Fatalf("DeletedAt = %v, want %s", document.DeletedAt, now)
	}
	wantPurgeAfter := now.Add(30 * 24 * time.Hour)
	if document.PurgeAfter == nil || !document.PurgeAfter.Equal(wantPurgeAfter) {
		t.Fatalf("PurgeAfter = %v, want %s", document.PurgeAfter, wantPurgeAfter)
	}
}
