package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/shxntanu/aether/backend/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CreateTag persists a reusable tag.
func (s *Store) CreateTag(ctx context.Context, tag domain.Tag) error {
	model := tagModel{
		ID:             tag.ID,
		DisplayName:    tag.DisplayName,
		NormalizedName: tag.NormalizedName,
	}
	err := s.orm.WithContext(ctx).Create(&model).Error
	return translateError("create tag", err)
}

// ReplaceDocumentTags replaces all tag associations for a document.
func (s *Store) ReplaceDocumentTags(
	ctx context.Context,
	documentID domain.DocumentID,
	tagIDs []domain.TagID,
) error {
	err := s.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("document_id = ?", documentID).
			Delete(&documentTagModel{}).Error; err != nil {
			return err
		}
		if len(tagIDs) == 0 {
			return nil
		}

		models := make([]documentTagModel, len(tagIDs))
		for index, tagID := range tagIDs {
			models[index] = documentTagModel{DocumentID: documentID, TagID: tagID}
		}
		return tx.Create(&models).Error
	})
	return translateError("replace document tags", err)
}

// ListTags returns tags containing a normalized query, ordered for autocomplete.
func (s *Store) ListTags(ctx context.Context, query string, limit int) ([]domain.Tag, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	pattern := "%" + escapeLike(strings.ToLower(strings.TrimSpace(query))) + "%"
	var models []tagModel
	err := s.orm.WithContext(ctx).
		Where("normalized_name LIKE ? ESCAPE '\\'", pattern).
		Order("normalized_name").
		Order("id").
		Limit(limit).
		Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}

	tags := make([]domain.Tag, 0, len(models))
	for _, model := range models {
		tags = append(tags, domain.Tag{
			ID:             model.ID,
			DisplayName:    model.DisplayName,
			NormalizedName: model.NormalizedName,
		})
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
	var model tagModel
	err := s.orm.WithContext(ctx).
		Where("normalized_name = ?", normalizedName).
		First(&model).Error
	if isRecordNotFound(err) {
		return domain.Tag{}, fmt.Errorf("get tag %q: %w", normalizedName, domain.ErrNotFound)
	}
	if err != nil {
		return domain.Tag{}, fmt.Errorf("get tag %q: %w", normalizedName, err)
	}
	return domain.Tag{
		ID:             model.ID,
		DisplayName:    model.DisplayName,
		NormalizedName: model.NormalizedName,
	}, nil
}

// AttachTag associates a reusable tag with a document.
func (s *Store) AttachTag(
	ctx context.Context,
	documentID domain.DocumentID,
	tagID domain.TagID,
) error {
	err := s.orm.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&documentTagModel{DocumentID: documentID, TagID: tagID}).Error
	return translateError("attach tag", err)
}

// ListDocumentTags returns a document's tags in stable normalized order.
func (s *Store) ListDocumentTags(
	ctx context.Context,
	documentID domain.DocumentID,
) ([]domain.Tag, error) {
	var models []tagModel
	err := s.orm.WithContext(ctx).
		Joins("JOIN document_tags_tbl ON document_tags_tbl.tag_id = tags_tbl.id").
		Where("document_tags_tbl.document_id = ?", documentID).
		Order("tags_tbl.normalized_name").
		Order("tags_tbl.id").
		Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("list document tags: %w", err)
	}

	tags := make([]domain.Tag, 0, len(models))
	for _, model := range models {
		tags = append(tags, domain.Tag{
			ID:             model.ID,
			DisplayName:    model.DisplayName,
			NormalizedName: model.NormalizedName,
		})
	}
	return tags, nil
}
