package httpapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/shxntanu/aether/backend/internal/audit"
	"github.com/shxntanu/aether/backend/internal/domain"
	"github.com/shxntanu/aether/backend/internal/storage"
	"github.com/shxntanu/aether/backend/internal/vault"
)

const multipartOverhead int64 = 1024 * 1024

type actorVaultService interface {
	DeleteByActor(context.Context, domain.DocumentID, domain.MemberID) error
	RestoreByActor(
		context.Context,
		domain.DocumentID,
		domain.MemberID,
	) (vault.DocumentRecord, error)
	PurgeByActor(
		context.Context,
		domain.DocumentID,
		vault.PurgePolicy,
		domain.MemberID,
	) error
	OpenContentByActor(
		context.Context,
		domain.DocumentID,
		*storage.ByteRange,
		domain.MemberID,
	) (vault.Content, error)
}

type actorVaultLinkService interface {
	ContentLinkByActor(
		context.Context,
		domain.DocumentID,
		bool,
		domain.MemberID,
	) (string, error)
}

func registerVaultRoutes(
	mux *http.ServeMux,
	identityService IdentityService,
	vaultService VaultService,
	recorder audit.Recorder,
) {
	memberRoute := func(handler http.HandlerFunc) http.Handler {
		return requireMember(
			identityService,
			recorder,
			requireCSRF(identityService, recorder, handler),
		)
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
	mux.Handle("POST /api/v1/documents/{id}/restore", requireAdminWithCSRF(Options{
		Identity: identityService,
		Audit:    recorder,
	}, http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			handleDocumentRestore(w, r, vaultService)
		},
	)))
	mux.Handle("DELETE /api/v1/documents/{id}/purge", requireAdminWithCSRF(Options{
		Identity: identityService,
		Audit:    recorder,
	}, http.HandlerFunc(
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
	documentID := domain.DocumentID(r.PathValue("id"))
	var err error
	if actorService, ok := service.(actorVaultService); ok {
		err = actorService.DeleteByActor(
			r.Context(),
			documentID,
			memberFromContext(r.Context()).ID,
		)
	} else {
		err = service.Delete(r.Context(), documentID)
	}
	if err != nil {
		writeVaultError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleDocumentRestore(w http.ResponseWriter, r *http.Request, service VaultService) {
	documentID := domain.DocumentID(r.PathValue("id"))
	var record vault.DocumentRecord
	var err error
	if actorService, ok := service.(actorVaultService); ok {
		record, err = actorService.RestoreByActor(
			r.Context(),
			documentID,
			memberFromContext(r.Context()).ID,
		)
	} else {
		record, err = service.Restore(r.Context(), documentID)
	}
	if err != nil {
		writeVaultError(w, r, err)
		return
	}
	writeRecord(w, http.StatusOK, record)
}

func handleDocumentPurge(w http.ResponseWriter, r *http.Request, service VaultService) {
	documentID := domain.DocumentID(r.PathValue("id"))
	var err error
	if actorService, ok := service.(actorVaultService); ok {
		err = actorService.PurgeByActor(
			r.Context(),
			documentID,
			vault.EnforceRetention,
			memberFromContext(r.Context()).ID,
		)
	} else {
		err = service.Purge(r.Context(), documentID, vault.EnforceRetention)
	}
	if err != nil {
		writeVaultError(w, r, err)
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
		writeVaultError(w, r, err)
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
		writeVaultError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"documents": records})
}

func handleDocumentGet(w http.ResponseWriter, r *http.Request, service VaultService) {
	record, err := service.Get(r.Context(), domain.DocumentID(r.PathValue("id")))
	if err != nil {
		writeVaultError(w, r, err)
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
			ActorID:         memberIDPointer(memberFromContext(r.Context()).ID),
		},
	)
	if err != nil {
		writeVaultError(w, r, err)
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
	if byteRange == nil {
		download := r.URL.Query().Get("download") == "true"
		if linkService, ok := service.(actorVaultLinkService); ok {
			link, linkErr := linkService.ContentLinkByActor(
				r.Context(),
				domain.DocumentID(r.PathValue("id")),
				download,
				memberFromContext(r.Context()).ID,
			)
			if linkErr == nil {
				http.Redirect(w, r, link, http.StatusFound)
				return
			}
			if !errors.Is(linkErr, vault.ErrContentLinkUnavailable) {
				writeVaultError(w, r, linkErr)
				return
			}
		}
	}
	documentID := domain.DocumentID(r.PathValue("id"))
	var content vault.Content
	if actorService, ok := service.(actorVaultService); ok {
		content, err = actorService.OpenContentByActor(
			r.Context(),
			documentID,
			byteRange,
			memberFromContext(r.Context()).ID,
		)
	} else {
		content, err = service.OpenContent(r.Context(), documentID, byteRange)
	}
	if err != nil {
		writeVaultError(w, r, err)
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
		writeVaultError(w, r, err)
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
		writeVaultError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, tag)
}

func writeRecord(w http.ResponseWriter, status int, record vault.DocumentRecord) {
	w.Header().Set("ETag", fmt.Sprintf("\"%d\"", record.Document.Version))
	writeJSON(w, status, record)
}

func writeVaultError(w http.ResponseWriter, r *http.Request, err error) {
	status := http.StatusInternalServerError
	code := "vault_error"
	switch {
	case errors.Is(err, domain.ErrNotFound), errors.Is(err, storage.ErrNotFound):
		status, code = http.StatusNotFound, "not_found"
	case errors.Is(err, domain.ErrConflict):
		status, code = http.StatusPreconditionFailed, "version_conflict"
	case errors.Is(err, vault.ErrDocumentTooLarge):
		status, code = http.StatusRequestEntityTooLarge, "document_too_large"
	case errors.Is(err, vault.ErrUnsupportedMediaType):
		status, code = http.StatusUnsupportedMediaType, "unsupported_media_type"
	case errors.Is(err, vault.ErrInvalidMetadata):
		status, code = http.StatusBadRequest, "invalid_request"
	case errors.Is(err, vault.ErrInvalidRange):
		status, code = http.StatusRequestedRangeNotSatisfiable, "invalid_range"
	case errors.Is(err, vault.ErrUploadInProgress):
		status, code = http.StatusConflict, "upload_in_progress"
	case errors.Is(err, vault.ErrRetentionActive):
		status, code = http.StatusConflict, "retention_active"
	}
	if status >= http.StatusInternalServerError {
		if logger := requestLoggerFromContext(r.Context()); logger != nil {
			logger.Printf(
				"vault request failed method=%s path=%s error=%v",
				r.Method,
				r.URL.Path,
				err,
			)
		}
	}
	writeError(w, status, code)
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
