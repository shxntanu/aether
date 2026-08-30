package domain

import "time"

type AuditEventID string

type AuditOutcome string

const (
	AuditOutcomeSucceeded AuditOutcome = "succeeded"
	AuditOutcomeRejected  AuditOutcome = "rejected"
	AuditOutcomeFailed    AuditOutcome = "failed"
)

type AuditEvent struct {
	ID         AuditEventID
	ActorID    *MemberID
	Action     string
	ObjectType string
	ObjectID   string
	Outcome    AuditOutcome
	OccurredAt time.Time
}
