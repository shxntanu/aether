package vault

import (
	"context"
	"testing"
	"time"

	"github.com/shxntanu/aether/backend/internal/domain"
	"github.com/shxntanu/aether/backend/internal/storage"
)

func TestDeleteQueuesStorageWorkAndReturnsCurrentRecord(t *testing.T) {
	now := time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)
	repository := &deletionRepository{document: domain.Document{
		ID:         "document-1",
		Title:      "Passport",
		StorageKey: "documents/document-1/original",
		Status:     domain.DocumentStatusReady,
		Version:    1,
		CreatedAt:  now,
		UpdatedAt:  now,
	}}
	objects := &deletionObjectStore{}
	service := NewService(repository, objects)
	service.now = func() time.Time { return now }

	record, err := service.Delete(context.Background(), repository.document.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if record.Document.DeletionStatus != domain.DeletionStatusQueued {
		t.Fatalf("Delete() status = %q, want queued", record.Document.DeletionStatus)
	}
	if len(objects.trashed) != 0 {
		t.Fatalf("Delete() trashed synchronously: %v", objects.trashed)
	}

	result, err := service.ProcessDeletions(context.Background(), 1)
	if err != nil {
		t.Fatalf("ProcessDeletions() error = %v", err)
	}
	if len(result.Completed) != 1 || result.Completed[0] != repository.document.ID {
		t.Fatalf("ProcessDeletions() completed = %v", result.Completed)
	}
	if repository.document.DeletionStatus != domain.DeletionStatusComplete {
		t.Fatalf("persisted deletion status = %q, want complete", repository.document.DeletionStatus)
	}
	if len(objects.trashed) != 2 {
		t.Fatalf("trashed keys = %v, want original and manifest", objects.trashed)
	}
}

type deletionRepository struct {
	domain.Repository
	document domain.Document
}

func (r *deletionRepository) GetDocument(
	_ context.Context,
	id domain.DocumentID,
) (domain.Document, error) {
	if id != r.document.ID {
		return domain.Document{}, domain.ErrNotFound
	}
	return r.document, nil
}

func (r *deletionRepository) UpdateDocument(
	_ context.Context,
	document domain.Document,
	expectedVersion int64,
) (domain.Document, error) {
	if expectedVersion != r.document.Version {
		return domain.Document{}, domain.ErrConflict
	}
	document.Version++
	r.document = document
	return document, nil
}

func (r *deletionRepository) ListDocuments(
	_ context.Context,
	_ domain.DocumentListOptions,
) ([]domain.Document, error) {
	return []domain.Document{r.document}, nil
}

func (r *deletionRepository) ListDocumentTags(
	context.Context,
	domain.DocumentID,
) ([]domain.Tag, error) {
	return []domain.Tag{}, nil
}

type deletionObjectStore struct {
	storage.ObjectStore
	trashed []string
}

func (s *deletionObjectStore) Trash(_ context.Context, key string) error {
	s.trashed = append(s.trashed, key)
	return nil
}
