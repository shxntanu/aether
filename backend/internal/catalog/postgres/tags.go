package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/shxntanu/aether/backend/internal/domain"
)

func (s *Store) CreateTag(ctx context.Context, tag domain.Tag) error {
	_, err := s.executor.ExecContext(ctx, `
		INSERT INTO tags (id, display_name, normalized_name)
		VALUES ($1, $2, $3)`, tag.ID, tag.DisplayName, tag.NormalizedName)
	return translateError("create tag", err)
}

func (s *Store) GetTagByNormalizedName(ctx context.Context, normalizedName string) (domain.Tag, error) {
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

func (s *Store) AttachTag(ctx context.Context, documentID domain.DocumentID, tagID domain.TagID) error {
	_, err := s.executor.ExecContext(ctx, `
		INSERT INTO document_tags (document_id, tag_id)
		VALUES ($1, $2)
		ON CONFLICT (document_id, tag_id) DO NOTHING`, documentID, tagID)
	return translateError("attach tag", err)
}

func (s *Store) ListDocumentTags(ctx context.Context, documentID domain.DocumentID) ([]domain.Tag, error) {
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
