package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/shxntanu/aether/backend/internal/domain"
)

func (s *Store) CreateSession(ctx context.Context, session domain.Session) error {
	_, err := s.executor.ExecContext(ctx, `INSERT INTO sessions (id, token_hash, member_id, created_at, expires_at) VALUES ($1,$2,$3,$4,$5)`, session.ID, session.TokenHash, session.MemberID, session.CreatedAt, session.ExpiresAt)
	return translateError("create session", err)
}
func (s *Store) GetSessionByTokenHash(ctx context.Context, hash string) (domain.Session, error) {
	var session domain.Session
	err := s.executor.QueryRowContext(ctx, `SELECT id, token_hash, member_id, created_at, expires_at FROM sessions WHERE token_hash=$1`, hash).Scan(&session.ID, &session.TokenHash, &session.MemberID, &session.CreatedAt, &session.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Session{}, fmt.Errorf("get session: %w", domain.ErrNotFound)
	}
	if err != nil {
		return domain.Session{}, fmt.Errorf("get session: %w", err)
	}
	return session, nil
}
func (s *Store) DeleteSessionByTokenHash(ctx context.Context, hash string) error {
	_, err := s.executor.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash=$1`, hash)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}
func (s *Store) CreateAuthFlow(ctx context.Context, flow domain.AuthFlow) error {
	_, err := s.executor.ExecContext(ctx, `INSERT INTO auth_flows (state_hash, nonce, pkce_verifier, created_at, expires_at) VALUES ($1,$2,$3,$4,$5)`, flow.StateHash, flow.Nonce, flow.PKCEVerifier, flow.CreatedAt, flow.ExpiresAt)
	return translateError("create auth flow", err)
}
func (s *Store) ConsumeAuthFlow(ctx context.Context, hash string) (domain.AuthFlow, error) {
	var flow domain.AuthFlow
	err := s.executor.QueryRowContext(ctx, `DELETE FROM auth_flows WHERE state_hash=$1 RETURNING state_hash, nonce, pkce_verifier, created_at, expires_at`, hash).Scan(&flow.StateHash, &flow.Nonce, &flow.PKCEVerifier, &flow.CreatedAt, &flow.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.AuthFlow{}, fmt.Errorf("consume auth flow: %w", domain.ErrNotFound)
	}
	if err != nil {
		return domain.AuthFlow{}, fmt.Errorf("consume auth flow: %w", err)
	}
	return flow, nil
}
