package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shxntanu/aether/backend/internal/domain"
)

// CreateDocument persists a new document catalog record.
func (s *Store) CreateDocument(ctx context.Context, document domain.Document) error {
	_, err := s.executor.ExecContext(ctx, `
		INSERT INTO documents (
			id, title, original_filename, media_type, size_bytes, sha256, storage_key,
			status, index_status, uploader_id, version, created_at, updated_at,
			deleted_at, purge_after, manifest_error
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		)`,
		document.ID,
		document.Title,
		document.OriginalFilename,
		document.MediaType,
		document.SizeBytes,
		document.SHA256,
		document.StorageKey,
		document.Status,
		document.IndexStatus,
		document.UploaderID,
		document.Version,
		document.CreatedAt,
		document.UpdatedAt,
		document.DeletedAt,
		document.PurgeAfter,
		document.ManifestError,
	)
	return translateError("create document", err)
}

// GetDocument returns a document by its catalog identifier.
func (s *Store) GetDocument(ctx context.Context, id domain.DocumentID) (domain.Document, error) {
	row := s.executor.QueryRowContext(ctx, `
		SELECT id, title, original_filename, media_type, size_bytes, sha256, storage_key,
		       status, index_status, uploader_id, version, created_at, updated_at,
		       deleted_at, purge_after, manifest_error
		FROM documents
		WHERE id = $1`, id)

	document, err := scanDocument(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Document{}, fmt.Errorf("get document %q: %w", id, domain.ErrNotFound)
	}
	if err != nil {
		return domain.Document{}, fmt.Errorf("get document %q: %w", id, err)
	}
	return document, nil
}

// UpdateDocument applies an optimistic-concurrency catalog update.
func (s *Store) UpdateDocument(
	ctx context.Context,
	document domain.Document,
	expectedVersion int64,
) (domain.Document, error) {
	result, err := s.executor.ExecContext(ctx, `
		UPDATE documents
		SET title = $1,
		    original_filename = $2,
		    media_type = $3,
		    size_bytes = $4,
		    sha256 = $5,
		    storage_key = $6,
		    status = $7,
		    index_status = $8,
		    uploader_id = $9,
		    updated_at = $10,
		    deleted_at = $11,
		    purge_after = $12,
		    manifest_error = $13,
		    version = version + 1
		WHERE id = $14 AND version = $15`,
		document.Title,
		document.OriginalFilename,
		document.MediaType,
		document.SizeBytes,
		document.SHA256,
		document.StorageKey,
		document.Status,
		document.IndexStatus,
		document.UploaderID,
		document.UpdatedAt,
		document.DeletedAt,
		document.PurgeAfter,
		document.ManifestError,
		document.ID,
		expectedVersion,
	)
	if err != nil {
		return domain.Document{}, translateError("update document", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return domain.Document{}, fmt.Errorf("inspect document update: %w", err)
	}
	if rowsAffected == 0 {
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

	query := `
		SELECT d.id, d.title, d.original_filename, d.media_type, d.size_bytes,
		       d.sha256, d.storage_key, d.status, d.index_status, d.uploader_id,
		       d.version, d.created_at, d.updated_at, d.deleted_at, d.purge_after,
		       d.manifest_error
		FROM documents d
		WHERE ` + queryPlan.statusClause
	arguments := make([]any, 0, len(options.NormalizedTags)+2)
	if queryPlan.purgeDueBefore != nil {
		arguments = append(arguments, queryPlan.purgeDueBefore.UTC())
		query += fmt.Sprintf(
			" AND d.purge_after IS NOT NULL AND d.purge_after <= $%d",
			len(arguments),
		)
	}
	if len(options.NormalizedTags) > 0 {
		placeholders := make([]string, len(options.NormalizedTags))
		for index, tag := range options.NormalizedTags {
			arguments = append(arguments, tag)
			placeholders[index] = fmt.Sprintf("$%d", len(arguments))
		}
		query += ` AND d.id IN (
			SELECT dt.document_id
			FROM document_tags dt
			JOIN tags t ON t.id = dt.tag_id
			WHERE t.normalized_name IN (` + strings.Join(placeholders, ",") + `)
			GROUP BY dt.document_id`
		if queryPlan.tagMatch != domain.TagMatchAny {
			arguments = append(arguments, len(options.NormalizedTags))
			query += fmt.Sprintf(" HAVING COUNT(DISTINCT t.id) = $%d", len(arguments))
		}
		query += ")"
	}
	query += " ORDER BY " + queryPlan.orderBy
	if queryPlan.limit > 0 {
		arguments = append(arguments, queryPlan.limit)
		query += fmt.Sprintf(" LIMIT $%d", len(arguments))
	}

	rows, err := s.executor.QueryContext(ctx, query, arguments...)
	if err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}
	defer rows.Close()
	documents := make([]domain.Document, 0)
	for rows.Next() {
		document, err := scanDocument(rows)
		if err != nil {
			return nil, fmt.Errorf("scan document: %w", err)
		}
		documents = append(documents, document)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate documents: %w", err)
	}
	return documents, nil
}

// PurgeDocument permanently removes a deleted catalog row and its cascading
// document-tag join rows.
func (s *Store) PurgeDocument(ctx context.Context, id domain.DocumentID) error {
	result, err := s.executor.ExecContext(
		ctx,
		"DELETE FROM documents WHERE id = $1 AND status = 'deleted'",
		id,
	)
	if err != nil {
		return fmt.Errorf("purge document %q: %w", id, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect document purge %q: %w", id, err)
	}
	if rowsAffected > 0 {
		return nil
	}

	var status domain.DocumentStatus
	err = s.executor.QueryRowContext(
		ctx,
		"SELECT status FROM documents WHERE id = $1",
		id,
	).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
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
	result, err := s.executor.ExecContext(ctx, `
		INSERT INTO upload_requests (member_id, key_hash, document_id, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (member_id, key_hash) DO NOTHING`,
		uploaderID,
		keyHash,
		documentID,
		createdAt,
	)
	if err != nil {
		return "", false, fmt.Errorf("claim upload: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return "", false, fmt.Errorf("inspect upload claim: %w", err)
	}
	if rows == 1 {
		return documentID, true, nil
	}
	var existing domain.DocumentID
	err = s.executor.QueryRowContext(ctx, `
		SELECT document_id
		FROM upload_requests
		WHERE member_id = $1 AND key_hash = $2`, uploaderID, keyHash).Scan(&existing)
	if err != nil {
		return "", false, fmt.Errorf("get upload claim: %w", err)
	}
	return existing, false, nil
}

// rowScanner reads the columns of a single database row into destination
// values.
type rowScanner interface {
	// Scan copies the current row's columns into destination values.
	Scan(dest ...any) error
}

func scanDocument(row rowScanner) (domain.Document, error) {
	var document domain.Document
	var deletedAt sql.NullTime
	var purgeAfter sql.NullTime
	if err := row.Scan(
		&document.ID,
		&document.Title,
		&document.OriginalFilename,
		&document.MediaType,
		&document.SizeBytes,
		&document.SHA256,
		&document.StorageKey,
		&document.Status,
		&document.IndexStatus,
		&document.UploaderID,
		&document.Version,
		&document.CreatedAt,
		&document.UpdatedAt,
		&deletedAt,
		&purgeAfter,
		&document.ManifestError,
	); err != nil {
		return domain.Document{}, err
	}
	if deletedAt.Valid {
		document.DeletedAt = &deletedAt.Time
	}
	if purgeAfter.Valid {
		document.PurgeAfter = &purgeAfter.Time
	}
	return document, nil
}

type documentListQuery struct {
	statusClause   string
	tagMatch       domain.TagMatch
	orderBy        string
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

	orderBy := "d.created_at DESC, d.id DESC"
	if options.PurgeDueBefore != nil {
		if !includesDeleted {
			return documentListQuery{}, fmt.Errorf(
				"list documents: purge due filter requires deleted status",
			)
		}
		orderBy = "d.purge_after ASC, d.id ASC"
	}

	return documentListQuery{
		statusClause:   statusClause,
		tagMatch:       tagMatch,
		orderBy:        orderBy,
		purgeDueBefore: options.PurgeDueBefore,
		limit:          options.Limit,
	}, nil
}

func buildDocumentStatusClause(statuses []domain.DocumentStatus) (
	string,
	bool,
	error,
) {
	if statuses == nil {
		return "d.status = 'ready'", false, nil
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
		return "d.status = 'uploading'", nil
	case domain.DocumentStatusReady:
		return "d.status = 'ready'", nil
	case domain.DocumentStatusFailed:
		return "d.status = 'failed'", nil
	case domain.DocumentStatusDeleted:
		return "d.status = 'deleted'", nil
	default:
		return "", fmt.Errorf("list documents: invalid status %q", status)
	}
}
