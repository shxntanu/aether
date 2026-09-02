package postgres

import (
	"context"
	"fmt"

	"github.com/shxntanu/aether/backend/internal/domain"
	"gorm.io/gorm/clause"
)

// CreateSession persists a server-side session.
func (s *Store) CreateSession(ctx context.Context, session domain.Session) error {
	model := sessionModel{
		ID:        session.ID,
		TokenHash: session.TokenHash,
		CSRFHash:  session.CSRFHash,
		MemberID:  session.MemberID,
		CreatedAt: session.CreatedAt,
		ExpiresAt: session.ExpiresAt,
	}
	err := s.orm.WithContext(ctx).Create(&model).Error
	return translateError("create session", err)
}

// GetSessionByTokenHash returns a session by its stored token digest.
func (s *Store) GetSessionByTokenHash(ctx context.Context, hash string) (domain.Session, error) {
	var model sessionModel
	err := s.orm.WithContext(ctx).Where("token_hash = ?", hash).First(&model).Error
	if isRecordNotFound(err) {
		return domain.Session{}, fmt.Errorf("get session: %w", domain.ErrNotFound)
	}
	if err != nil {
		return domain.Session{}, fmt.Errorf("get session: %w", err)
	}
	return domain.Session{
		ID:        model.ID,
		TokenHash: model.TokenHash,
		CSRFHash:  model.CSRFHash,
		MemberID:  model.MemberID,
		CreatedAt: model.CreatedAt,
		ExpiresAt: model.ExpiresAt,
	}, nil
}

// DeleteSessionByTokenHash invalidates a session by its stored token digest.
func (s *Store) DeleteSessionByTokenHash(ctx context.Context, hash string) error {
	err := s.orm.WithContext(ctx).Where("token_hash = ?", hash).Delete(&sessionModel{}).Error
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// CreateAuthFlow persists a short-lived OIDC authorization flow.
func (s *Store) CreateAuthFlow(ctx context.Context, flow domain.AuthFlow) error {
	model := authFlowModel{
		StateHash:    flow.StateHash,
		Nonce:        flow.Nonce,
		PKCEVerifier: flow.PKCEVerifier,
		CreatedAt:    flow.CreatedAt,
		ExpiresAt:    flow.ExpiresAt,
	}
	err := s.orm.WithContext(ctx).Create(&model).Error
	return translateError("create auth flow", err)
}

// ConsumeAuthFlow atomically returns and invalidates a flow by state digest.
func (s *Store) ConsumeAuthFlow(ctx context.Context, hash string) (domain.AuthFlow, error) {
	var model authFlowModel
	result := s.orm.WithContext(ctx).
		Clauses(clause.Returning{Columns: []clause.Column{
			{Name: "state_hash"},
			{Name: "nonce"},
			{Name: "pkce_verifier"},
			{Name: "created_at"},
			{Name: "expires_at"},
		}}).
		Where("state_hash = ?", hash).
		Delete(&model)
	if result.Error != nil {
		return domain.AuthFlow{}, fmt.Errorf("consume auth flow: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.AuthFlow{}, fmt.Errorf("consume auth flow: %w", domain.ErrNotFound)
	}
	return domain.AuthFlow{
		StateHash:    model.StateHash,
		Nonce:        model.Nonce,
		PKCEVerifier: model.PKCEVerifier,
		CreatedAt:    model.CreatedAt,
		ExpiresAt:    model.ExpiresAt,
	}, nil
}
