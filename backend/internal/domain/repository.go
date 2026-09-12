package domain

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrNotFound indicates that the requested domain record does not exist.
	ErrNotFound = errors.New("not found")
	// ErrConflict indicates that the requested mutation conflicts with the
	// current stored state.
	ErrConflict = errors.New("version conflict")
	// ErrAlreadyExists indicates that a create operation duplicated a unique
	// domain record.
	ErrAlreadyExists = errors.New("already exists")
)

// Repository persists catalog entities and coordinates operations that must
// share a transaction. Implementations return ErrNotFound, ErrConflict, or
// ErrAlreadyExists when the corresponding domain condition applies.
type Repository interface {
	// CreateDocument persists a new document.
	CreateDocument(ctx context.Context, document Document) error
	// GetDocument returns a document by ID.
	GetDocument(ctx context.Context, id DocumentID) (Document, error)
	// UpdateDocument applies metadata changes only when expectedVersion matches.
	UpdateDocument(ctx context.Context, document Document, expectedVersion int64) (Document, error)
	// ListDocuments returns documents matching the supplied listing options.
	//
	// A nil options.Statuses slice preserves the legacy ready-only default.
	ListDocuments(ctx context.Context, options DocumentListOptions) ([]Document, error)
	// PurgeDocument permanently removes a soft-deleted catalog document.
	//
	// Implementations must return ErrConflict when the document exists but is
	// not currently deleted. Missing documents are treated as already purged.
	PurgeDocument(ctx context.Context, id DocumentID) error
	// ClaimUpload binds an uploader's idempotency digest to one document ID.
	// It returns the prior document ID and false when the digest already exists.
	ClaimUpload(
		ctx context.Context,
		uploaderID MemberID,
		keyHash string,
		documentID DocumentID,
		createdAt time.Time,
	) (DocumentID, bool, error)

	// CreateTag persists a new reusable tag.
	CreateTag(ctx context.Context, tag Tag) error
	// UpdateTag changes the display and normalized names of an existing tag.
	UpdateTag(ctx context.Context, tag Tag) (Tag, error)
	// DeleteTag removes a reusable tag and its document associations.
	DeleteTag(ctx context.Context, id TagID) error
	// GetTag returns a reusable or implicit tag by ID.
	GetTag(ctx context.Context, id TagID) (Tag, error)
	// GetTagByNormalizedName returns a tag by its case-normalized name.
	GetTagByNormalizedName(ctx context.Context, normalizedName string) (Tag, error)
	// AttachTag associates an existing tag with a document.
	AttachTag(ctx context.Context, documentID DocumentID, tagID TagID) error
	// ListDocumentTags returns all tags associated with a document.
	ListDocumentTags(ctx context.Context, documentID DocumentID) ([]Tag, error)
	// ReplaceDocumentTags atomically replaces a document's tag associations.
	ReplaceDocumentTags(ctx context.Context, documentID DocumentID, tagIDs []TagID) error
	// ListTags returns reusable tags whose normalized names contain query.
	ListTags(ctx context.Context, query string, limit int) ([]Tag, error)

	// CreateMember persists a new allowlisted member.
	CreateMember(ctx context.Context, member Member) error
	// GetMember returns an allowlisted member by ID.
	GetMember(ctx context.Context, id MemberID) (Member, error)
	// GetMemberByEmail returns an allowlisted member by normalized email.
	GetMemberByEmail(ctx context.Context, email string) (Member, error)
	// UpdateMember persists changes to an existing member.
	UpdateMember(ctx context.Context, member Member) (Member, error)
	// ListMembers returns all allowlisted members.
	ListMembers(ctx context.Context) ([]Member, error)
	// CreateSession persists a server-side session.
	CreateSession(ctx context.Context, session Session) error
	// GetSessionByTokenHash returns a session by its stored token digest.
	GetSessionByTokenHash(ctx context.Context, tokenHash string) (Session, error)
	// DeleteSessionByTokenHash invalidates a session by its stored token digest.
	DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error
	// CreateAuthFlow persists a short-lived OIDC authorization flow.
	CreateAuthFlow(ctx context.Context, flow AuthFlow) error
	// ConsumeAuthFlow atomically returns and invalidates a flow by state digest.
	ConsumeAuthFlow(ctx context.Context, stateHash string) (AuthFlow, error)

	// AppendAuditEvent adds an immutable event to the audit log.
	AppendAuditEvent(ctx context.Context, event AuditEvent) error
	// ListAuditEvents returns audit events for a catalog object.
	ListAuditEvents(ctx context.Context, objectType, objectID string) ([]AuditEvent, error)

	// WithinTransaction runs operation with all repository calls in one transaction.
	WithinTransaction(ctx context.Context, operation func(Repository) error) error
}
