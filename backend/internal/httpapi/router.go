package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/shxntanu/aether/backend/internal/domain"
	"github.com/shxntanu/aether/backend/internal/identity"
	"github.com/shxntanu/aether/backend/internal/storage"
	"github.com/shxntanu/aether/backend/internal/vault"
)

// SessionCookieName is the browser cookie containing the opaque session token.
const SessionCookieName = "aether_session"

// IdentityService provides membership administration and session operations
// required by the HTTP handlers.
type IdentityService interface {
	// CompleteLogin creates a session for a verified, allowlisted identity.
	CompleteLogin(context.Context, identity.Identity) (string, domain.Member, error)
	// Authenticate resolves a session token to its current member.
	Authenticate(context.Context, string) (domain.Member, error)
	// Logout invalidates the session identified by a raw token.
	Logout(context.Context, string) error
	// ListMembers returns all allowlisted members.
	ListMembers(context.Context) ([]domain.Member, error)
	// AddMember creates an active allowlisted member.
	AddMember(context.Context, string, domain.MemberRole) (domain.Member, error)
	// ChangeMember updates a member's role and active or disabled status.
	ChangeMember(
		context.Context,
		domain.MemberID,
		domain.MemberRole,
		domain.MemberStatus,
	) (domain.Member, error)
}

// OIDCService starts and completes the browser-facing OIDC login flow.
type OIDCService interface {
	// Start creates a protected login flow and returns its provider URL.
	Start(context.Context) (string, error)
	// Complete validates the callback and returns the provider identity.
	Complete(context.Context, string, string) (identity.Identity, error)
}

// VaultService provides tagged document operations required by HTTP handlers.
type VaultService interface {
	// Upload stores an immutable original and its initial metadata.
	Upload(context.Context, vault.Upload) (vault.DocumentRecord, error)
	// Get returns a ready document and its tags.
	Get(context.Context, domain.DocumentID) (vault.DocumentRecord, error)
	// List returns ready documents matching optional tag filters.
	List(context.Context, []string, domain.TagMatch) ([]vault.DocumentRecord, error)
	// UpdateMetadata changes title and tags using optimistic concurrency.
	UpdateMetadata(
		context.Context,
		domain.DocumentID,
		vault.MetadataUpdate,
	) (vault.DocumentRecord, error)
	// ListTags returns reusable tags for autocomplete.
	ListTags(context.Context, string, int) ([]domain.Tag, error)
	// CreateTag creates or returns a reusable tag.
	CreateTag(context.Context, string) (domain.Tag, error)
	// OpenContent opens a whole document or requested byte range.
	OpenContent(
		context.Context,
		domain.DocumentID,
		*storage.ByteRange,
	) (vault.Content, error)
	// Delete soft-deletes a ready document and is idempotent for deleted rows.
	Delete(context.Context, domain.DocumentID) error
	// Restore returns a deleted document to ready state.
	Restore(context.Context, domain.DocumentID) (vault.DocumentRecord, error)
	// Purge permanently removes a deleted document under the supplied policy.
	Purge(context.Context, domain.DocumentID, vault.PurgePolicy) error
}

// Options supplies optional application services to NewRouter.
type Options struct {
	// Identity authenticates sessions and manages membership.
	Identity IdentityService
	// OIDC starts and completes Google login when configured.
	OIDC OIDCService
	// Vault enables authenticated document and tag routes when configured.
	Vault VaultService
	// SecureCookies restricts session cookies to HTTPS.
	SecureCookies bool
}

// NewRouter creates the public API router. With no options it exposes only the
// health endpoint; each configured service enables its corresponding routes.
func NewRouter(options ...Options) http.Handler {
	var opts Options
	if len(options) > 0 {
		opts = options[0]
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	if opts.Identity == nil {
		return mux
	}
	if opts.OIDC != nil {
		registerOIDCRoutes(mux, opts)
	}
	registerIdentityRoutes(mux, opts)
	if opts.Vault != nil {
		registerVaultRoutes(mux, opts.Identity, opts.Vault)
	}
	return mux
}

func registerOIDCRoutes(mux *http.ServeMux, opts Options) {
	mux.HandleFunc("GET /auth/google/start", func(w http.ResponseWriter, r *http.Request) {
		location, err := opts.OIDC.Start(r.Context())
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "login_unavailable")
			return
		}
		http.Redirect(w, r, location, http.StatusFound)
	})
	mux.HandleFunc("GET /auth/google/callback", func(w http.ResponseWriter, r *http.Request) {
		claims, err := opts.OIDC.Complete(r.Context(), r.URL.Query().Get("state"), r.URL.Query().Get("code"))
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid_login")
			return
		}
		token, _, err := opts.Identity.CompleteLogin(r.Context(), claims)
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, identity.ErrNotAllowlisted) || errors.Is(err, identity.ErrMemberDisabled) || errors.Is(err, identity.ErrEmailUnverified) {
				status = http.StatusForbidden
			}
			writeError(w, status, "access_denied")
			return
		}
		http.SetCookie(w, sessionCookie(token, opts.SecureCookies, 7*24*60*60))
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
}

func registerIdentityRoutes(mux *http.ServeMux, opts Options) {
	mux.HandleFunc("POST /api/v1/logout", func(w http.ResponseWriter, r *http.Request) {
		cookie, _ := r.Cookie(SessionCookieName)
		if cookie != nil {
			_ = opts.Identity.Logout(r.Context(), cookie.Value)
		}
		http.SetCookie(w, sessionCookie("", opts.SecureCookies, -1))
		w.WriteHeader(http.StatusNoContent)
	})
	mux.Handle("GET /api/v1/session", requireMember(opts.Identity, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"member": memberFromContext(r.Context())})
	})))
	mux.Handle("GET /api/v1/admin/members", requireAdmin(opts.Identity, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		members, err := opts.Identity.ListMembers(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "catalog_error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"members": members})
	})))
	mux.Handle("POST /api/v1/admin/members", requireAdmin(opts.Identity, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Email string            `json:"email"`
			Role  domain.MemberRole `json:"role"`
		}
		if err := decodeJSON(r, &input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request")
			return
		}
		member, err := opts.Identity.AddMember(r.Context(), input.Email, input.Role)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, member)
	})))
	mux.Handle("PATCH /api/v1/admin/members/{id}", requireAdmin(opts.Identity, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Role   domain.MemberRole   `json:"role"`
			Status domain.MemberStatus `json:"status"`
		}
		if err := decodeJSON(r, &input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request")
			return
		}
		member, err := opts.Identity.ChangeMember(r.Context(), domain.MemberID(r.PathValue("id")), input.Role, input.Status)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, member)
	})))
}

type memberContextKey struct{}

func memberFromContext(ctx context.Context) domain.Member {
	member, _ := ctx.Value(memberContextKey{}).(domain.Member)
	return member
}
func requireMember(service IdentityService, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cc, err := r.Cookie(SessionCookieName)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "authentication_required")
			return
		}
		member, err := service.Authenticate(r.Context(), cc.Value)
		if err != nil {
			if errors.Is(err, identity.ErrMemberDisabled) {
				writeError(w, http.StatusForbidden, "membership_disabled")
			} else {
				writeError(w, http.StatusUnauthorized, "authentication_required")
			}
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), memberContextKey{}, member)))
	})
}
func requireAdmin(service IdentityService, next http.Handler) http.Handler {
	return requireMember(service, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if memberFromContext(r.Context()).Role != domain.MemberRoleAdmin {
			writeError(w, http.StatusForbidden, "administrator_required")
			return
		}
		next.ServeHTTP(w, r)
	}))
}
func sessionCookie(value string, secure bool, maxAge int) *http.Cookie {
	return &http.Cookie{Name: SessionCookieName, Value: value, Path: "/", HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode, MaxAge: maxAge}
}
func decodeJSON(r *http.Request, target any) error {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		return errors.New("content type")
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}
func writeDomainError(w http.ResponseWriter, err error) {
	if errors.Is(err, domain.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found")
	} else if errors.Is(err, domain.ErrAlreadyExists) {
		writeError(w, http.StatusConflict, "already_exists")
	} else {
		writeError(w, http.StatusBadRequest, "invalid_request")
	}
}
func writeError(w http.ResponseWriter, status int, code string) {
	writeJSON(w, status, map[string]string{"error": code})
}
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
