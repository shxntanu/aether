package identity

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shxntanu/aether/backend/internal/audit"
	"github.com/shxntanu/aether/backend/internal/domain"
)

func TestBootstrapAdminCreatesConfiguredAdministrator(t *testing.T) {
	repository := newMemoryRepository()
	service := NewService(repository, time.Now, func() string { return "generated-id" })

	member, err := service.BootstrapAdmin(context.Background(), " First.Admin@Example.com ")
	if err != nil {
		t.Fatalf("BootstrapAdmin() error = %v", err)
	}
	if member.Email != "first.admin@example.com" || member.Role != domain.MemberRoleAdmin || member.Status != domain.MemberStatusActive {
		t.Fatalf("BootstrapAdmin() = %#v, want normalized active administrator", member)
	}
}

func TestBootstrapAdminPromotesExistingMemberWithoutChangingStatus(t *testing.T) {
	repository := newMemoryRepository()
	repository.members["family@example.com"] = domain.Member{ID: "member-1", Email: "family@example.com", Role: domain.MemberRoleMember, Status: domain.MemberStatusDisabled}
	service := NewService(repository, time.Now, func() string { return "unused" })

	member, err := service.BootstrapAdmin(context.Background(), "family@example.com")
	if err != nil {
		t.Fatalf("BootstrapAdmin() error = %v", err)
	}
	if member.Role != domain.MemberRoleAdmin || member.Status != domain.MemberStatusDisabled {
		t.Fatalf("BootstrapAdmin() = %#v, want disabled administrator", member)
	}
}

func TestCompleteLoginAllowsOnlyActiveAllowlistedMember(t *testing.T) {
	now := time.Date(2026, 8, 31, 10, 0, 0, 0, time.UTC)
	repository := newMemoryRepository()
	repository.members["allowed@example.com"] = domain.Member{ID: "member-1", Email: "allowed@example.com", DisplayName: "Old Name", Role: domain.MemberRoleMember, Status: domain.MemberStatusActive}
	service := NewService(repository, func() time.Time { return now }, func() string { return "session-id" })

	token, member, err := service.CompleteLogin(context.Background(), Identity{Subject: "google-subject", Email: "Allowed@Example.com", EmailVerified: true, DisplayName: "New Name"})
	if err != nil {
		t.Fatalf("CompleteLogin() error = %v", err)
	}
	if token == "" || member.DisplayName != "New Name" {
		t.Fatalf("CompleteLogin() token/member = %q/%#v, want session and refreshed name", token, member)
	}
	if len(repository.sessions) != 1 || repository.sessions[0].TokenHash == token {
		t.Fatalf("stored session = %#v, want one hashed token", repository.sessions)
	}
}

func TestCompleteLoginRejectsUnverifiedUnknownAndDisabledAccounts(t *testing.T) {
	tests := []struct {
		name     string
		identity Identity
		member   *domain.Member
		want     error
	}{
		{name: "unverified", identity: Identity{Email: "member@example.com"}, want: ErrEmailUnverified},
		{name: "unknown", identity: Identity{Email: "unknown@example.com", EmailVerified: true}, want: ErrNotAllowlisted},
		{name: "disabled", identity: Identity{Email: "member@example.com", EmailVerified: true}, member: &domain.Member{ID: "disabled", Email: "member@example.com", Status: domain.MemberStatusDisabled}, want: ErrMemberDisabled},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := newMemoryRepository()
			if test.member != nil {
				repository.members[test.member.Email] = *test.member
			}
			service := NewService(repository, time.Now, func() string { return "id" })
			_, _, err := service.CompleteLogin(context.Background(), test.identity)
			if !errors.Is(err, test.want) {
				t.Fatalf("CompleteLogin() error = %v, want %v", err, test.want)
			}
			if len(repository.sessions) != 0 {
				t.Fatal("rejected login created a session")
			}
		})
	}
}

func TestAuthenticateEnforcesSessionExpiryAndCurrentMembershipStatus(t *testing.T) {
	now := time.Date(2026, 8, 31, 10, 0, 0, 0, time.UTC)
	repository := newMemoryRepository()
	repository.members["member@example.com"] = domain.Member{ID: "member-1", Email: "member@example.com", Role: domain.MemberRoleMember, Status: domain.MemberStatusActive}
	service := NewService(repository, func() time.Time { return now }, func() string { return "raw-token" })
	token, _, err := service.CompleteLogin(context.Background(), Identity{Email: "member@example.com", EmailVerified: true})
	if err != nil {
		t.Fatal(err)
	}

	member, err := service.Authenticate(context.Background(), token)
	if err != nil || member.ID != "member-1" {
		t.Fatalf("Authenticate() = %#v, %v, want active member", member, err)
	}
	repository.members["member@example.com"] = domain.Member{ID: "member-1", Email: "member@example.com", Status: domain.MemberStatusDisabled}
	_, err = service.Authenticate(context.Background(), token)
	if !errors.Is(err, ErrMemberDisabled) {
		t.Fatalf("Authenticate() error = %v, want ErrMemberDisabled", err)
	}
}

func TestMemberManagementValidatesRolesStatusesAndNormalizesEmail(t *testing.T) {
	repository := newMemoryRepository()
	service := NewService(
		repository,
		time.Now,
		sequenceSecrets("member-id"),
		discardAuditRecorder{},
	)
	member, err := service.AddMember(context.Background(), " Family@Example.com ", domain.MemberRoleMember)
	if err != nil {
		t.Fatalf("AddMember() error = %v", err)
	}
	if member.Email != "family@example.com" || member.Status != domain.MemberStatusActive {
		t.Fatalf("AddMember() = %#v", member)
	}

	member, err = service.ChangeMember(context.Background(), member.ID, domain.MemberRoleAdmin, domain.MemberStatusDisabled)
	if err != nil {
		t.Fatalf("ChangeMember() error = %v", err)
	}
	if member.Role != domain.MemberRoleAdmin || member.Status != domain.MemberStatusDisabled {
		t.Fatalf("ChangeMember() = %#v", member)
	}

	if _, err := service.AddMember(context.Background(), "invalid", domain.MemberRole("owner")); err == nil {
		t.Fatal("AddMember() accepted invalid email and role")
	}
	if _, err := service.ChangeMember(context.Background(), member.ID, domain.MemberRoleMember, domain.MemberStatus("pending")); err == nil {
		t.Fatal("ChangeMember() accepted invalid status")
	}
}

type discardAuditRecorder struct{}

func (discardAuditRecorder) Record(context.Context, audit.Event) error {
	return nil
}

func TestLogoutDeletesHashedSessionToken(t *testing.T) {
	repository := newMemoryRepository()
	service := NewService(repository, time.Now, func() string { return "unused" })
	if err := service.Logout(context.Background(), "raw-token"); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if repository.deletedHash == "raw-token" || repository.deletedHash != tokenHash("raw-token") {
		t.Fatalf("deleted hash = %q", repository.deletedHash)
	}
}

type memoryRepository struct {
	members     map[string]domain.Member
	sessions    []domain.Session
	deletedHash string
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{members: map[string]domain.Member{}}
}

func (r *memoryRepository) CreateMember(_ context.Context, member domain.Member) error {
	if _, exists := r.members[member.Email]; exists {
		return domain.ErrAlreadyExists
	}
	r.members[member.Email] = member
	return nil
}
func (r *memoryRepository) GetMemberByEmail(_ context.Context, email string) (domain.Member, error) {
	member, ok := r.members[email]
	if !ok {
		return domain.Member{}, domain.ErrNotFound
	}
	return member, nil
}
func (r *memoryRepository) GetMember(_ context.Context, id domain.MemberID) (domain.Member, error) {
	for _, member := range r.members {
		if member.ID == id {
			return member, nil
		}
	}
	return domain.Member{}, domain.ErrNotFound
}
func (r *memoryRepository) UpdateMember(_ context.Context, member domain.Member) (domain.Member, error) {
	found := false
	for email, current := range r.members {
		if current.ID == member.ID {
			delete(r.members, email)
			found = true
			break
		}
	}
	if !found && member.ID != "" {
		return domain.Member{}, domain.ErrNotFound
	}
	r.members[member.Email] = member
	return member, nil
}
func (r *memoryRepository) ListMembers(context.Context) ([]domain.Member, error) {
	result := make([]domain.Member, 0, len(r.members))
	for _, member := range r.members {
		result = append(result, member)
	}
	return result, nil
}
func (r *memoryRepository) CreateSession(_ context.Context, session domain.Session) error {
	r.sessions = append(r.sessions, session)
	return nil
}
func (r *memoryRepository) GetSessionByTokenHash(_ context.Context, hash string) (domain.Session, error) {
	for _, session := range r.sessions {
		if session.TokenHash == hash {
			return session, nil
		}
	}
	return domain.Session{}, domain.ErrNotFound
}
func (r *memoryRepository) DeleteSessionByTokenHash(_ context.Context, hash string) error {
	r.deletedHash = hash
	return nil
}
