package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/shxntanu/aether/backend/internal/domain"
)

func (s *Store) CreateDocument(ctx context.Context, document domain.Document) error {
	_, err := s.executor.ExecContext(ctx, `
		INSERT INTO documents (
			id, title, original_filename, media_type, size_bytes, sha256, storage_key,
			status, index_status, uploader_id, version, created_at, updated_at,
			deleted_at, purge_after
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`,
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
	)
	return translateError("create document", err)
}

func (s *Store) GetDocument(ctx context.Context, id domain.DocumentID) (domain.Document, error) {
	row := s.executor.QueryRowContext(ctx, `
		SELECT id, title, original_filename, media_type, size_bytes, sha256, storage_key,
		       status, index_status, uploader_id, version, created_at, updated_at,
		       deleted_at, purge_after
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

func (s *Store) UpdateDocument(ctx context.Context, document domain.Document, expectedVersion int64) (domain.Document, error) {
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
		    version = version + 1
		WHERE id = $13 AND version = $14`,
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

type rowScanner interface {
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
