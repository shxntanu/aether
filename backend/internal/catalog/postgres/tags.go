package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/shxntanu/aether/backend/internal/domain"
)

// CreateTag persists a reusable tag.
func (s *Store) CreateTag(ctx context.Context, tag domain.Tag) error {
	_, err := s.executor.ExecContext(ctx, `
		INSERT INTO tags (id, display_name, normalized_name)
		VALUES ($1, $2, $3)`, tag.ID, tag.DisplayName, tag.NormalizedName)
	return translateError("create tag", err)
}

// ReplaceDocumentTags replaces all tag associations for a document.
func (s *Store) ReplaceDocumentTags(
	ctx context.Context,
	documentID domain.DocumentID,
	tagIDs []domain.TagID,
) error {
	values := make([]string, len(tagIDs))
	arguments := make([]any, 0, len(tagIDs)+1)
	arguments = append(arguments, documentID)
	for index, tagID := range tagIDs {
		arguments = append(arguments, tagID)
		values[index] = fmt.Sprintf("($1, $%d)", index+2)
	}
	query := "WITH removed AS (DELETE FROM document_tags WHERE document_id = $1) "
	if len(values) == 0 {
		query += "SELECT $1"
	} else {
		query += "INSERT INTO document_tags (document_id, tag_id) VALUES " +
			strings.Join(values, ", ")
	}
	if _, err := s.executor.ExecContext(ctx, query, arguments...); err != nil {
		return translateError("replace document tags", err)
	}
	return nil
}

// ListTags returns tags containing a normalized query, ordered for autocomplete.
func (s *Store) ListTags(ctx context.Context, query string, limit int) ([]domain.Tag, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := s.executor.QueryContext(ctx, `
		SELECT id, display_name, normalized_name
		FROM tags
		WHERE normalized_name LIKE $1 ESCAPE '\'
		ORDER BY normalized_name, id
		LIMIT $2`, "%"+escapeLike(strings.ToLower(strings.TrimSpace(query)))+"%", limit)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	defer rows.Close()
	tags := make([]domain.Tag, 0)
	for rows.Next() {
		var tag domain.Tag
		if err := rows.Scan(&tag.ID, &tag.DisplayName, &tag.NormalizedName); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		tags = append(tags, tag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tags: %w", err)
	}
	return tags, nil
}

func escapeLike(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "%", "\\%")
	return strings.ReplaceAll(value, "_", "\\_")
}

// GetTagByNormalizedName returns a tag by its case-insensitive identity.
func (s *Store) GetTagByNormalizedName(
	ctx context.Context,
	normalizedName string,
) (domain.Tag, error) {
	var tag domain.Tag
	err := s.executor.QueryRowContext(ctx, `
		SELECT id, display_name, normalized_name
		FROM tags
		WHERE normalized_name = $1`, normalizedName).Scan(&tag.ID, &tag.DisplayName, &tag.NormalizedName)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Tag{}, fmt.Errorf("get tag %q: %w", normalizedName, domain.ErrNotFound)
	}
	if err != nil {
		return domain.Tag{}, fmt.Errorf("get tag %q: %w", normalizedName, err)
	}
	return tag, nil
}

// AttachTag associates a reusable tag with a document.
func (s *Store) AttachTag(
	ctx context.Context,
	documentID domain.DocumentID,
	tagID domain.TagID,
) error {
	_, err := s.executor.ExecContext(ctx, `
		INSERT INTO document_tags (document_id, tag_id)
		VALUES ($1, $2)
		ON CONFLICT (document_id, tag_id) DO NOTHING`, documentID, tagID)
	return translateError("attach tag", err)
}

// ListDocumentTags returns a document's tags in stable normalized order.
func (s *Store) ListDocumentTags(
	ctx context.Context,
	documentID domain.DocumentID,
) ([]domain.Tag, error) {
	rows, err := s.executor.QueryContext(ctx, `
		SELECT tags.id, tags.display_name, tags.normalized_name
		FROM tags
		JOIN document_tags ON document_tags.tag_id = tags.id
		WHERE document_tags.document_id = $1
		ORDER BY tags.normalized_name, tags.id`, documentID)
	if err != nil {
		return nil, fmt.Errorf("list document tags: %w", err)
	}
	defer rows.Close()

	tags := make([]domain.Tag, 0)
	for rows.Next() {
		var tag domain.Tag
		if err := rows.Scan(&tag.ID, &tag.DisplayName, &tag.NormalizedName); err != nil {
			return nil, fmt.Errorf("scan document tag: %w", err)
		}
		tags = append(tags, tag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate document tags: %w", err)
	}
	return tags, nil
}
