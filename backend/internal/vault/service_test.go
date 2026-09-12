package vault

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/shxntanu/aether/backend/internal/domain"
)

func TestGetIncludesUploaderDisplayName(t *testing.T) {
	createdAt := time.Date(2026, time.September, 12, 12, 0, 0, 0, time.UTC)
	repository := &displayNameRepository{
		document: domain.Document{
			ID:         "document-1",
			Title:      "Barclays Logo",
			Status:     domain.DocumentStatusReady,
			UploaderID: "opaque-member-id",
			Version:    1,
			CreatedAt:  createdAt,
			UpdatedAt:  createdAt,
		},
		member: domain.Member{
			ID:          "opaque-member-id",
			DisplayName: "Shantanu Wable",
		},
	}
	service := NewService(repository, nil)

	record, err := service.Get(context.Background(), repository.document.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	payload, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if !strings.Contains(string(payload), `"uploaderName":"Shantanu Wable"`) {
		t.Fatalf("Get() JSON = %s, want uploaderName", payload)
	}
}

type displayNameRepository struct {
	domain.Repository
	document domain.Document
	member   domain.Member
}

func (r *displayNameRepository) GetDocument(
	_ context.Context,
	id domain.DocumentID,
) (domain.Document, error) {
	if id != r.document.ID {
		return domain.Document{}, domain.ErrNotFound
	}
	return r.document, nil
}

func (r *displayNameRepository) ListDocumentTags(
	context.Context,
	domain.DocumentID,
) ([]domain.Tag, error) {
	return []domain.Tag{}, nil
}

func (r *displayNameRepository) GetMember(
	_ context.Context,
	id domain.MemberID,
) (domain.Member, error) {
	if id != r.member.ID {
		return domain.Member{}, domain.ErrNotFound
	}
	return r.member, nil
}
