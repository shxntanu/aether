package identity

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/shxntanu/aether/backend/internal/domain"
)

var (
	ErrEmailUnverified  = errors.New("Google email is not verified")
	ErrNotAllowlisted   = errors.New("account is not allowlisted")
	ErrMemberDisabled   = errors.New("membership is disabled")
	ErrInvalidSession   = errors.New("session is invalid or expired")
	ErrInvalidCSRFToken = errors.New("csrf token is invalid or expired")
)

type Identity struct {
	Subject       string
	Email         string
	EmailVerified bool
	DisplayName   string
}

// Repository persists allowlisted members and their server-side sessions.
type Repository interface {
	// CreateMember persists an allowlisted member.
	CreateMember(context.Context, domain.Member) error
	// GetMemberByEmail returns a member by normalized email.
	GetMemberByEmail(context.Context, string) (domain.Member, error)
	// GetMember returns a member by ID.
	GetMember(context.Context, domain.MemberID) (domain.Member, error)
	// UpdateMember persists changes to an existing member.
	UpdateMember(context.Context, domain.Member) (domain.Member, error)
	// ListMembers returns all allowlisted members.
	ListMembers(context.Context) ([]domain.Member, error)
	// CreateSession persists a server-side session.
	CreateSession(context.Context, domain.Session) error
	// GetSessionByTokenHash returns a session by its stored token digest.
	GetSessionByTokenHash(context.Context, string) (domain.Session, error)
	// DeleteSessionByTokenHash invalidates a session by its stored token digest.
	DeleteSessionByTokenHash(context.Context, string) error
}

// Service applies membership policy and manages authenticated sessions.
type Service struct {
	repository Repository
	now        func() time.Time
	newSecret  func() string
}

// NewService constructs a Service using injected clock and secret generators.
func NewService(repository Repository, now func() time.Time, newSecret func() string) *Service {
	return &Service{repository: repository, now: now, newSecret: newSecret}
}

// BootstrapAdmin creates the initial active administrator, or promotes the
// configured account while preserving an existing disabled status.
func (s *Service) BootstrapAdmin(ctx context.Context, email string) (domain.Member, error) {
	email = normalizeEmail(email)
	if email == "" {
		return domain.Member{}, fmt.Errorf("bootstrap administrator email is required")
	}
	member, err := s.repository.GetMemberByEmail(ctx, email)
	if err == nil {
		if member.Role != domain.MemberRoleAdmin {
			member.Role = domain.MemberRoleAdmin
			member.UpdatedAt = s.now()
			return s.repository.UpdateMember(ctx, member)
		}
		return member, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return domain.Member{}, err
	}
	now := s.now()
	member = domain.Member{ID: domain.MemberID(s.newSecret()), Email: email, Role: domain.MemberRoleAdmin, Status: domain.MemberStatusActive, CreatedAt: now, UpdatedAt: now}
	if err := s.repository.CreateMember(ctx, member); err != nil {
		return domain.Member{}, err
	}
	return member, nil
}

// CompleteLogin admits a verified, active allowlisted identity and returns a
// raw opaque session token. Only its SHA-256 digest is persisted.
func (s *Service) CompleteLogin(ctx context.Context, identity Identity) (string, domain.Member, error) {
	token, session, err := s.CompleteLoginWithCSRF(ctx, identity)
	return token, session.Member, err
}

// CompleteLoginWithCSRF admits an identity and returns its session token plus
// the raw CSRF token exactly once. Only SHA-256 digests are persisted.
func (s *Service) CompleteLoginWithCSRF(
	ctx context.Context,
	identity Identity,
) (string, domain.AuthenticatedSession, error) {
	if !identity.EmailVerified {
		return "", domain.AuthenticatedSession{}, ErrEmailUnverified
	}
	member, err := s.repository.GetMemberByEmail(ctx, normalizeEmail(identity.Email))
	if errors.Is(err, domain.ErrNotFound) {
		return "", domain.AuthenticatedSession{}, ErrNotAllowlisted
	}
	if err != nil {
		return "", domain.AuthenticatedSession{}, err
	}
	if member.Status != domain.MemberStatusActive {
		return "", domain.AuthenticatedSession{}, ErrMemberDisabled
	}
	if identity.DisplayName != "" || identity.Subject != "" {
		member.DisplayName = identity.DisplayName
		member.OIDCSubject = identity.Subject
		member.UpdatedAt = s.now()
		member, err = s.repository.UpdateMember(ctx, member)
		if err != nil {
			return "", domain.AuthenticatedSession{}, err
		}
	}
	token, csrfToken := s.newSecret(), s.newSecret()
	now := s.now()
	session := domain.Session{
		ID:        s.newSecret(),
		TokenHash: tokenHash(token),
		CSRFHash:  tokenHash(csrfToken),
		MemberID:  member.ID,
		CreatedAt: now,
		ExpiresAt: now.Add(7 * 24 * time.Hour),
	}
	if err := s.repository.CreateSession(ctx, session); err != nil {
		return "", domain.AuthenticatedSession{}, err
	}
	return token, domain.AuthenticatedSession{Member: member, CSRFToken: csrfToken}, nil
}

// Authenticate resolves a raw session token against current membership state.
func (s *Service) Authenticate(ctx context.Context, token string) (domain.Member, error) {
	if token == "" {
		return domain.Member{}, ErrInvalidSession
	}
	session, err := s.repository.GetSessionByTokenHash(ctx, tokenHash(token))
	if err != nil || !session.ExpiresAt.After(s.now()) {
		return domain.Member{}, ErrInvalidSession
	}
	member, err := s.repository.GetMember(ctx, session.MemberID)
	if err != nil {
		return domain.Member{}, ErrInvalidSession
	}
	if member.Status != domain.MemberStatusActive {
		return domain.Member{}, ErrMemberDisabled
	}
	return member, nil
}

// AuthenticateWithCSRF authenticates a session and validates its raw CSRF
// token in constant time. Missing, expired, and mismatched CSRF state all
// return ErrInvalidCSRFToken without including either secret in an error.
func (s *Service) AuthenticateWithCSRF(
	ctx context.Context,
	token string,
	csrfToken string,
) (domain.AuthenticatedSession, error) {
	if token == "" || csrfToken == "" {
		return domain.AuthenticatedSession{}, ErrInvalidCSRFToken
	}
	session, err := s.repository.GetSessionByTokenHash(ctx, tokenHash(token))
	if err != nil || !session.ExpiresAt.After(s.now()) {
		return domain.AuthenticatedSession{}, ErrInvalidCSRFToken
	}
	if subtle.ConstantTimeCompare(
		[]byte(tokenHash(csrfToken)),
		[]byte(session.CSRFHash),
	) != 1 {
		return domain.AuthenticatedSession{}, ErrInvalidCSRFToken
	}
	member, err := s.repository.GetMember(ctx, session.MemberID)
	if err != nil {
		return domain.AuthenticatedSession{}, ErrInvalidSession
	}
	if member.Status != domain.MemberStatusActive {
		return domain.AuthenticatedSession{}, ErrMemberDisabled
	}
	return domain.AuthenticatedSession{Member: member, CSRFToken: csrfToken}, nil
}

// Logout invalidates the server-side session associated with token.
func (s *Service) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.repository.DeleteSessionByTokenHash(ctx, tokenHash(token))
}

// ListMembers returns all allowlisted members.
func (s *Service) ListMembers(ctx context.Context) ([]domain.Member, error) {
	return s.repository.ListMembers(ctx)
}

// AddMember adds an active allowlisted member after validating email and role.
func (s *Service) AddMember(ctx context.Context, email string, role domain.MemberRole) (domain.Member, error) {
	email = normalizeEmail(email)
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email || !validRole(role) {
		return domain.Member{}, fmt.Errorf("invalid member email or role")
	}
	now := s.now()
	member := domain.Member{ID: domain.MemberID(s.newSecret()), Email: email, Role: role, Status: domain.MemberStatusActive, CreatedAt: now, UpdatedAt: now}
	if err := s.repository.CreateMember(ctx, member); err != nil {
		return domain.Member{}, err
	}
	return member, nil
}

// ChangeMember changes the role and active/disabled state of a member.
func (s *Service) ChangeMember(ctx context.Context, id domain.MemberID, role domain.MemberRole, status domain.MemberStatus) (domain.Member, error) {
	if !validRole(role) || (status != domain.MemberStatusActive && status != domain.MemberStatusDisabled) {
		return domain.Member{}, fmt.Errorf("invalid member role or status")
	}
	member, err := s.repository.GetMember(ctx, id)
	if err != nil {
		return domain.Member{}, err
	}
	member.Role, member.Status, member.UpdatedAt = role, status, s.now()
	return s.repository.UpdateMember(ctx, member)
}

func validRole(role domain.MemberRole) bool {
	return role == domain.MemberRoleMember || role == domain.MemberRoleAdmin
}

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func normalizeEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }
