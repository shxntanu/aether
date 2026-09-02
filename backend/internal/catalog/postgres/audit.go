package postgres

import (
	"context"
	"fmt"

	"github.com/shxntanu/aether/backend/internal/domain"
)

// AppendAuditEvent adds one immutable event to the audit log.
//
// The store exposes no audit update or delete operation, preserving append-only
// semantics for the security history.
func (s *Store) AppendAuditEvent(ctx context.Context, event domain.AuditEvent) error {
	var actorID *string
	if event.ActorID != nil {
		value := string(*event.ActorID)
		actorID = &value
	}
	model := auditEventModel{
		ID:         string(event.ID),
		ActorID:    actorID,
		Action:     event.Action,
		ObjectType: event.ObjectType,
		ObjectID:   event.ObjectID,
		Outcome:    event.Outcome,
		OccurredAt: event.OccurredAt,
	}
	err := s.orm.WithContext(ctx).Create(&model).Error
	return translateError("append audit event", err)
}

// ListAuditEvents returns audit events for one object in chronological order.
//
// This read path does not provide mutation access to the append-only audit log.
func (s *Store) ListAuditEvents(
	ctx context.Context,
	objectType string,
	objectID string,
) ([]domain.AuditEvent, error) {
	var models []auditEventModel
	err := s.orm.WithContext(ctx).
		Where("object_type = ? AND object_id = ?", objectType, objectID).
		Order("occurred_at").
		Order("id").
		Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("list audit events: %w", err)
	}

	events := make([]domain.AuditEvent, 0, len(models))
	for _, model := range models {
		var actorID *domain.MemberID
		if model.ActorID != nil {
			value := domain.MemberID(*model.ActorID)
			actorID = &value
		}
		events = append(events, domain.AuditEvent{
			ID:         domain.AuditEventID(model.ID),
			ActorID:    actorID,
			Action:     model.Action,
			ObjectType: model.ObjectType,
			ObjectID:   model.ObjectID,
			Outcome:    model.Outcome,
			OccurredAt: model.OccurredAt,
		})
	}
	return events, nil
}
