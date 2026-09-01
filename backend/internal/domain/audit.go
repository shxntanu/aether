package domain

import "time"

// AuditEventID uniquely identifies an immutable audit event.
type AuditEventID string

// AuditOutcome describes whether an audited operation succeeded, was rejected,
// or failed.
type AuditOutcome string

const (
	// AuditOutcomeSucceeded indicates that the audited operation completed.
	AuditOutcomeSucceeded AuditOutcome = "succeeded"
	// AuditOutcomeRejected indicates that policy or authorization rejected the operation.
	AuditOutcomeRejected AuditOutcome = "rejected"
	// AuditOutcomeFailed indicates that the operation was attempted but failed.
	AuditOutcomeFailed AuditOutcome = "failed"
)

// AuditEvent is the privacy-safe, append-only record of a security-relevant operation.
// It intentionally contains no request payload, query text, or free-form metadata.
type AuditEvent struct {
	// ID uniquely identifies this event.
	ID AuditEventID
	// ActorID identifies the member who initiated the operation, when known.
	ActorID *MemberID
	// Action is the stable operation name recorded by the audit policy.
	Action string
	// ObjectType identifies the allowlisted kind of object affected.
	ObjectType string
	// ObjectID identifies the affected object without carrying request content.
	ObjectID string
	// Outcome describes the operation result.
	Outcome AuditOutcome
	// OccurredAt is the UTC time at which the event was created.
	OccurredAt time.Time
}
