package gdrive

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSendChunkIncludesDriveErrorBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(writer, `{"error":"insufficientFilePermissions"}`)
	}))
	defer server.Close()

	store := &Store{client: server.Client()}
	_, err := store.sendChunk(
		context.Background(),
		server.URL,
		[]byte("payload"),
		0,
		"7",
		"application/pdf",
	)
	if err == nil || !strings.Contains(err.Error(), "insufficientFilePermissions") {
		t.Fatalf("sendChunk() error = %v, want Drive response details", err)
	}
}

func TestResumableUploadSendsOnlyWritableMetadata(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodPost {
			body, err := io.ReadAll(request.Body)
			if err != nil {
				t.Errorf("read upload metadata: %v", err)
				return
			}
			var metadata map[string]json.RawMessage
			if err := json.Unmarshal(body, &metadata); err != nil {
				t.Errorf("decode upload metadata: %v", err)
				return
			}
			for _, field := range []string{"id", "size", "modifiedTime", "trashed"} {
				if _, ok := metadata[field]; ok {
					t.Errorf("upload metadata contains read-only field %q", field)
				}
			}
			writer.Header().Set("Location", server.URL+"/session")
			writer.WriteHeader(http.StatusOK)
			return
		}
		if request.Method == http.MethodPut && request.URL.Path == "/session" {
			writer.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(writer,
				`{"id":"file-1","mimeType":"application/pdf","size":"7"}`)
			return
		}
		http.NotFound(writer, request)
	}))
	defer server.Close()

	store := &Store{client: server.Client(), uploadBaseURL: server.URL, chunkSize: 256 * 1024}
	_, err := store.resumableUpload(
		context.Background(),
		driveFile{
			Name:          "documents/file-1/original",
			MimeType:      "application/pdf",
			Parents:       []string{"folder-1"},
			AppProperties: map[string]string{"aetherStorageKey": "documents/file-1/original"},
		},
		bytes.NewReader([]byte("payload")),
		"application/pdf",
	)
	if err != nil {
		t.Fatalf("resumableUpload() error = %v", err)
	}
}
