package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/shxntanu/aether/backend/internal/domain"
	"github.com/shxntanu/aether/backend/internal/identity"
)

func TestGoogleLoginStartAndCallbackCreateSecureSession(t *testing.T) {
	backend := &fakeIdentityBackend{member: activeMember(domain.MemberRoleMember)}
	router := NewRouter(Options{Identity: backend, OIDC: backend, SecureCookies: true})

	start := httptest.NewRecorder()
	router.ServeHTTP(start, httptest.NewRequest(http.MethodGet, "/auth/google/start", nil))
	if start.Code != http.StatusFound || start.Header().Get("Location") != "https://accounts.google.test/authorize" {
		t.Fatalf("login start = %d %q", start.Code, start.Header().Get("Location"))
	}

	callback := httptest.NewRecorder()
	router.ServeHTTP(callback, httptest.NewRequest(http.MethodGet, "/auth/google/callback?state=valid&code=code", nil))
	if callback.Code != http.StatusSeeOther || callback.Header().Get("Location") != "/" {
		t.Fatalf("callback = %d %q", callback.Code, callback.Header().Get("Location"))
	}
	cookie := callback.Result().Cookies()[0]
	if cookie.Name != SessionCookieName || cookie.Value != "session-token" || !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("session cookie = %#v, want secure server session cookie", cookie)
	}
}

func TestSessionAuthorizationRejectsUnauthenticatedDisabledAndNonAdminMembers(t *testing.T) {
	tests := []struct {
		name, token string
		member      domain.Member
		authErr     error
		path        string
		want        int
	}{
		{name: "unauthenticated", path: "/api/v1/session", authErr: identity.ErrInvalidSession, want: http.StatusUnauthorized},
		{name: "disabled", token: "disabled", path: "/api/v1/session", authErr: identity.ErrMemberDisabled, want: http.StatusForbidden},
		{name: "member cannot administer", token: "member", member: activeMember(domain.MemberRoleMember), path: "/api/v1/admin/members", want: http.StatusForbidden},
		{name: "admin can administer", token: "admin", member: activeMember(domain.MemberRoleAdmin), path: "/api/v1/admin/members", want: http.StatusOK},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			backend := &fakeIdentityBackend{member: test.member, authErr: test.authErr}
			router := NewRouter(Options{Identity: backend, OIDC: backend})
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			if test.token != "" {
				request.AddCookie(&http.Cookie{Name: SessionCookieName, Value: test.token})
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != test.want {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, test.want, response.Body.String())
			}
		})
	}
}

func TestSessionResponseUsesPublicMemberFieldsOnly(t *testing.T) {
	member := activeMember(domain.MemberRoleMember)
	member.OIDCSubject = "private-google-subject"
	request := httptest.NewRequest(http.MethodGet, "/api/v1/session", nil)
	request.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "member"})
	response := httptest.NewRecorder()
	backend := &fakeIdentityBackend{member: member}
	NewRouter(Options{Identity: backend, OIDC: backend}).ServeHTTP(response, request)
	body := response.Body.String()
	if !strings.Contains(body, `"email":"member@example.com"`) || strings.Contains(body, "private-google-subject") || strings.Contains(body, `"Email"`) {
		t.Fatalf("session body exposes wrong member representation: %s", body)
	}
}

func TestAdminCanAddAndDisableMember(t *testing.T) {
	backend := &fakeIdentityBackend{member: activeMember(domain.MemberRoleAdmin)}
	router := NewRouter(Options{Identity: backend, OIDC: backend})

	createRequest := httptest.NewRequest(http.MethodPost, "/api/v1/admin/members", strings.NewReader(`{"email":"Family@Example.com","role":"member"}`))
	createRequest.Header.Set("Content-Type", "application/json")
	createRequest.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "admin"})
	createResponse := httptest.NewRecorder()
	router.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusCreated || backend.added.Email != "Family@Example.com" {
		t.Fatalf("create response=%d body=%s added=%#v", createResponse.Code, createResponse.Body.String(), backend.added)
	}

	patchRequest := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/members/member-1", strings.NewReader(`{"status":"disabled","role":"member"}`))
	patchRequest.Header.Set("Content-Type", "application/json")
	patchRequest.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "admin"})
	patchResponse := httptest.NewRecorder()
	router.ServeHTTP(patchResponse, patchRequest)
	if patchResponse.Code != http.StatusOK || backend.updated.Status != domain.MemberStatusDisabled {
		t.Fatalf("patch response=%d body=%s updated=%#v", patchResponse.Code, patchResponse.Body.String(), backend.updated)
	}
}

func TestLogoutDeletesServerSessionAndExpiresCookie(t *testing.T) {
	backend := &fakeIdentityBackend{member: activeMember(domain.MemberRoleMember)}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/logout", nil)
	request.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "session-token"})
	response := httptest.NewRecorder()
	NewRouter(Options{Identity: backend, OIDC: backend, SecureCookies: true}).ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || backend.loggedOut != "session-token" {
		t.Fatalf("logout = %d token=%q", response.Code, backend.loggedOut)
	}
	if cookie := response.Result().Cookies()[0]; cookie.MaxAge >= 0 || !cookie.Secure {
		t.Fatalf("expired cookie = %#v", cookie)
	}
}

type fakeIdentityBackend struct {
	member    domain.Member
	authErr   error
	added     domain.Member
	updated   domain.Member
	loggedOut string
}

func (f *fakeIdentityBackend) Start(context.Context) (string, error) {
	return "https://accounts.google.test/authorize", nil
}
func (f *fakeIdentityBackend) Complete(_ context.Context, state, code string) (identity.Identity, error) {
	if state != "valid" || code != "code" {
		return identity.Identity{}, identity.ErrInvalidOIDCFlow
	}
	return identity.Identity{Email: "member@example.com", EmailVerified: true}, nil
}
func (f *fakeIdentityBackend) CompleteLogin(context.Context, identity.Identity) (string, domain.Member, error) {
	return "session-token", f.member, nil
}
func (f *fakeIdentityBackend) Authenticate(context.Context, string) (domain.Member, error) {
	return f.member, f.authErr
}
func (f *fakeIdentityBackend) Logout(_ context.Context, token string) error {
	f.loggedOut = token
	return nil
}
func (f *fakeIdentityBackend) ListMembers(context.Context) ([]domain.Member, error) {
	return []domain.Member{f.member}, nil
}
func (f *fakeIdentityBackend) AddMember(_ context.Context, email string, role domain.MemberRole) (domain.Member, error) {
	f.added = domain.Member{Email: email, Role: role, Status: domain.MemberStatusActive}
	return f.added, nil
}
func (f *fakeIdentityBackend) ChangeMember(_ context.Context, id domain.MemberID, role domain.MemberRole, status domain.MemberStatus) (domain.Member, error) {
	f.updated = domain.Member{ID: id, Role: role, Status: status}
	return f.updated, nil
}

func activeMember(role domain.MemberRole) domain.Member {
	return domain.Member{ID: "member-1", Email: "member@example.com", DisplayName: "Family Member", Role: role, Status: domain.MemberStatusActive}
}
