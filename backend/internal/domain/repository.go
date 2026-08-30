package domain

import (
	"context"
	"errors"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrConflict      = errors.New("version conflict")
	ErrAlreadyExists = errors.New("already exists")
)

type Repository interface {
	CreateDocument(ctx context.Context, document Document) error
	GetDocument(ctx context.Context, id DocumentID) (Document, error)
	UpdateDocument(ctx context.Context, document Document, expectedVersion int64) (Document, error)

	CreateTag(ctx context.Context, tag Tag) error
	GetTagByNormalizedName(ctx context.Context, normalizedName string) (Tag, error)
	AttachTag(ctx context.Context, documentID DocumentID, tagID TagID) error
	ListDocumentTags(ctx context.Context, documentID DocumentID) ([]Tag, error)

	CreateMember(ctx context.Context, member Member) error
	GetMemberByEmail(ctx context.Context, email string) (Member, error)

	AppendAuditEvent(ctx context.Context, event AuditEvent) error
	ListAuditEvents(ctx context.Context, objectType, objectID string) ([]AuditEvent, error)

	WithinTransaction(ctx context.Context, operation func(Repository) error) error
}
