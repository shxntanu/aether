package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/shxntanu/aether/backend/internal/domain"
)

func (s *Store) CreateMember(ctx context.Context, member domain.Member) error {
	_, err := s.executor.ExecContext(ctx, `
		INSERT INTO members (id, email, display_name, role, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		member.ID,
		normalizeEmail(member.Email),
		member.DisplayName,
		member.Role,
		member.Status,
		member.CreatedAt,
		member.UpdatedAt,
	)
	return translateError("create member", err)
}

func (s *Store) GetMemberByEmail(ctx context.Context, email string) (domain.Member, error) {
	var member domain.Member
	err := s.executor.QueryRowContext(ctx, `
		SELECT id, email, display_name, role, status, created_at, updated_at
		FROM members
		WHERE email = $1`, normalizeEmail(email)).Scan(
		&member.ID,
		&member.Email,
		&member.DisplayName,
		&member.Role,
		&member.Status,
		&member.CreatedAt,
		&member.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Member{}, fmt.Errorf("get member by email: %w", domain.ErrNotFound)
	}
	if err != nil {
		return domain.Member{}, fmt.Errorf("get member by email: %w", err)
	}
	return member, nil
}
