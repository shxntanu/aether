package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shxntanu/aether/backend/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CreateDocument persists a new document catalog record.
func (s *Store) CreateDocument(ctx context.Context, document domain.Document) error {
	model := documentModelFromDomain(document)
	err := s.orm.WithContext(ctx).Create(&model).Error
	return translateError("create document", err)
}

// GetDocument returns a document by its catalog identifier.
func (s *Store) GetDocument(ctx context.Context, id domain.DocumentID) (domain.Document, error) {
	var model documentModel
	err := s.orm.WithContext(ctx).First(&model, "id = ?", id).Error
	if isRecordNotFound(err) {
		return domain.Document{}, fmt.Errorf("get document %q: %w", id, domain.ErrNotFound)
	}
	if err != nil {
		return domain.Document{}, fmt.Errorf("get document %q: %w", id, err)
	}
	return model.domain(), nil
}

// UpdateDocument applies an optimistic-concurrency catalog update.
func (s *Store) UpdateDocument(
	ctx context.Context,
	document domain.Document,
	expectedVersion int64,
) (domain.Document, error) {
	updates := map[string]any{
		"title":             document.Title,
		"original_filename": document.OriginalFilename,
		"media_type":        document.MediaType,
		"size_bytes":        document.SizeBytes,
		"sha256":            document.SHA256,
		"storage_key":       document.StorageKey,
		"status":            document.Status,
		"index_status":      document.IndexStatus,
		"uploader_id":       document.UploaderID,
		"updated_at":        document.UpdatedAt,
		"deleted_at":        document.DeletedAt,
		"purge_after":       document.PurgeAfter,
		"deletion_status":   document.DeletionStatus,
		"deletion_error":    document.DeletionError,
		"manifest_error":    document.ManifestError,
		"version":           gorm.Expr("version + ?", 1),
	}
	result := s.orm.WithContext(ctx).
		Model(&documentModel{}).
		Where("id = ? AND version = ?", document.ID, expectedVersion).
		Updates(updates)
	if result.Error != nil {
		return domain.Document{}, translateError("update document", result.Error)
	}
	if result.RowsAffected == 0 {
		if _, err := s.GetDocument(ctx, document.ID); errors.Is(err, domain.ErrNotFound) {
			return domain.Document{}, err
		}
		return domain.Document{}, fmt.Errorf("update document %q: %w", document.ID, domain.ErrConflict)
	}
	return s.GetDocument(ctx, document.ID)
}

// ListDocuments returns documents matching validated lifecycle and tag filters.
func (s *Store) ListDocuments(
	ctx context.Context,
	options domain.DocumentListOptions,
) ([]domain.Document, error) {
	queryPlan, err := buildDocumentListQuery(options)
	if err != nil {
		return nil, err
	}

	query := s.orm.WithContext(ctx).Model(&documentModel{})
	if queryPlan.statusClause == "FALSE" {
		query = query.Where("FALSE")
	} else {
		query = query.Where(queryPlan.statusClause)
	}
	if queryPlan.purgeDueBefore != nil {
		query = query.Where(
			"purge_after IS NOT NULL AND purge_after <= ?",
			queryPlan.purgeDueBefore.UTC(),
		)
	}
	if len(options.NormalizedTags) > 0 {
		tags := s.orm.WithContext(ctx).
			Model(&documentTagModel{}).
			Select("document_id").
			Joins("JOIN tags_tbl AS t ON t.id = document_tags_tbl.tag_id").
			Where("t.normalized_name IN ?", options.NormalizedTags).
			Group("document_id")
		if queryPlan.tagMatch != domain.TagMatchAny {
			tags = tags.Having("COUNT(DISTINCT t.id) = ?", len(options.NormalizedTags))
		}
		query = query.Where("id IN (?)", tags)
	}
	if queryPlan.purgeDueBefore != nil {
		query = query.Order("purge_after ASC").Order("id ASC")
	} else {
		query = query.Order("created_at DESC").Order("id DESC")
	}
	if queryPlan.limit > 0 {
		query = query.Limit(queryPlan.limit)
	}

	var models []documentModel
	if err := query.Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}
	documents := make([]domain.Document, 0, len(models))
	for _, model := range models {
		documents = append(documents, model.domain())
	}
	return documents, nil
}

// PurgeDocument permanently removes a deleted catalog row and its cascading
// document-tag join rows.
func (s *Store) PurgeDocument(ctx context.Context, id domain.DocumentID) error {
	result := s.orm.WithContext(ctx).
		Where("id = ? AND status = ?", id, domain.DocumentStatusDeleted).
		Delete(&documentModel{})
	if result.Error != nil {
		return fmt.Errorf("purge document %q: %w", id, result.Error)
	}
	if result.RowsAffected > 0 {
		return nil
	}

	var status struct {
		Status domain.DocumentStatus `gorm:"column:status"`
	}
	err := s.orm.WithContext(ctx).
		Model(&documentModel{}).
		Select("status").
		Where("id = ?", id).
		First(&status).Error
	if isRecordNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect document purge target %q: %w", id, err)
	}
	return fmt.Errorf("purge document %q: %w", id, domain.ErrConflict)
}

// ClaimUpload atomically binds an idempotency digest to one document ID.
func (s *Store) ClaimUpload(
	ctx context.Context,
	uploaderID domain.MemberID,
	keyHash string,
	documentID domain.DocumentID,
	createdAt time.Time,
) (domain.DocumentID, bool, error) {
	request := uploadRequestModel{
		MemberID:   uploaderID,
		KeyHash:    keyHash,
		DocumentID: documentID,
		CreatedAt:  createdAt,
	}
	result := s.orm.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&request)
	if result.Error != nil {
		return "", false, fmt.Errorf("claim upload: %w", result.Error)
	}
	if result.RowsAffected == 1 {
		return documentID, true, nil
	}

	var existing uploadRequestModel
	err := s.orm.WithContext(ctx).
		Where("member_id = ? AND key_hash = ?", uploaderID, keyHash).
		First(&existing).Error
	if err != nil {
		return "", false, fmt.Errorf("get upload claim: %w", err)
	}
	return existing.DocumentID, false, nil
}

type documentListQuery struct {
	statusClause   string
	tagMatch       domain.TagMatch
	purgeDueBefore *time.Time
	limit          int
}

func buildDocumentListQuery(
	options domain.DocumentListOptions,
) (documentListQuery, error) {
	if options.Limit < 0 {
		return documentListQuery{}, fmt.Errorf("list documents: limit must be non-negative")
	}

	tagMatch := options.TagMatch
	if tagMatch == "" {
		tagMatch = domain.TagMatchAll
	}
	if tagMatch != domain.TagMatchAll && tagMatch != domain.TagMatchAny {
		return documentListQuery{}, fmt.Errorf(
			"list documents: invalid tag match %q",
			options.TagMatch,
		)
	}

	statusClause, includesDeleted, err := buildDocumentStatusClause(options.Statuses)
	if err != nil {
		return documentListQuery{}, err
	}
	if options.PurgeDueBefore != nil && !includesDeleted {
		return documentListQuery{}, fmt.Errorf(
			"list documents: purge due filter requires deleted status",
		)
	}

	return documentListQuery{
		statusClause:   statusClause,
		tagMatch:       tagMatch,
		purgeDueBefore: options.PurgeDueBefore,
		limit:          options.Limit,
	}, nil
}

func buildDocumentStatusClause(statuses []domain.DocumentStatus) (string, bool, error) {
	if statuses == nil {
		return "status = 'ready'", false, nil
	}
	if len(statuses) == 0 {
		return "FALSE", false, nil
	}

	clauses := make([]string, 0, len(statuses))
	seen := make(map[domain.DocumentStatus]struct{}, len(statuses))
	includesDeleted := false
	for _, status := range statuses {
		if _, ok := seen[status]; ok {
			continue
		}
		seen[status] = struct{}{}

		clause, err := documentStatusClause(status)
		if err != nil {
			return "", false, err
		}
		if status == domain.DocumentStatusDeleted {
			includesDeleted = true
		}
		clauses = append(clauses, clause)
	}

	if len(clauses) == 1 {
		return clauses[0], includesDeleted, nil
	}
	return "(" + strings.Join(clauses, " OR ") + ")", includesDeleted, nil
}

func documentStatusClause(status domain.DocumentStatus) (string, error) {
	switch status {
	case domain.DocumentStatusUploading:
		return "status = 'uploading'", nil
	case domain.DocumentStatusReady:
		return "status = 'ready'", nil
	case domain.DocumentStatusFailed:
		return "status = 'failed'", nil
	case domain.DocumentStatusDeleted:
		return "status = 'deleted'", nil
	default:
		return "", fmt.Errorf("list documents: invalid status %q", status)
	}
}
