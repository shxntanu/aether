package postgres

import (
	"context"
	"fmt"

	"github.com/shxntanu/aether/backend/internal/domain"
)

// CreateMember persists a new allowlisted member.
func (s *Store) CreateMember(ctx context.Context, member domain.Member) error {
	model := memberModelFromDomain(member)
	model.Email = normalizeEmail(model.Email)
	err := s.orm.WithContext(ctx).Create(&model).Error
	return translateError("create member", err)
}

// GetMemberByEmail returns an allowlisted member by normalized email.
func (s *Store) GetMemberByEmail(ctx context.Context, email string) (domain.Member, error) {
	return s.getMember(ctx, "email", normalizeEmail(email))
}

// GetMember returns an allowlisted member by ID.
func (s *Store) GetMember(ctx context.Context, id domain.MemberID) (domain.Member, error) {
	return s.getMember(ctx, "id", id)
}

func (s *Store) getMember(ctx context.Context, column string, value any) (domain.Member, error) {
	if column != "email" && column != "id" {
		return domain.Member{}, fmt.Errorf("get member: invalid lookup column %q", column)
	}

	var model memberModel
	err := s.orm.WithContext(ctx).Where(column+" = ?", value).First(&model).Error
	if isRecordNotFound(err) {
		return domain.Member{}, fmt.Errorf("get member: %w", domain.ErrNotFound)
	}
	if err != nil {
		return domain.Member{}, fmt.Errorf("get member: %w", err)
	}
	return model.domain(), nil
}

// UpdateMember persists changes to an existing member.
func (s *Store) UpdateMember(ctx context.Context, member domain.Member) (domain.Member, error) {
	updates := map[string]any{
		"email":        normalizeEmail(member.Email),
		"display_name": member.DisplayName,
		"oidc_subject": nil,
		"role":         member.Role,
		"status":       member.Status,
		"updated_at":   member.UpdatedAt,
	}
	if member.OIDCSubject != "" {
		updates["oidc_subject"] = member.OIDCSubject
	}
	result := s.orm.WithContext(ctx).
		Model(&memberModel{}).
		Where("id = ?", member.ID).
		Updates(updates)
	if result.Error != nil {
		return domain.Member{}, translateError("update member", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.Member{}, fmt.Errorf("update member: %w", domain.ErrNotFound)
	}
	return s.GetMember(ctx, member.ID)
}

// ListMembers returns all allowlisted members ordered by email.
func (s *Store) ListMembers(ctx context.Context) ([]domain.Member, error) {
	var models []memberModel
	err := s.orm.WithContext(ctx).Order("email").Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}

	members := make([]domain.Member, 0, len(models))
	for _, model := range models {
		members = append(members, model.domain())
	}
	return members, nil
}
