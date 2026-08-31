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
	return s.getMember(ctx, "email", normalizeEmail(email))
}

func (s *Store) GetMember(ctx context.Context, id domain.MemberID) (domain.Member, error) {
	return s.getMember(ctx, "id", id)
}

func (s *Store) getMember(ctx context.Context, column string, value any) (domain.Member, error) {
	var member domain.Member
	err := s.executor.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT id, email, display_name, COALESCE(oidc_subject, ''), role, status, created_at, updated_at
		FROM members
		WHERE %s = $1`, column), value).Scan(
		&member.ID,
		&member.Email,
		&member.DisplayName,
		&member.OIDCSubject,
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

func (s *Store) UpdateMember(ctx context.Context, member domain.Member) (domain.Member, error) {
	var subject any
	if member.OIDCSubject != "" {
		subject = member.OIDCSubject
	}
	result, err := s.executor.ExecContext(ctx, `UPDATE members SET email=$2, display_name=$3, oidc_subject=$4, role=$5, status=$6, updated_at=$7 WHERE id=$1`, member.ID, normalizeEmail(member.Email), member.DisplayName, subject, member.Role, member.Status, member.UpdatedAt)
	if err != nil {
		return domain.Member{}, translateError("update member", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return domain.Member{}, fmt.Errorf("update member rows: %w", err)
	}
	if rows == 0 {
		return domain.Member{}, fmt.Errorf("update member: %w", domain.ErrNotFound)
	}
	return s.GetMember(ctx, member.ID)
}

func (s *Store) ListMembers(ctx context.Context) ([]domain.Member, error) {
	rows, err := s.executor.QueryContext(ctx, `SELECT id, email, display_name, COALESCE(oidc_subject, ''), role, status, created_at, updated_at FROM members ORDER BY email`)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()
	var members []domain.Member
	for rows.Next() {
		var member domain.Member
		if err := rows.Scan(&member.ID, &member.Email, &member.DisplayName, &member.OIDCSubject, &member.Role, &member.Status, &member.CreatedAt, &member.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan member: %w", err)
		}
		members = append(members, member)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	return members, nil
}
