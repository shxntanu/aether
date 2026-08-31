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

// ListDocuments returns ready documents matching optional normalized tags.
func (s *Store) ListDocuments(
	ctx context.Context,
	options domain.DocumentListOptions,
) ([]domain.Document, error) {
	query := `
		SELECT d.id, d.title, d.original_filename, d.media_type, d.size_bytes,
		       d.sha256, d.storage_key, d.status, d.index_status, d.uploader_id,
		       d.version, d.created_at, d.updated_at, d.deleted_at, d.purge_after,
		       d.manifest_error
		FROM documents d
		WHERE d.status = 'ready'`
	arguments := make([]any, 0, len(options.NormalizedTags)+1)
	if len(options.NormalizedTags) > 0 {
		placeholders := make([]string, len(options.NormalizedTags))
		for index, tag := range options.NormalizedTags {
			arguments = append(arguments, tag)
			placeholders[index] = fmt.Sprintf("$%d", index+1)
		}
		query += ` AND d.id IN (
			SELECT dt.document_id
			FROM document_tags dt
			JOIN tags t ON t.id = dt.tag_id
			WHERE t.normalized_name IN (` + strings.Join(placeholders, ",") + `)
			GROUP BY dt.document_id`
		if options.TagMatch != domain.TagMatchAny {
			arguments = append(arguments, len(options.NormalizedTags))
			query += fmt.Sprintf(" HAVING COUNT(DISTINCT t.id) = $%d", len(arguments))
		}
		query += ")"
	}
	query += " ORDER BY d.created_at DESC, d.id DESC"

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
