package httpapi

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/shxntanu/aether/backend/internal/audit"
	"github.com/shxntanu/aether/backend/internal/domain"
	"github.com/shxntanu/aether/backend/internal/identity"
	"github.com/shxntanu/aether/backend/internal/storage"
	"github.com/shxntanu/aether/backend/internal/vault"
)

// SessionCookieName is the browser cookie containing the opaque session token.
const SessionCookieName = "aether_session"

// CSRFCookieName is the readable browser cookie containing the synchronizer
// token that clients copy into X-CSRF-Token for unsafe API requests.
const CSRFCookieName = "aether_csrf"

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

type csrfIdentityService interface {
	CompleteLoginWithCSRF(
		context.Context,
		identity.Identity,
	) (string, domain.AuthenticatedSession, error)
	AuthenticateWithCSRF(
		context.Context,
		string,
		string,
	) (domain.AuthenticatedSession, error)
}

type actorIdentityService interface {
	AddMemberWithActor(
		context.Context,
		domain.MemberID,
		string,
		domain.MemberRole,
	) (domain.Member, error)
	ChangeMemberWithActor(
		context.Context,
		domain.MemberID,
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
	// Delete queues a soft deletion and returns its complete current record.
	Delete(context.Context, domain.DocumentID) (vault.DocumentRecord, error)
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
	// Audit records security-sensitive successes and authorization rejections.
	// It is required whenever Identity is configured.
	Audit audit.Recorder
}

// healthOnlyAuditRecorder is used only by the deliberately unconfigured
// health-only router. Configured application routers never fall back to it.
type healthOnlyAuditRecorder struct{}

func (healthOnlyAuditRecorder) Record(context.Context, audit.Event) error { return nil }

// NewRouter creates the public API router. With no options it exposes only the
// health endpoint; each configured service enables its corresponding routes.
func NewRouter(options ...Options) http.Handler {
	var opts Options
	if len(options) > 0 {
		opts = options[0]
	} else {
		opts.Audit = healthOnlyAuditRecorder{}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	if opts.Identity == nil {
		if opts.Audit == nil {
			opts.Audit = healthOnlyAuditRecorder{}
		}
		return mux
	}
	if opts.OIDC != nil {
		registerOIDCRoutes(mux, opts)
	}
	registerIdentityRoutes(mux, opts)
	if opts.Vault != nil {
		registerVaultRoutes(mux, opts.Identity, opts.Vault, opts.Audit)
	}
	return mux
}

func registerOIDCRoutes(mux *http.ServeMux, opts Options) {
	mux.HandleFunc("GET /auth/google/start", func(w http.ResponseWriter, r *http.Request) {
		setNoStore(w)
		location, err := opts.OIDC.Start(r.Context())
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "login_unavailable")
			return
		}
		http.Redirect(w, r, location, http.StatusFound)
	})
	mux.HandleFunc("GET /auth/google/callback", func(w http.ResponseWriter, r *http.Request) {
		setNoStore(w)
		claims, err := opts.OIDC.Complete(
			r.Context(),
			r.URL.Query().Get("state"),
			r.URL.Query().Get("code"),
		)
		if err != nil {
			recordAuthorizationReject(opts, r.Context(), "login")
			writeError(w, http.StatusUnauthorized, "invalid_login")
			return
		}
		token, csrfToken, err := completeLogin(opts.Identity, r.Context(), claims)
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, identity.ErrNotAllowlisted) ||
				errors.Is(err, identity.ErrMemberDisabled) ||
				errors.Is(err, identity.ErrEmailUnverified) {
				status = http.StatusForbidden
			}
			recordAuthorizationReject(opts, r.Context(), "login")
			writeError(w, status, "access_denied")
			return
		}
		http.SetCookie(w, sessionCookie(token, opts.SecureCookies, 7*24*60*60))
		if csrfToken != "" {
			http.SetCookie(w, csrfCookie(csrfToken, opts.SecureCookies, 7*24*60*60))
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
}

func registerIdentityRoutes(mux *http.ServeMux, opts Options) {
	mux.Handle("POST /api/v1/logout", requireMember(opts.Identity, opts.Audit, requireCSRF(
		opts.Identity,
		opts.Audit,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			setNoStore(w)
			cookie, _ := r.Cookie(SessionCookieName)
			if cookie != nil {
				if err := opts.Identity.Logout(r.Context(), cookie.Value); err != nil {
					http.SetCookie(w, sessionCookie("", opts.SecureCookies, -1))
					writeError(w, http.StatusInternalServerError, "logout_failed")
					return
				}
			}
			http.SetCookie(w, sessionCookie("", opts.SecureCookies, -1))
			http.SetCookie(w, csrfCookie("", opts.SecureCookies, -1))
			w.WriteHeader(http.StatusNoContent)
		}),
	)))
	mux.Handle(
		"GET /api/v1/session",
		requireMember(
			opts.Identity,
			opts.Audit,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				setNoStore(w)
				body := map[string]any{"member": memberFromContext(r.Context())}
				if session, ok := authenticatedSessionFromContext(r.Context()); ok && session.CSRFToken != "" {
					body["csrfToken"] = session.CSRFToken
				}
				writeJSON(w, http.StatusOK, body)
			}),
		),
	)
	mux.Handle(
		"GET /api/v1/admin/members",
		requireAdmin(
			opts.Identity,
			opts.Audit,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				members, err := opts.Identity.ListMembers(r.Context())
				if err != nil {
					writeError(w, http.StatusInternalServerError, "catalog_error")
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{"members": members})
			}),
		),
	)
	mux.Handle(
		"POST /api/v1/admin/members",
		requireAdminWithCSRF(
			opts,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var input struct {
					Email string            `json:"email"`
					Role  domain.MemberRole `json:"role"`
				}
				if err := decodeJSON(r, &input); err != nil {
					writeError(w, http.StatusBadRequest, "invalid_request")
					return
				}
				actorID := memberFromContext(r.Context()).ID
				member, err := addMember(opts, r.Context(), actorID, input.Email, input.Role)
				if err != nil {
					writeDomainError(w, err)
					return
				}
				writeJSON(w, http.StatusCreated, member)
			}),
		),
	)
	mux.Handle(
		"PATCH /api/v1/admin/members/{id}",
		requireAdminWithCSRF(
			opts,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var input struct {
					Role   domain.MemberRole   `json:"role"`
					Status domain.MemberStatus `json:"status"`
				}
				if err := decodeJSON(r, &input); err != nil {
					writeError(w, http.StatusBadRequest, "invalid_request")
					return
				}
				actorID := memberFromContext(r.Context()).ID
				memberID := domain.MemberID(r.PathValue("id"))
				member, err := changeMember(
					opts,
					r.Context(),
					actorID,
					memberID,
					input.Role,
					input.Status,
				)
				if err != nil {
					writeDomainError(w, err)
					return
				}
				writeJSON(w, http.StatusOK, member)
			}),
		),
	)
}

type memberContextKey struct{}
type authenticatedSessionContextKey struct{}

func memberFromContext(ctx context.Context) domain.Member {
	member, _ := ctx.Value(memberContextKey{}).(domain.Member)
	return member
}

func authenticatedSessionFromContext(ctx context.Context) (domain.AuthenticatedSession, bool) {
	session, ok := ctx.Value(authenticatedSessionContextKey{}).(domain.AuthenticatedSession)
	return session, ok
}

func requireMember(
	service IdentityService,
	recorder audit.Recorder,
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setNoStore(w)
		cc, err := r.Cookie(SessionCookieName)
		if err != nil {
			recordAuthorizationRejectWithRecorder(recorder, r.Context(), "session", nil)
			writeError(w, http.StatusUnauthorized, "authentication_required")
			return
		}
		member, err := service.Authenticate(r.Context(), cc.Value)
		if err != nil {
			recordAuthorizationRejectWithRecorder(recorder, r.Context(), "session", nil)
			if errors.Is(err, identity.ErrMemberDisabled) {
				writeError(w, http.StatusForbidden, "membership_disabled")
			} else {
				writeError(w, http.StatusUnauthorized, "authentication_required")
			}
			return
		}
		if limiter := accountLimiterFromContext(r.Context()); limiter != nil {
			allowed, retryAfter := limiter.Allow(string(member.ID))
			if !allowed {
				logRateLimitRejection(requestLoggerFromContext(r.Context()), "member", r)
				recordAuthorizationRejectWithRecorder(
					recorder,
					r.Context(),
					"rate_limit:member",
					memberIDPointer(member.ID),
				)
				writeRateLimitResponse(w, retryAfter)
				return
			}
		}
		requestContext := context.WithValue(r.Context(), memberContextKey{}, member)
		if csrfCookie, csrfErr := r.Cookie(CSRFCookieName); csrfErr == nil {
			if csrfService, ok := service.(csrfIdentityService); ok {
				session, validateErr := csrfService.AuthenticateWithCSRF(
					r.Context(),
					cc.Value,
					csrfCookie.Value,
				)
				if validateErr == nil && session.Member.ID == member.ID {
					requestContext = context.WithValue(
						requestContext,
						authenticatedSessionContextKey{},
						session,
					)
				}
			}
		}
		next.ServeHTTP(w, r.WithContext(requestContext))
	})
}

func requireCSRF(
	service IdentityService,
	recorder audit.Recorder,
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !unsafeAPIRequest(r) {
			next.ServeHTTP(w, r)
			return
		}
		member := memberFromContext(r.Context())
		csrfCookie, cookieErr := r.Cookie(CSRFCookieName)
		csrfHeader := r.Header.Get("X-CSRF-Token")
		if cookieErr != nil || csrfCookie.Value == "" || csrfHeader == "" ||
			subtle.ConstantTimeCompare(
				[]byte(csrfCookie.Value),
				[]byte(csrfHeader),
			) != 1 {
			recordAuthorizationRejectWithRecorder(
				recorder,
				r.Context(),
				"csrf",
				memberIDPointer(member.ID),
			)
			writeError(w, http.StatusForbidden, "csrf_required")
			return
		}
		csrfService, ok := service.(csrfIdentityService)
		if !ok {
			recordAuthorizationRejectWithRecorder(
				recorder,
				r.Context(),
				"csrf",
				memberIDPointer(member.ID),
			)
			writeError(w, http.StatusForbidden, "csrf_required")
			return
		}
		if _, err := csrfService.AuthenticateWithCSRF(
			r.Context(),
			sessionToken(r),
			csrfCookie.Value,
		); err != nil {
			recordAuthorizationRejectWithRecorder(
				recorder,
				r.Context(),
				"csrf",
				memberIDPointer(member.ID),
			)
			writeError(w, http.StatusForbidden, "csrf_required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func requireAdminWithCSRF(opts Options, next http.Handler) http.Handler {
	return requireMember(opts.Identity, opts.Audit, requireCSRF(
		opts.Identity,
		opts.Audit,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if memberFromContext(r.Context()).Role != domain.MemberRoleAdmin {
				recordAuthorizationRejectWithRecorder(
					opts.Audit,
					r.Context(),
					"admin",
					memberIDPointer(memberFromContext(r.Context()).ID),
				)
				writeError(w, http.StatusForbidden, "administrator_required")
				return
			}
			next.ServeHTTP(w, r)
		}),
	))
}

func requireAdmin(
	service IdentityService,
	recorder audit.Recorder,
	next http.Handler,
) http.Handler {
	return requireMember(
		service,
		recorder,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if memberFromContext(r.Context()).Role != domain.MemberRoleAdmin {
				recordAuthorizationRejectWithRecorder(
					recorder,
					r.Context(),
					"admin",
					memberIDPointer(memberFromContext(r.Context()).ID),
				)
				writeError(w, http.StatusForbidden, "administrator_required")
				return
			}
			next.ServeHTTP(w, r)
		}),
	)
}

func completeLogin(
	service IdentityService,
	ctx context.Context,
	claims identity.Identity,
) (string, string, error) {
	if csrfService, ok := service.(csrfIdentityService); ok {
		token, session, err := csrfService.CompleteLoginWithCSRF(ctx, claims)
		if err != nil {
			return "", "", err
		}
		return token, session.CSRFToken, nil
	}
	token, _, err := service.CompleteLogin(ctx, claims)
	return token, "", err
}

func addMember(
	opts Options,
	ctx context.Context,
	actorID domain.MemberID,
	email string,
	role domain.MemberRole,
) (domain.Member, error) {
	if actorService, ok := opts.Identity.(actorIdentityService); ok {
		return actorService.AddMemberWithActor(ctx, actorID, email, role)
	}
	member, err := opts.Identity.AddMember(ctx, email, role)
	if err != nil {
		return domain.Member{}, err
	}
	if err := recordAudit(opts.Audit, ctx, audit.Event{
		ActorID:    memberIDPointer(actorID),
		Action:     audit.ActionMemberCreate,
		ObjectType: "member",
		ObjectID:   string(member.ID),
		Outcome:    domain.AuditOutcomeSucceeded,
	}); err != nil {
		return domain.Member{}, fmt.Errorf("record member creation: %w", err)
	}
	return member, nil
}

func changeMember(
	opts Options,
	ctx context.Context,
	actorID domain.MemberID,
	id domain.MemberID,
	role domain.MemberRole,
	status domain.MemberStatus,
) (domain.Member, error) {
	if actorService, ok := opts.Identity.(actorIdentityService); ok {
		return actorService.ChangeMemberWithActor(ctx, actorID, id, role, status)
	}
	member, err := opts.Identity.ChangeMember(ctx, id, role, status)
	if err != nil {
		return domain.Member{}, err
	}
	if err := recordAudit(opts.Audit, ctx, audit.Event{
		ActorID:    memberIDPointer(actorID),
		Action:     audit.ActionMemberChange,
		ObjectType: "member",
		ObjectID:   string(member.ID),
		Outcome:    domain.AuditOutcomeSucceeded,
	}); err != nil {
		return domain.Member{}, fmt.Errorf("record member change: %w", err)
	}
	return member, nil
}

func sessionCookie(value string, secure bool, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     SessionCookieName,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
	}
}

func csrfCookie(value string, secure bool, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     CSRFCookieName,
		Value:    value,
		Path:     "/api",
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   maxAge,
	}
}

func sessionToken(r *http.Request) string {
	cookie, _ := r.Cookie(SessionCookieName)
	if cookie == nil {
		return ""
	}
	return cookie.Value
}

func unsafeAPIRequest(r *http.Request) bool {
	if !strings.HasPrefix(r.URL.Path, "/api/") {
		return false
	}
	switch r.Method {
	case http.MethodPost, http.MethodPatch, http.MethodPut, http.MethodDelete:
		return true
	default:
		return false
	}
}

func setNoStore(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
}

func memberIDPointer(id domain.MemberID) *domain.MemberID {
	if id == "" {
		return nil
	}
	copy := id
	return &copy
}

func recordAuthorizationReject(opts Options, ctx context.Context, objectID string) {
	recordAuthorizationRejectWithRecorder(opts.Audit, ctx, objectID, nil)
}

func recordAuthorizationRejectWithRecorder(
	recorder audit.Recorder,
	ctx context.Context,
	objectID string,
	actorID *domain.MemberID,
) {
	if err := recordAudit(recorder, ctx, audit.Event{
		ActorID:    actorID,
		Action:     audit.ActionAuthorizationReject,
		ObjectType: "request",
		ObjectID:   objectID,
		Outcome:    domain.AuditOutcomeRejected,
	}); err != nil {
		logger := requestLoggerFromContext(ctx)
		if logger == nil {
			logger = log.Default()
		}
		logAuditFailure(logger, objectID, err)
	}
}

func recordAudit(recorder audit.Recorder, ctx context.Context, event audit.Event) error {
	if recorder == nil {
		return errors.New("audit recorder is not configured")
	}
	return recorder.Record(ctx, event)
}

func logAuditFailure(logger *log.Logger, route string, err error) {
	if logger == nil {
		return
	}
	logger.Printf(
		"audit failure route=%s action=authorization.reject error=%v",
		route,
		err,
	)
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
