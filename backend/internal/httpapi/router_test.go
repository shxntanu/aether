package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthReturnsOK(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	response := httptest.NewRecorder()

	NewRouter().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	var body struct {
		Status string `json:"status"`
	}

	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Status != "ok" {
		t.Fatalf("expected status value %q, got %q", "ok", body.Status)
	}
}

func TestWriteVaultErrorLogsInternalFailure(t *testing.T) {
	var logs bytes.Buffer
	logger := log.New(&logs, "", 0)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/documents", nil)
	request = request.WithContext(context.WithValue(
		request.Context(),
		requestLoggerContextKey{},
		logger,
	))
	response := httptest.NewRecorder()

	writeVaultError(response, request, errors.New("storage unavailable"))

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, response.Code)
	}
	if !strings.Contains(logs.String(), "storage unavailable") {
		t.Fatalf("expected underlying error in log, got %q", logs.String())
	}
}
