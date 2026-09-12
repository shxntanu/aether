package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/shxntanu/aether/backend/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type documentSearchRow struct {
	Document      documentModel `gorm:"embedded"`
	TitleMatch    bool          `gorm:"column:title_match"`
	FilenameMatch bool          `gorm:"column:filename_match"`
}

type documentSearchTagRow struct {
	DocumentID     domain.DocumentID `gorm:"column:document_id"`
	ID             domain.TagID      `gorm:"column:id"`
	DisplayName    string            `gorm:"column:display_name"`
	NormalizedName string            `gorm:"column:normalized_name"`
	Implicit       bool              `gorm:"column:is_implicit"`
	Matched        bool              `gorm:"column:matched"`
}

// SearchDocuments returns deterministically ranked ready documents, their
// complete tag sets, and visible match evidence in two bounded queries.
func (s *Store) SearchDocuments(
	ctx context.Context,
	query string,
	limit int,
) ([]domain.DocumentSearchResult, error) {
	return s.SearchDocumentsWithFilters(
		ctx,
		query,
		nil,
		domain.TagMatchAll,
		limit,
	)
}

// SearchDocumentsWithFilters ranks ready documents while requiring all or any
// of the supplied normalized tags to be attached.
func (s *Store) SearchDocumentsWithFilters(
	ctx context.Context,
	query string,
	normalizedTags []string,
	tagMatch domain.TagMatch,
	limit int,
) ([]domain.DocumentSearchResult, error) {
	query = strings.ToLower(strings.TrimSpace(query))
	if limit <= 0 || limit > 20 {
		limit = 10
	}
	if tagMatch == "" {
		tagMatch = domain.TagMatchAll
	}
	if tagMatch != domain.TagMatchAll && tagMatch != domain.TagMatchAny {
		return nil, fmt.Errorf("search documents: invalid tag match %q", tagMatch)
	}

	rows, err := s.searchDocumentRows(ctx, query, normalizedTags, tagMatch, limit)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []domain.DocumentSearchResult{}, nil
	}

	results := make([]domain.DocumentSearchResult, len(rows))
	indices := make(map[domain.DocumentID]int, len(rows))
	ids := make([]domain.DocumentID, len(rows))
	for index, row := range rows {
		ids[index] = row.Document.ID
		indices[row.Document.ID] = index
		results[index] = domain.DocumentSearchResult{
			Document: row.Document.domain(),
			Tags:     []domain.Tag{},
			Evidence: documentFieldEvidence(row, query),
		}
	}

	tags, err := s.searchDocumentTags(ctx, ids, query)
	if err != nil {
		return nil, err
	}
	for _, row := range tags {
		index := indices[row.DocumentID]
		results[index].Tags = append(results[index].Tags, domain.Tag{
			ID:             row.ID,
			DisplayName:    row.DisplayName,
			NormalizedName: row.NormalizedName,
			Implicit:       row.Implicit,
		})
		if row.Matched {
			results[index].Evidence = append(
				results[index].Evidence,
				domain.DocumentSearchEvidence{
					Field: domain.DocumentSearchFieldTag,
					Value: row.DisplayName,
				},
			)
		}
	}
	return results, nil
}

func (s *Store) searchDocumentRows(
	ctx context.Context,
	query string,
	normalizedTags []string,
	tagMatch domain.TagMatch,
	limit int,
) ([]documentSearchRow, error) {
	var rows []documentSearchRow
	tagFilter := documentTagFilterClause(normalizedTags, tagMatch)
	arguments := map[string]any{"limit": limit}
	for index, tag := range normalizedTags {
		arguments[fmt.Sprintf("tag%d", index)] = tag
	}
	if len(normalizedTags) > 0 {
		arguments["tagCount"] = len(normalizedTags)
	}
	if query == "" {
		statement := `
SELECT d.*, FALSE AS title_match, FALSE AS filename_match
FROM documents_tbl d
WHERE d.status = 'ready'` + tagFilter + `
ORDER BY d.updated_at DESC, d.id DESC
LIMIT @limit`
		if err := s.searchORM(ctx).Raw(statement, arguments).Scan(&rows).Error; err != nil {
			return nil, fmt.Errorf("search recent documents: %w", err)
		}
		return rows, nil
	}

	pattern := escapeLike(query) + "%"
	arguments["query"] = query
	arguments["pattern"] = pattern
	ranking := `GREATEST(
CASE
  WHEN lower(d.title) = @query THEN 8
  WHEN lower(d.title) LIKE @pattern ESCAPE '\' THEN 6
  ELSE similarity(lower(d.title), @query) * 4
END,
CASE
  WHEN lower(d.original_filename) = @query THEN 7.8
  WHEN lower(d.original_filename) LIKE @pattern ESCAPE '\' THEN 5.8
  ELSE similarity(lower(d.original_filename), @query) * 3.8
END,
COALESCE((
  SELECT MAX(CASE
    WHEN lower(t.normalized_name) = @query THEN 7.6
    WHEN lower(t.normalized_name) LIKE @pattern ESCAPE '\' THEN 5.6
    ELSE similarity(lower(t.normalized_name), @query) * 3.6
  END)
  FROM document_tags_tbl dt
  JOIN tags_tbl t ON t.id = dt.tag_id
  WHERE dt.document_id = d.id
), 0))`
	statement := `
SELECT d.*,
       (lower(d.title) = @query
        OR lower(d.title) LIKE @pattern ESCAPE '\'
        OR lower(d.title) % @query) AS title_match,
       (lower(d.original_filename) = @query
        OR lower(d.original_filename) LIKE @pattern ESCAPE '\'
        OR lower(d.original_filename) % @query) AS filename_match,
       ` + ranking + ` AS search_rank
FROM documents_tbl d
WHERE d.status = 'ready'
  AND (
    lower(d.title) = @query
    OR lower(d.title) LIKE @pattern ESCAPE '\'
    OR lower(d.title) % @query
    OR lower(d.original_filename) = @query
    OR lower(d.original_filename) LIKE @pattern ESCAPE '\'
    OR lower(d.original_filename) % @query
    OR EXISTS (
      SELECT 1
      FROM document_tags_tbl dt
      JOIN tags_tbl t ON t.id = dt.tag_id
      WHERE dt.document_id = d.id
        AND (
          lower(t.normalized_name) = @query
          OR lower(t.normalized_name) LIKE @pattern ESCAPE '\'
          OR lower(t.normalized_name) % @query
        )
    )
  )` + tagFilter + `
ORDER BY search_rank DESC, d.updated_at DESC, d.id DESC
LIMIT @limit`
	if err := s.searchORM(ctx).Raw(statement, arguments).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("search documents: %w", err)
	}
	return rows, nil
}

func documentTagFilterClause(tags []string, match domain.TagMatch) string {
	if len(tags) == 0 {
		return ""
	}
	arguments := make([]string, len(tags))
	for index := range tags {
		arguments[index] = fmt.Sprintf("@tag%d", index)
	}
	tagList := strings.Join(arguments, ", ")
	if match == domain.TagMatchAny {
		return `
  AND EXISTS (
    SELECT 1
    FROM document_tags_tbl dt
    JOIN tags_tbl t ON t.id = dt.tag_id
    WHERE dt.document_id = d.id
      AND t.normalized_name IN (` + tagList + `)
  )`
	}
	return `
  AND d.id IN (
    SELECT dt.document_id
    FROM document_tags_tbl dt
    JOIN tags_tbl t ON t.id = dt.tag_id
    WHERE t.normalized_name IN (` + tagList + `)
    GROUP BY dt.document_id
    HAVING COUNT(DISTINCT t.id) = @tagCount
  )`
}

func (s *Store) searchDocumentTags(
	ctx context.Context,
	documentIDs []domain.DocumentID,
	query string,
) ([]documentSearchTagRow, error) {
	var rows []documentSearchTagRow
	selectClause := `document_tags_tbl.document_id, tags_tbl.id,
tags_tbl.display_name, tags_tbl.normalized_name, tags_tbl.is_implicit,
FALSE AS matched`
	arguments := []any{}
	if query != "" {
		selectClause = `document_tags_tbl.document_id, tags_tbl.id,
tags_tbl.display_name, tags_tbl.normalized_name, tags_tbl.is_implicit,
(lower(tags_tbl.normalized_name) = ?
 OR lower(tags_tbl.normalized_name) LIKE ? ESCAPE '\'
 OR lower(tags_tbl.normalized_name) % ?) AS matched`
		arguments = append(
			arguments,
			query,
			escapeLike(query)+"%",
			query,
		)
	}
	err := s.searchORM(ctx).
		Table("tags_tbl").
		Select(selectClause, arguments...).
		Joins("JOIN document_tags_tbl ON document_tags_tbl.tag_id = tags_tbl.id").
		Where("document_tags_tbl.document_id IN ?", documentIDs).
		Order("document_tags_tbl.document_id").
		Order("tags_tbl.normalized_name").
		Order("tags_tbl.id").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("search document tags: %w", err)
	}
	return rows, nil
}

func (s *Store) searchORM(ctx context.Context) *gorm.DB {
	return s.orm.WithContext(ctx).Session(&gorm.Session{Logger: logger.Discard})
}

func documentFieldEvidence(
	row documentSearchRow,
	query string,
) []domain.DocumentSearchEvidence {
	evidence := []domain.DocumentSearchEvidence{}
	if query == "" {
		return evidence
	}
	if row.TitleMatch {
		evidence = append(evidence, domain.DocumentSearchEvidence{
			Field: domain.DocumentSearchFieldTitle,
			Value: row.Document.Title,
		})
	}
	if row.FilenameMatch {
		evidence = append(evidence, domain.DocumentSearchEvidence{
			Field: domain.DocumentSearchFieldFilename,
			Value: row.Document.OriginalFilename,
		})
	}
	return evidence
}
