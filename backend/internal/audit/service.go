package audit

import (
	"context"
	"fmt"
	"time"
	"unicode"

	"github.com/shxntanu/aether/backend/internal/domain"
)

// Action identifies a security-relevant operation that may be audited.
type Action string

const (
	// ActionDocumentUpload records creation of a document.
	ActionDocumentUpload Action = "document.upload"
	// ActionDocumentEdit records a document metadata change.
	ActionDocumentEdit Action = "document.edit"
	// ActionDocumentDownload records a document download.
	ActionDocumentDownload Action = "document.download"
	// ActionDocumentDelete records a document soft deletion.
	ActionDocumentDelete Action = "document.delete"
	// ActionDocumentRestore records restoration of a deleted document.
	ActionDocumentRestore Action = "document.restore"
	// ActionDocumentPurge records permanent deletion of a document.
	ActionDocumentPurge Action = "document.purge"
	// ActionMemberCreate records creation of a member.
	ActionMemberCreate Action = "member.create"
	// ActionMemberChange records a member role or status change.
	ActionMemberChange Action = "member.change"
	// ActionAuthorizationReject records an authorization rejection.
	ActionAuthorizationReject Action = "authorization.reject"
)

// Recorder appends validated, privacy-safe audit events.
type Recorder interface {
	// Record validates and persists one immutable audit event.
	Record(context.Context, Event) error
}

// Appender persists an immutable audit event.
type Appender interface {
	// AppendAuditEvent adds an event to the append-only audit log.
	AppendAuditEvent(context.Context, domain.AuditEvent) error
}

// Event contains the caller-supplied facts needed to create an audit event.
// Object identifiers are bounded and restricted by Recorder before persistence.
type Event struct {
	// ActorID identifies the member responsible for the operation, when known.
	ActorID *domain.MemberID
	// Action identifies the allowlisted operation being recorded.
	Action Action
	// ObjectType identifies the allowlisted kind of object affected.
	ObjectType string
	// ObjectID identifies the affected object without request content.
	ObjectID string
	// Outcome describes the result of the operation.
	Outcome domain.AuditOutcome
}

type service struct {
	repository Appender
	now        func() time.Time
	newID      func() (string, error)
}

// NewRecorder constructs a Recorder with injected time and identifier sources.
func NewRecorder(repository Appender, now func() time.Time, newID func() (string, error)) Recorder {
	return &service{repository: repository, now: now, newID: newID}
}

// Record validates an event, constructs its immutable domain representation,
// and appends it without persisting request content or free-form metadata.
func (s *service) Record(ctx context.Context, input Event) error {
	if !validAction(input.Action) {
		return fmt.Errorf("invalid audit action %q", input.Action)
	}
	if !validOutcome(input.Outcome) {
		return fmt.Errorf("invalid audit outcome %q", input.Outcome)
	}
	if !validObjectType(input.ObjectType) {
		return fmt.Errorf("invalid audit object type %q", input.ObjectType)
	}
	if input.ObjectID == "" {
		return fmt.Errorf("audit object ID is required")
	}
	if len([]byte(input.ObjectID)) > 200 {
		return fmt.Errorf("audit object ID exceeds 200 bytes")
	}
	for _, character := range input.ObjectID {
		if unicode.IsControl(character) {
			return fmt.Errorf("audit object ID contains a control character")
		}
	}

	id, err := s.newID()
	if err != nil {
		return fmt.Errorf("generate audit event ID: %w", err)
	}

	var actorID *domain.MemberID
	if input.ActorID != nil {
		actor := *input.ActorID
		actorID = &actor
	}
	event := domain.AuditEvent{
		ID:         domain.AuditEventID(id),
		ActorID:    actorID,
		Action:     string(input.Action),
		ObjectType: input.ObjectType,
		ObjectID:   input.ObjectID,
		Outcome:    input.Outcome,
		OccurredAt: s.now().UTC(),
	}
	if err := s.repository.AppendAuditEvent(ctx, event); err != nil {
		return fmt.Errorf(
			"record audit event %q for %s %q: %w",
			input.Action,
			input.ObjectType,
			input.ObjectID,
			err,
		)
	}
	return nil
}

func validAction(action Action) bool {
	switch action {
	case ActionDocumentUpload,
		ActionDocumentEdit,
		ActionDocumentDownload,
		ActionDocumentDelete,
		ActionDocumentRestore,
		ActionDocumentPurge,
		ActionMemberCreate,
		ActionMemberChange,
		ActionAuthorizationReject:
		return true
	default:
		return false
	}
}

func validOutcome(outcome domain.AuditOutcome) bool {
	switch outcome {
	case domain.AuditOutcomeSucceeded, domain.AuditOutcomeRejected, domain.AuditOutcomeFailed:
		return true
	default:
		return false
	}
}

func validObjectType(objectType string) bool {
	switch objectType {
	case "document", "member", "request":
		return true
	default:
		return false
	}
}
