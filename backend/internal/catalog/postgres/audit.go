package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/shxntanu/aether/backend/internal/domain"
)

// AppendAuditEvent adds one immutable event to the audit log.
//
// The store exposes no audit update or delete operation, preserving append-only
// semantics for the security history.
func (s *Store) AppendAuditEvent(ctx context.Context, event domain.AuditEvent) error {
	_, err := s.executor.ExecContext(ctx, `
		INSERT INTO audit_events (id, actor_id, action, object_type, object_id, outcome, occurred_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		event.ID,
		event.ActorID,
		event.Action,
		event.ObjectType,
		event.ObjectID,
		event.Outcome,
		event.OccurredAt,
	)
	return translateError("append audit event", err)
}

// ListAuditEvents returns audit events for one object in chronological order.
//
// This read path does not provide mutation access to the append-only audit log.
func (s *Store) ListAuditEvents(ctx context.Context, objectType, objectID string) ([]domain.AuditEvent, error) {
	rows, err := s.executor.QueryContext(ctx, `
		SELECT id, actor_id, action, object_type, object_id, outcome, occurred_at
		FROM audit_events
		WHERE object_type = $1 AND object_id = $2
		ORDER BY occurred_at, id`, objectType, objectID)
	if err != nil {
		return nil, fmt.Errorf("list audit events: %w", err)
	}
	defer rows.Close()

	events := make([]domain.AuditEvent, 0)
	for rows.Next() {
		var event domain.AuditEvent
		var actorID sql.NullString
		if err := rows.Scan(
			&event.ID,
			&actorID,
			&event.Action,
			&event.ObjectType,
			&event.ObjectID,
			&event.Outcome,
			&event.OccurredAt,
		); err != nil {
			return nil, fmt.Errorf("scan audit event: %w", err)
		}
		if actorID.Valid {
			value := domain.MemberID(actorID.String)
			event.ActorID = &value
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit events: %w", err)
	}
	return events, nil
}
