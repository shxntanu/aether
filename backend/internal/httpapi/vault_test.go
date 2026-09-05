package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shxntanu/aether/backend/internal/domain"
	"github.com/shxntanu/aether/backend/internal/vault"
)

func TestDocumentDeleteReturnsQueuedCurrentRecord(t *testing.T) {
	service := &deleteVaultStub{record: vault.DocumentRecord{
		Document: domain.Document{
			ID:             "document-1",
			Status:         domain.DocumentStatusDeleted,
			DeletionStatus: domain.DeletionStatusQueued,
			Version:        2,
		},
		Tags: []domain.Tag{},
	}}
	request := httptest.NewRequest(http.MethodDelete, "/api/v1/documents/document-1", nil)
	request.SetPathValue("id", "document-1")
	response := httptest.NewRecorder()

	handleDocumentDelete(response, request, service)

	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusAccepted)
	}
	var record vault.DocumentRecord
	if err := json.NewDecoder(response.Body).Decode(&record); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if record.Document != service.record.Document {
		t.Fatalf("response document = %#v, want %#v", record.Document, service.record.Document)
	}
	if response.Header().Get("ETag") != `"2"` {
		t.Fatalf("ETag = %q, want %q", response.Header().Get("ETag"), `"2"`)
	}
}

type deleteVaultStub struct {
	VaultService
	record vault.DocumentRecord
}

func (s *deleteVaultStub) Delete(
	context.Context,
	domain.DocumentID,
) (vault.DocumentRecord, error) {
	return s.record, nil
}
