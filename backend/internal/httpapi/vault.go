package httpapi

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/shxntanu/aether/backend/internal/domain"
	"github.com/shxntanu/aether/backend/internal/storage"
	"github.com/shxntanu/aether/backend/internal/vault"
)

const multipartOverhead int64 = 1024 * 1024

func registerVaultRoutes(
	mux *http.ServeMux,
	identityService IdentityService,
	vaultService VaultService,
) {
	memberRoute := func(handler http.HandlerFunc) http.Handler {
		return requireMember(identityService, handler)
	}
	mux.Handle("POST /api/v1/documents", memberRoute(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			handleDocumentUpload(w, r, vaultService)
		},
	)))
	mux.Handle("GET /api/v1/documents", memberRoute(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			handleDocumentList(w, r, vaultService)
		},
	)))
	mux.Handle("GET /api/v1/documents/{id}", memberRoute(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			handleDocumentGet(w, r, vaultService)
		},
	)))
	mux.Handle("PATCH /api/v1/documents/{id}", memberRoute(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			handleDocumentPatch(w, r, vaultService)
		},
	)))
	mux.Handle("DELETE /api/v1/documents/{id}", memberRoute(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			handleDocumentDelete(w, r, vaultService)
		},
	)))
	mux.Handle("POST /api/v1/documents/{id}/restore", requireAdmin(identityService, http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			handleDocumentRestore(w, r, vaultService)
		},
	)))
	mux.Handle("DELETE /api/v1/documents/{id}/purge", requireAdmin(identityService, http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			handleDocumentPurge(w, r, vaultService)
		},
	)))
	mux.Handle("GET /api/v1/documents/{id}/content", memberRoute(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			handleDocumentContent(w, r, vaultService)
		},
	)))
	mux.Handle("GET /api/v1/tags", memberRoute(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			handleTagList(w, r, vaultService)
		},
	)))
	mux.Handle("POST /api/v1/tags", memberRoute(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			handleTagCreate(w, r, vaultService)
		},
	)))
}

func handleDocumentDelete(w http.ResponseWriter, r *http.Request, service VaultService) {
	if err := service.Delete(r.Context(), domain.DocumentID(r.PathValue("id"))); err != nil {
		writeVaultError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleDocumentRestore(w http.ResponseWriter, r *http.Request, service VaultService) {
	record, err := service.Restore(r.Context(), domain.DocumentID(r.PathValue("id")))
	if err != nil {
		writeVaultError(w, err)
		return
	}
	writeRecord(w, http.StatusOK, record)
}

func handleDocumentPurge(w http.ResponseWriter, r *http.Request, service VaultService) {
	if err := service.Purge(
		r.Context(),
		domain.DocumentID(r.PathValue("id")),
		vault.EnforceRetention,
	); err != nil {
		writeVaultError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleDocumentUpload(w http.ResponseWriter, r *http.Request, service VaultService) {
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" || len(idempotencyKey) > 200 {
		writeError(w, http.StatusBadRequest, "idempotency_key_required")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, vault.MaxDocumentSize+multipartOverhead)
	if err := r.ParseMultipartForm(1024 * 1024); err != nil {
		var sizeError *http.MaxBytesError
		if errors.As(err, &sizeError) {
			writeError(w, http.StatusRequestEntityTooLarge, "document_too_large")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid_upload")
		return
	}
	if r.MultipartForm != nil {
		defer func() { _ = r.MultipartForm.RemoveAll() }()
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file_required")
		return
	}
	defer file.Close()
	record, err := service.Upload(r.Context(), vault.Upload{
		Filename:       header.Filename,
		Body:           file,
		Title:          r.FormValue("title"),
		Tags:           formTags(r.MultipartForm.Value["tags"]),
		Uploader:       memberFromContext(r.Context()).ID,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		writeVaultError(w, err)
		return
	}
	writeRecord(w, http.StatusCreated, record)
}

func handleDocumentList(w http.ResponseWriter, r *http.Request, service VaultService) {
	records, err := service.List(
		r.Context(),
		r.URL.Query()["tag"],
		domain.TagMatch(r.URL.Query().Get("match")),
	)
	if err != nil {
		writeVaultError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"documents": records})
}

func handleDocumentGet(w http.ResponseWriter, r *http.Request, service VaultService) {
	record, err := service.Get(r.Context(), domain.DocumentID(r.PathValue("id")))
	if err != nil {
		writeVaultError(w, err)
		return
	}
	writeRecord(w, http.StatusOK, record)
}

func handleDocumentPatch(w http.ResponseWriter, r *http.Request, service VaultService) {
	var input struct {
		Title   string   `json:"title"`
		Tags    []string `json:"tags"`
		Version int64    `json:"version"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request")
		return
	}
	record, err := service.UpdateMetadata(
		r.Context(),
		domain.DocumentID(r.PathValue("id")),
		vault.MetadataUpdate{
			Title:           input.Title,
			Tags:            input.Tags,
			ExpectedVersion: input.Version,
		},
	)
	if err != nil {
		writeVaultError(w, err)
		return
	}
	writeRecord(w, http.StatusOK, record)
}

func handleDocumentContent(w http.ResponseWriter, r *http.Request, service VaultService) {
	byteRange, err := parseRange(r.Header.Get("Range"))
	if err != nil {
		writeError(w, http.StatusRequestedRangeNotSatisfiable, "invalid_range")
		return
	}
	content, err := service.OpenContent(
		r.Context(),
		domain.DocumentID(r.PathValue("id")),
		byteRange,
	)
	if err != nil {
		writeVaultError(w, err)
		return
	}
	defer content.Body.Close()
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Type", content.Document.MediaType)
	disposition := "inline"
	if r.URL.Query().Get("download") == "true" {
		disposition = "attachment"
	}
	w.Header().Set("Content-Disposition", mime.FormatMediaType(
		disposition,
		map[string]string{"filename": sanitizedFilename(content.Document.OriginalFilename)},
	))
	if disposition == "inline" {
		w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'")
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	length := content.Document.SizeBytes
	status := http.StatusOK
	if content.ByteRange != nil {
		length = content.ByteRange.End - content.ByteRange.Start + 1
		status = http.StatusPartialContent
		w.Header().Set("Content-Range", fmt.Sprintf(
			"bytes %d-%d/%d",
			content.ByteRange.Start,
			content.ByteRange.End,
			content.Document.SizeBytes,
		))
	}
	w.Header().Set("Content-Length", strconv.FormatInt(length, 10))
	w.WriteHeader(status)
	_, _ = io.Copy(w, content.Body)
}

func handleTagList(w http.ResponseWriter, r *http.Request, service VaultService) {
	limit := 20
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed < 1 || parsed > 100 {
			writeError(w, http.StatusBadRequest, "invalid_request")
			return
		}
		limit = parsed
	}
	tags, err := service.ListTags(r.Context(), r.URL.Query().Get("q"), limit)
	if err != nil {
		writeVaultError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tags": tags})
}

func handleTagCreate(w http.ResponseWriter, r *http.Request, service VaultService) {
	var input struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request")
		return
	}
	tag, err := service.CreateTag(r.Context(), input.Name)
	if err != nil {
		writeVaultError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, tag)
}

func writeRecord(w http.ResponseWriter, status int, record vault.DocumentRecord) {
	w.Header().Set("ETag", fmt.Sprintf("\"%d\"", record.Document.Version))
	writeJSON(w, status, record)
}

func writeVaultError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound), errors.Is(err, storage.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found")
	case errors.Is(err, domain.ErrConflict):
		writeError(w, http.StatusPreconditionFailed, "version_conflict")
	case errors.Is(err, vault.ErrDocumentTooLarge):
		writeError(w, http.StatusRequestEntityTooLarge, "document_too_large")
	case errors.Is(err, vault.ErrUnsupportedMediaType):
		writeError(w, http.StatusUnsupportedMediaType, "unsupported_media_type")
	case errors.Is(err, vault.ErrInvalidMetadata):
		writeError(w, http.StatusBadRequest, "invalid_request")
	case errors.Is(err, vault.ErrInvalidRange):
		writeError(w, http.StatusRequestedRangeNotSatisfiable, "invalid_range")
	case errors.Is(err, vault.ErrUploadInProgress):
		writeError(w, http.StatusConflict, "upload_in_progress")
	case errors.Is(err, vault.ErrRetentionActive):
		writeError(w, http.StatusConflict, "retention_active")
	default:
		writeError(w, http.StatusInternalServerError, "vault_error")
	}
}

func sanitizedFilename(filename string) string {
	filename = strings.TrimSpace(strings.ReplaceAll(filename, "\\", "/"))
	filename = filepath.Base(filename)
	filename = strings.ToValidUTF8(filename, "_")
	filename = strings.Map(func(character rune) rune {
		if character < 0x20 || character == 0x7f {
			return '_'
		}
		return character
	}, filename)
	if filename == "" || filename == "." || filename == ".." ||
		filename == string(filepath.Separator) {
		return "download"
	}
	return filename
}

func parseRange(value string) (*storage.ByteRange, error) {
	if value == "" {
		return nil, nil
	}
	if !strings.HasPrefix(value, "bytes=") || strings.Contains(value, ",") {
		return nil, errors.New("unsupported range")
	}
	parts := strings.Split(strings.TrimPrefix(value, "bytes="), "-")
	if len(parts) != 2 || (parts[0] == "" && parts[1] == "") {
		return nil, errors.New("range endpoint required")
	}
	start := int64(-1)
	end := int64(-1)
	var startErr error
	var endErr error
	if parts[0] != "" {
		start, startErr = strconv.ParseInt(parts[0], 10, 64)
	}
	if parts[1] != "" {
		end, endErr = strconv.ParseInt(parts[1], 10, 64)
	}
	if startErr != nil || endErr != nil || start < -1 || end < -1 ||
		(start >= 0 && end >= 0 && end < start) || (start == -1 && end == 0) {
		return nil, errors.New("invalid range")
	}
	return &storage.ByteRange{Start: start, End: end}, nil
}

func formTags(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		for _, name := range strings.Split(value, ",") {
			if strings.TrimSpace(name) != "" {
				result = append(result, name)
			}
		}
	}
	return result
}
