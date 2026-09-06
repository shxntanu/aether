package vault

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/shxntanu/aether/backend/internal/audit"
	"github.com/shxntanu/aether/backend/internal/domain"
	"github.com/shxntanu/aether/backend/internal/storage"
)

const (
	// MaxDocumentSize is the largest accepted original document size.
	MaxDocumentSize int64 = 50 * 1024 * 1024
	maxTagCount           = 50
	maxTagLength          = 100
	maxTitleLength        = 500
)

var (
	// ErrUnsupportedMediaType indicates that file signatures do not identify
	// an allowed PDF, JPEG, PNG, or WebP document.
	ErrUnsupportedMediaType = errors.New("unsupported document media type")
	// ErrDocumentTooLarge indicates that an upload exceeds MaxDocumentSize.
	ErrDocumentTooLarge = errors.New("document exceeds maximum size")
	// ErrInvalidMetadata indicates invalid title, tag, or filter input.
	ErrInvalidMetadata = errors.New("invalid document metadata")
	// ErrInvalidRange indicates an unsatisfiable content byte range.
	ErrInvalidRange = errors.New("invalid document byte range")
	// ErrUploadInProgress indicates that an idempotent upload has not completed.
	ErrUploadInProgress = errors.New("idempotent upload is still in progress")
	// ErrContentLinkUnavailable indicates that the configured provider has no
	// provider-hosted link capability, so content must be streamed through Aether.
	ErrContentLinkUnavailable = errors.New("provider content link unavailable")
)

// Repository contains catalog operations required by the tagged vault.
type Repository interface {
	domain.Repository
}

type searchRepository interface {
	SearchDocuments(
		context.Context,
		string,
		int,
	) ([]domain.DocumentSearchResult, error)
}

// Service coordinates catalog records, immutable objects, and manifests.
type Service struct {
	repository Repository
	objects    storage.ObjectStore
	now        func() time.Time
	newID      func() (string, error)
	recorder   audit.Recorder
}

// DocumentRecord combines catalog metadata with its reusable tags.
type DocumentRecord struct {
	// Document is the public catalog metadata.
	Document domain.Document `json:"document"`
	// Tags contains the document's reusable tags.
	Tags []domain.Tag `json:"tags"`
}

// DocumentSearchResult combines a ranked document record with typed visible
// evidence explaining why it matched.
type DocumentSearchResult struct {
	// Document is the public catalog metadata.
	Document domain.Document `json:"document"`
	// Tags contains every reusable tag attached to the document.
	Tags []domain.Tag `json:"tags"`
	// Evidence identifies matched titles, filenames, or tags.
	Evidence []domain.DocumentSearchEvidence `json:"evidence"`
}

// Upload describes a streamed original document and optional metadata.
type Upload struct {
	// Filename is the untrusted client filename used for display.
	Filename string
	// Body streams the original bytes.
	Body io.Reader
	// Title optionally overrides filename-based title derivation.
	Title string
	// Tags contains optional reusable display names.
	Tags []string
	// Uploader is the authenticated member creating the document.
	Uploader domain.MemberID
	// IdempotencyKey scopes replay protection to Uploader.
	IdempotencyKey string
}

// MetadataUpdate describes mutable document metadata guarded by a version.
type MetadataUpdate struct {
	// Title is the new non-empty display title.
	Title string
	// Tags completely replaces current tag associations.
	Tags []string
	// ExpectedVersion must match the current catalog version.
	ExpectedVersion int64
	// ActorID identifies the member changing the metadata, when known.
	ActorID *domain.MemberID
}

// Content is an opened document body and the metadata needed for HTTP ranges.
type Content struct {
	// Body is the content stream and must be closed by the caller.
	Body io.ReadCloser
	// Document describes the opened original.
	Document domain.Document
	// ByteRange is resolved, or nil for the complete original.
	ByteRange *storage.ByteRange
}

// NewService creates a vault service using the supplied catalog, object store,
// and optional audit recorder. A missing recorder makes successful mutations
// fail closed after their required storage work completes.
func NewService(
	repository Repository,
	objects storage.ObjectStore,
	recorders ...audit.Recorder,
) *Service {
	var recorder audit.Recorder
	if len(recorders) > 0 {
		recorder = recorders[0]
	}
	return &Service{
		repository: repository,
		objects:    objects,
		now:        time.Now,
		newID:      randomID,
		recorder:   recorder,
	}
}

// Upload stores an original document, optional tags, and a versioned manifest.
func (s *Service) Upload(ctx context.Context, input Upload) (DocumentRecord, error) {
	filename := filepath.Base(strings.ReplaceAll(strings.TrimSpace(input.Filename), "\\", "/"))
	idempotencyKey := strings.TrimSpace(input.IdempotencyKey)
	if filename == "" || filename == "." || input.Body == nil ||
		len(filename) > 255 || !validDisplayText(filename) ||
		idempotencyKey == "" || len(idempotencyKey) > 200 {
		return DocumentRecord{}, ErrInvalidMetadata
	}
	title := strings.TrimSpace(input.Title)
	if title == "" {
		title = strings.TrimSpace(strings.TrimSuffix(filename, filepath.Ext(filename)))
	}
	if title == "" || len(title) > maxTitleLength || !validDisplayText(title) {
		return DocumentRecord{}, ErrInvalidMetadata
	}
	if err := validateTagNames(input.Tags); err != nil {
		return DocumentRecord{}, err
	}

	header := make([]byte, 512)
	read, readErr := io.ReadFull(input.Body, header)
	if readErr != nil && !errors.Is(readErr, io.EOF) && !errors.Is(readErr, io.ErrUnexpectedEOF) {
		return DocumentRecord{}, fmt.Errorf("read upload signature: %w", readErr)
	}
	header = header[:read]
	mediaType, err := detectMediaType(header)
	if err != nil {
		return DocumentRecord{}, err
	}

	id, err := s.newID()
	if err != nil {
		return DocumentRecord{}, err
	}
	now := s.now().UTC()
	document := domain.Document{
		ID:               domain.DocumentID(id),
		Title:            title,
		OriginalFilename: filename,
		MediaType:        mediaType,
		StorageKey:       "documents/" + id + "/original",
		Status:           domain.DocumentStatusUploading,
		IndexStatus:      domain.IndexStatusNotScheduled,
		UploaderID:       input.Uploader,
		Version:          1,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	keyHash := sha256.Sum256([]byte(idempotencyKey))
	var existingID domain.DocumentID
	var claimed bool
	err = s.repository.WithinTransaction(ctx, func(transaction domain.Repository) error {
		existingID, claimed, err = transaction.ClaimUpload(
			ctx,
			input.Uploader,
			hex.EncodeToString(keyHash[:]),
			document.ID,
			now,
		)
		if err != nil || !claimed {
			return err
		}
		return transaction.CreateDocument(ctx, document)
	})
	if err != nil {
		return DocumentRecord{}, err
	}
	if !claimed {
		existing, getErr := s.Get(ctx, existingID)
		if errors.Is(getErr, domain.ErrNotFound) {
			return DocumentRecord{}, ErrUploadInProgress
		}
		return existing, getErr
	}
	hash := sha256.New()
	limited := &io.LimitedReader{
		R: io.MultiReader(bytes.NewReader(header), input.Body),
		N: MaxDocumentSize + 1,
	}
	info, putErr := s.objects.Put(
		ctx,
		document.StorageKey,
		io.TeeReader(limited, hash),
		storage.PutOptions{ContentType: mediaType},
	)
	if putErr != nil {
		s.markFailed(ctx, document)
		return DocumentRecord{}, fmt.Errorf("store document: %w", putErr)
	}
	if info.Size > MaxDocumentSize {
		_ = s.objects.Delete(ctx, document.StorageKey)
		s.markFailed(ctx, document)
		return DocumentRecord{}, ErrDocumentTooLarge
	}

	original := document
	var tags []domain.Tag
	err = s.repository.WithinTransaction(ctx, func(transaction domain.Repository) error {
		document.SizeBytes = info.Size
		document.SHA256 = hex.EncodeToString(hash.Sum(nil))
		document.Status = domain.DocumentStatusReady
		document.UpdatedAt = s.now().UTC()
		document, err = transaction.UpdateDocument(ctx, document, document.Version)
		if err != nil {
			return err
		}
		tags, err = s.resolveAndReplaceTags(ctx, transaction, document.ID, input.Tags)
		return err
	})
	if err != nil {
		_ = s.objects.Delete(ctx, document.StorageKey)
		s.markFailed(ctx, original)
		return DocumentRecord{}, err
	}
	document = s.writeManifest(ctx, document, tags)
	actorID := input.Uploader
	if err := s.recordDocumentEvent(
		ctx,
		audit.ActionDocumentUpload,
		document.ID,
		memberIDPointer(actorID),
	); err != nil {
		return DocumentRecord{}, fmt.Errorf("record document upload: %w", err)
	}
	return DocumentRecord{Document: document, Tags: tags}, nil
}

// Get returns one ready document and its tags.
func (s *Service) Get(ctx context.Context, id domain.DocumentID) (DocumentRecord, error) {
	document, err := s.repository.GetDocument(ctx, id)
	if err != nil {
		return DocumentRecord{}, err
	}
	if document.Status != domain.DocumentStatusReady {
		return DocumentRecord{}, domain.ErrNotFound
	}
	tags, err := s.repository.ListDocumentTags(ctx, id)
	if err != nil {
		return DocumentRecord{}, err
	}
	return DocumentRecord{Document: document, Tags: tags}, nil
}

// List returns ready documents matching all or any normalized tag names.
func (s *Service) List(
	ctx context.Context,
	tags []string,
	match domain.TagMatch,
) ([]DocumentRecord, error) {
	if match == "" {
		match = domain.TagMatchAll
	}
	if match != domain.TagMatchAll && match != domain.TagMatchAny {
		return nil, ErrInvalidMetadata
	}
	normalized, err := normalizeTagFilters(tags)
	if err != nil {
		return nil, err
	}
	documents, err := s.repository.ListDocuments(ctx, domain.DocumentListOptions{
		NormalizedTags: normalized,
		TagMatch:       match,
	})
	if err != nil {
		return nil, err
	}
	records := make([]DocumentRecord, 0, len(documents))
	for _, document := range documents {
		documentTags, err := s.repository.ListDocumentTags(ctx, document.ID)
		if err != nil {
			return nil, err
		}
		records = append(records, DocumentRecord{Document: document, Tags: documentTags})
	}
	return records, nil
}

// Search returns recent ready documents for an empty query or ranked metadata
// matches for a non-empty query.
func (s *Service) Search(
	ctx context.Context,
	query string,
	limit int,
) ([]DocumentSearchResult, error) {
	normalizedQuery := strings.TrimSpace(query)
	runes := []rune(normalizedQuery)
	if len(runes) > 200 {
		normalizedQuery = string(runes[:200])
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 20 {
		limit = 20
	}
	repository, ok := s.repository.(searchRepository)
	if !ok {
		return nil, errors.New("document search is not configured")
	}
	results, err := repository.SearchDocuments(ctx, normalizedQuery, limit)
	if err != nil {
		return nil, err
	}
	searchResults := make([]DocumentSearchResult, len(results))
	for index, result := range results {
		searchResults[index] = DocumentSearchResult{
			Document: result.Document,
			Tags:     result.Tags,
			Evidence: result.Evidence,
		}
	}
	return searchResults, nil
}

// ListDeleted returns soft-deleted documents and their tags for Trash views.
func (s *Service) ListDeleted(
	ctx context.Context,
	tags []string,
	match domain.TagMatch,
) ([]DocumentRecord, error) {
	if match == "" {
		match = domain.TagMatchAll
	}
	if match != domain.TagMatchAll && match != domain.TagMatchAny {
		return nil, ErrInvalidMetadata
	}
	normalized, err := normalizeTagFilters(tags)
	if err != nil {
		return nil, err
	}
	documents, err := s.repository.ListDocuments(ctx, domain.DocumentListOptions{
		NormalizedTags: normalized,
		TagMatch:       match,
		Statuses:       []domain.DocumentStatus{domain.DocumentStatusDeleted},
	})
	if err != nil {
		return nil, err
	}
	records := make([]DocumentRecord, 0, len(documents))
	for _, document := range documents {
		record, recordErr := s.deletedRecord(ctx, document)
		if recordErr != nil {
			return nil, recordErr
		}
		records = append(records, record)
	}
	return records, nil
}

// UpdateMetadata changes title and tags if the supplied version is current.
func (s *Service) UpdateMetadata(
	ctx context.Context,
	id domain.DocumentID,
	input MetadataUpdate,
) (DocumentRecord, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" || len(title) > maxTitleLength || !validDisplayText(title) ||
		input.ExpectedVersion <= 0 {
		return DocumentRecord{}, ErrInvalidMetadata
	}
	if err := validateTagNames(input.Tags); err != nil {
		return DocumentRecord{}, err
	}
	var document domain.Document
	var tags []domain.Tag
	err := s.repository.WithinTransaction(ctx, func(transaction domain.Repository) error {
		var transactionErr error
		document, transactionErr = transaction.GetDocument(ctx, id)
		if transactionErr != nil {
			return transactionErr
		}
		if document.Status != domain.DocumentStatusReady {
			return domain.ErrNotFound
		}
		document.Title = title
		document.ManifestError = ""
		document.UpdatedAt = s.now().UTC()
		document, transactionErr = transaction.UpdateDocument(
			ctx,
			document,
			input.ExpectedVersion,
		)
		if transactionErr != nil {
			return transactionErr
		}
		tags, transactionErr = s.resolveAndReplaceTags(
			ctx,
			transaction,
			id,
			input.Tags,
		)
		return transactionErr
	})
	if err != nil {
		return DocumentRecord{}, err
	}
	document = s.writeManifest(ctx, document, tags)
	if err := s.recordDocumentEvent(
		ctx,
		audit.ActionDocumentEdit,
		document.ID,
		input.ActorID,
	); err != nil {
		return DocumentRecord{}, fmt.Errorf("record document metadata edit: %w", err)
	}
	return DocumentRecord{Document: document, Tags: tags}, nil
}

// ListTags returns reusable tags for autocomplete.
func (s *Service) ListTags(ctx context.Context, query string, limit int) ([]domain.Tag, error) {
	if len(query) > maxTagLength {
		return nil, ErrInvalidMetadata
	}
	return s.repository.ListTags(ctx, query, limit)
}

// CreateTag creates or returns a reusable tag with case-insensitive identity.
func (s *Service) CreateTag(ctx context.Context, name string) (domain.Tag, error) {
	if err := validateTagNames([]string{name}); err != nil {
		return domain.Tag{}, err
	}
	tag, _ := domain.NewTag("", name)
	existing, err := s.repository.GetTagByNormalizedName(ctx, tag.NormalizedName)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return domain.Tag{}, err
	}
	id, err := s.newID()
	if err != nil {
		return domain.Tag{}, err
	}
	tag.ID = domain.TagID(id)
	if err := s.repository.CreateTag(ctx, tag); err != nil {
		if errors.Is(err, domain.ErrAlreadyExists) {
			return s.repository.GetTagByNormalizedName(ctx, tag.NormalizedName)
		}
		return domain.Tag{}, err
	}
	return tag, nil
}

// UpdateTag renames a reusable tag while preserving existing document associations.
func (s *Service) UpdateTag(
	ctx context.Context,
	id domain.TagID,
	name string,
) (domain.Tag, error) {
	if id == "" || validateTagNames([]string{name}) != nil {
		return domain.Tag{}, ErrInvalidMetadata
	}
	tag, _ := domain.NewTag(id, name)
	return s.repository.UpdateTag(ctx, tag)
}

// DeleteTag removes a reusable tag and all of its document associations.
func (s *Service) DeleteTag(ctx context.Context, id domain.TagID) error {
	if id == "" {
		return ErrInvalidMetadata
	}
	return s.repository.DeleteTag(ctx, id)
}

// OpenContent opens a ready document, optionally restricted to a byte range.
func (s *Service) OpenContent(
	ctx context.Context,
	id domain.DocumentID,
	byteRange *storage.ByteRange,
) (Content, error) {
	document, err := s.repository.GetDocument(ctx, id)
	if err != nil {
		return Content{}, err
	}
	if document.Status != domain.DocumentStatusReady {
		return Content{}, domain.ErrNotFound
	}
	resolvedRange, err := resolveRange(byteRange, document.SizeBytes)
	if err != nil {
		return Content{}, err
	}
	body, _, err := s.objects.OpenRange(ctx, document.StorageKey, resolvedRange)
	if err != nil {
		return Content{}, err
	}
	return Content{Body: body, Document: document, ByteRange: resolvedRange}, nil
}

// OpenContentByActor opens content and records the successful document access.
// The opened body is closed if recording fails so callers cannot leak it.
func (s *Service) OpenContentByActor(
	ctx context.Context,
	id domain.DocumentID,
	byteRange *storage.ByteRange,
	actorID domain.MemberID,
) (Content, error) {
	content, err := s.OpenContent(ctx, id, byteRange)
	if err != nil {
		return Content{}, err
	}
	if err := s.recordDocumentEvent(
		ctx,
		audit.ActionDocumentDownload,
		content.Document.ID,
		memberIDPointer(actorID),
	); err != nil {
		_ = content.Body.Close()
		return Content{}, fmt.Errorf("record document download: %w", err)
	}
	return content, nil
}

// ContentLink returns a provider-hosted view or download link for a ready
// document. Providers without this optional capability return
// ErrContentLinkUnavailable.
func (s *Service) ContentLink(
	ctx context.Context,
	id domain.DocumentID,
	download bool,
) (string, error) {
	document, err := s.repository.GetDocument(ctx, id)
	if err != nil {
		return "", err
	}
	if document.Status != domain.DocumentStatusReady {
		return "", domain.ErrNotFound
	}
	linker, ok := s.objects.(storage.ObjectLinker)
	if !ok {
		return "", ErrContentLinkUnavailable
	}
	links, err := linker.Links(ctx, document.StorageKey)
	if err != nil {
		return "", err
	}
	if download {
		if links.DownloadURL == "" {
			return "", ErrContentLinkUnavailable
		}
		return links.DownloadURL, nil
	}
	if links.ViewURL == "" {
		return "", ErrContentLinkUnavailable
	}
	return links.ViewURL, nil
}

// ContentLinkByActor returns a provider-hosted link and audits the successful
// access without opening or proxying the document body.
func (s *Service) ContentLinkByActor(
	ctx context.Context,
	id domain.DocumentID,
	download bool,
	actorID domain.MemberID,
) (string, error) {
	link, err := s.ContentLink(ctx, id, download)
	if err != nil {
		return "", err
	}
	if err := s.recordDocumentEvent(
		ctx,
		audit.ActionDocumentDownload,
		domain.DocumentID(id),
		memberIDPointer(actorID),
	); err != nil {
		return "", fmt.Errorf("record document link access: %w", err)
	}
	return link, nil
}

// DeleteByActor soft-deletes a document and records the successful operation.
func (s *Service) DeleteByActor(
	ctx context.Context,
	id domain.DocumentID,
	actorID domain.MemberID,
) (DocumentRecord, error) {
	record, err := s.Delete(ctx, id)
	if err != nil {
		return DocumentRecord{}, err
	}
	if err := s.recordDocumentEvent(
		ctx,
		audit.ActionDocumentDelete,
		id,
		memberIDPointer(actorID),
	); err != nil {
		return DocumentRecord{}, fmt.Errorf("record document deletion: %w", err)
	}
	return record, nil
}

// RestoreByActor restores a document and records the successful operation.
func (s *Service) RestoreByActor(
	ctx context.Context,
	id domain.DocumentID,
	actorID domain.MemberID,
) (DocumentRecord, error) {
	record, err := s.Restore(ctx, id)
	if err != nil {
		return DocumentRecord{}, err
	}
	if err := s.recordDocumentEvent(
		ctx,
		audit.ActionDocumentRestore,
		id,
		memberIDPointer(actorID),
	); err != nil {
		return DocumentRecord{}, fmt.Errorf("record document restoration: %w", err)
	}
	return record, nil
}

// PurgeByActor permanently removes a document and records the successful
// operation after both storage objects and the catalog row are gone.
func (s *Service) PurgeByActor(
	ctx context.Context,
	id domain.DocumentID,
	policy PurgePolicy,
	actorID domain.MemberID,
) error {
	if err := s.Purge(ctx, id, policy); err != nil {
		return err
	}
	if err := s.recordDocumentEvent(
		ctx,
		audit.ActionDocumentPurge,
		id,
		memberIDPointer(actorID),
	); err != nil {
		return fmt.Errorf("record document purge: %w", err)
	}
	return nil
}

func (s *Service) recordDocumentEvent(
	ctx context.Context,
	action audit.Action,
	id domain.DocumentID,
	actorID *domain.MemberID,
) error {
	if s.recorder == nil {
		return errors.New("audit recorder is not configured")
	}
	return s.recorder.Record(ctx, audit.Event{
		ActorID:    actorID,
		Action:     action,
		ObjectType: "document",
		ObjectID:   string(id),
		Outcome:    domain.AuditOutcomeSucceeded,
	})
}

func memberIDPointer(id domain.MemberID) *domain.MemberID {
	if id == "" {
		return nil
	}
	copy := id
	return &copy
}

func (s *Service) resolveAndReplaceTags(
	ctx context.Context,
	repository domain.Repository,
	documentID domain.DocumentID,
	names []string,
) ([]domain.Tag, error) {
	tags := make([]domain.Tag, 0, len(names))
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		tag, err := domain.NewTag("", name)
		if err != nil {
			return nil, ErrInvalidMetadata
		}
		if _, exists := seen[tag.NormalizedName]; exists {
			continue
		}
		seen[tag.NormalizedName] = struct{}{}
		existing, err := repository.GetTagByNormalizedName(ctx, tag.NormalizedName)
		if errors.Is(err, domain.ErrNotFound) {
			id, idErr := s.newID()
			if idErr != nil {
				return nil, idErr
			}
			tag.ID = domain.TagID(id)
			if createErr := repository.CreateTag(ctx, tag); createErr != nil {
				if !errors.Is(createErr, domain.ErrAlreadyExists) {
					return nil, createErr
				}
				tag, err = repository.GetTagByNormalizedName(ctx, tag.NormalizedName)
				if err != nil {
					return nil, err
				}
			}
		} else if err != nil {
			return nil, err
		} else {
			tag = existing
		}
		tags = append(tags, tag)
	}
	tagIDs := make([]domain.TagID, len(tags))
	for index, tag := range tags {
		tagIDs[index] = tag.ID
	}
	if err := repository.ReplaceDocumentTags(ctx, documentID, tagIDs); err != nil {
		return nil, err
	}
	return tags, nil
}

func (s *Service) writeManifest(
	ctx context.Context,
	document domain.Document,
	tags []domain.Tag,
) domain.Document {
	manifest := struct {
		SchemaVersion int             `json:"schemaVersion"`
		Document      domain.Document `json:"document"`
		Tags          []domain.Tag    `json:"tags"`
	}{SchemaVersion: 1, Document: document, Tags: tags}
	body, err := json.MarshalIndent(manifest, "", "  ")
	if err == nil {
		_, err = s.objects.Put(
			ctx,
			document.StorageKey+".manifest.json",
			bytes.NewReader(body),
			storage.PutOptions{ContentType: "application/json"},
		)
	}
	if err == nil {
		return document
	}
	document.ManifestError = err.Error()
	document.UpdatedAt = s.now().UTC()
	updated, updateErr := s.repository.UpdateDocument(ctx, document, document.Version)
	if updateErr == nil {
		return updated
	}
	document.ManifestError += "; persist manifest failure: " + updateErr.Error()
	return document
}

func (s *Service) markFailed(ctx context.Context, document domain.Document) {
	document.Status = domain.DocumentStatusFailed
	document.UpdatedAt = s.now().UTC()
	_, _ = s.repository.UpdateDocument(ctx, document, document.Version)
}

func detectMediaType(header []byte) (string, error) {
	switch {
	case bytes.HasPrefix(header, []byte("%PDF-")):
		return "application/pdf", nil
	case len(header) >= 3 && header[0] == 0xff && header[1] == 0xd8 && header[2] == 0xff:
		return "image/jpeg", nil
	case bytes.HasPrefix(header, []byte("\x89PNG\r\n\x1a\n")):
		return "image/png", nil
	case len(header) >= 12 && string(header[:4]) == "RIFF" && string(header[8:12]) == "WEBP":
		return "image/webp", nil
	default:
		return "", ErrUnsupportedMediaType
	}
}

func validateTagNames(names []string) error {
	if len(names) > maxTagCount {
		return ErrInvalidMetadata
	}
	for _, name := range names {
		if len(name) > maxTagLength || !validDisplayText(name) {
			return ErrInvalidMetadata
		}
		if _, err := domain.NewTag("", name); err != nil {
			return ErrInvalidMetadata
		}
	}
	return nil
}

func validDisplayText(value string) bool {
	for _, character := range value {
		if character < 0x20 || character == 0x7f {
			return false
		}
	}
	return true
}

func normalizeTagFilters(names []string) ([]string, error) {
	if err := validateTagNames(names); err != nil {
		return nil, err
	}
	result := make([]string, 0, len(names))
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		tag, _ := domain.NewTag("", name)
		if _, exists := seen[tag.NormalizedName]; !exists {
			seen[tag.NormalizedName] = struct{}{}
			result = append(result, tag.NormalizedName)
		}
	}
	return result, nil
}

func resolveRange(byteRange *storage.ByteRange, size int64) (*storage.ByteRange, error) {
	if byteRange == nil {
		return nil, nil
	}
	if size <= 0 {
		return nil, ErrInvalidRange
	}
	resolved := *byteRange
	switch {
	case resolved.Start == -1:
		if resolved.End <= 0 {
			return nil, ErrInvalidRange
		}
		if resolved.End > size {
			resolved.End = size
		}
		resolved.Start = size - resolved.End
		resolved.End = size - 1
	case resolved.End == -1:
		resolved.End = size - 1
	}
	if resolved.Start < 0 || resolved.End < resolved.Start || resolved.End >= size {
		return nil, ErrInvalidRange
	}
	return &resolved, nil
}

func randomID() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("generate identifier: %w", err)
	}
	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(value)
	return encoded[0:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" +
		encoded[16:20] + "-" + encoded[20:32], nil
}
