package httpapi

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"math"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/shxntanu/aether/backend/internal/audit"
)

const (
	defaultRequestTimeout = 30 * time.Second
	defaultUploadTimeout  = 10 * time.Minute
	loginStartPath        = "/auth/google/start"
	loginCallbackPath     = "/auth/google/callback"
	documentUploadPath    = "/api/v1/documents"
)

// SecurityOptions configures the HTTP hardening middleware returned by Secure.
type SecurityOptions struct {
	// PublicURL is the normalized browser-facing origin used by later checks.
	PublicURL *url.URL
	// TrustedProxyRanges enumerates peers allowed to supply forwarding headers.
	TrustedProxyRanges []*net.IPNet
	// LoginLimiter throttles browser login start and callback requests.
	LoginLimiter *Limiter
	// AccountLimiter throttles requests after authentication by member ID.
	AccountLimiter *Limiter
	// IPLimiter throttles requests by resolved client address.
	IPLimiter *Limiter
	// RequestTimeout bounds ordinary request handling.
	RequestTimeout time.Duration
	// UploadTimeout bounds document upload request handling.
	UploadTimeout time.Duration
	// Logger records limiter rejections without logging raw client addresses.
	Logger *log.Logger
	// Audit records rate-limit authorization rejections without request data.
	Audit audit.Recorder
}

// Secure composes the process-wide HTTP boundary around handler.
//
// Requests pass through panic recovery, security headers, request IDs,
// deadlines, client-IP/login limits, and finally the supplied route handler in
// that order. Authenticated member limits are applied by route middleware
// after the member has been resolved.
func Secure(handler http.Handler, options SecurityOptions) http.Handler {
	if handler == nil {
		handler = http.NotFoundHandler()
	}

	requestTimeout := options.RequestTimeout
	if requestTimeout <= 0 {
		requestTimeout = defaultRequestTimeout
	}
	uploadTimeout := options.UploadTimeout
	if uploadTimeout <= 0 {
		uploadTimeout = defaultUploadTimeout
	}

	secured := withRateLimits(
		handler,
		options,
	)
	secured = withDeadline(secured, requestTimeout, uploadTimeout)
	secured = withRequestID(secured)
	secured = withSecurityHeaders(secured)
	secured = withPanicRecovery(secured, options.Logger)
	return secured
}

type requestIDContextKey struct{}
type accountLimiterContextKey struct{}
type requestLoggerContextKey struct{}

var requestIDCounter atomic.Uint64

func withPanicRecovery(next http.Handler, logger *log.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := newRequestID()
		ctx := context.WithValue(r.Context(), requestIDContextKey{}, requestID)
		if logger != nil {
			ctx = context.WithValue(ctx, requestLoggerContextKey{}, logger)
		}
		r = r.WithContext(ctx)
		w.Header().Set("X-Request-ID", requestID)
		defer func() {
			if recover() != nil {
				if logger != nil {
					logger.Printf(
						"panic recovered request_id=%s method=%s path=%s",
						requestID,
						r.Method,
						r.URL.Path,
					)
				}
				writeError(w, http.StatusInternalServerError, "internal_error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setSecurityHeaders(w.Header())
		next.ServeHTTP(w, r)
	})
}

func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID, ok := r.Context().Value(requestIDContextKey{}).(string)
		if !ok || requestID == "" {
			requestID = newRequestID()
			r = r.WithContext(
				context.WithValue(r.Context(), requestIDContextKey{}, requestID),
			)
		}
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r)
	})
}

func withDeadline(
	next http.Handler,
	requestTimeout time.Duration,
	uploadTimeout time.Duration,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timeout := requestTimeoutFor(r, requestTimeout, uploadTimeout)
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func withRateLimits(next http.Handler, options SecurityOptions) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if denied, retryAfter := rateLimitRequest(
			r,
			options.TrustedProxyRanges,
			options.IPLimiter,
		); denied {
			logRateLimitRejection(options.Logger, "ip", r)
			recordAuthorizationRejectWithRecorder(
				options.Audit,
				r.Context(),
				"rate_limit:ip",
				nil,
			)
			writeRateLimitResponse(w, retryAfter)
			return
		}
		if isLoginRequest(r) {
			if denied, retryAfter := rateLimitRequest(
				r,
				options.TrustedProxyRanges,
				options.LoginLimiter,
			); denied {
				logRateLimitRejection(options.Logger, "login", r)
				recordAuthorizationRejectWithRecorder(
					options.Audit,
					r.Context(),
					"rate_limit:login",
					nil,
				)
				writeRateLimitResponse(w, retryAfter)
				return
			}
		}
		if unsafeAPIRequest(r) && !sameSiteRequest(r, options.PublicURL) {
			recordAuthorizationRejectWithRecorder(
				options.Audit,
				r.Context(),
				"csrf:origin",
				nil,
			)
			writeError(w, http.StatusForbidden, "csrf_required")
			return
		}

		ctx := context.WithValue(r.Context(), accountLimiterContextKey{}, options.AccountLimiter)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func newRequestID() string {
	var random [16]byte
	if _, err := rand.Read(random[:]); err == nil {
		return hex.EncodeToString(random[:])
	}
	return "fallback-" + strconv.FormatUint(requestIDCounter.Add(1), 10)
}

func accountLimiterFromContext(ctx context.Context) *Limiter {
	limiter, _ := ctx.Value(accountLimiterContextKey{}).(*Limiter)
	return limiter
}

func requestLoggerFromContext(ctx context.Context) *log.Logger {
	logger, _ := ctx.Value(requestLoggerContextKey{}).(*log.Logger)
	return logger
}

func setSecurityHeaders(header http.Header) {
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("Referrer-Policy", "no-referrer")
	header.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
	header.Set("Cross-Origin-Opener-Policy", "same-origin")
	header.Set(
		"Content-Security-Policy",
		"default-src 'self'; img-src 'self' data:; style-src 'self'; "+
			"script-src 'self'; connect-src 'self'; frame-ancestors 'none'; "+
			"base-uri 'none'; form-action 'self'",
	)
}

func rateLimitRequest(
	r *http.Request,
	trustedProxyRanges []*net.IPNet,
	limiter *Limiter,
) (bool, time.Duration) {
	if limiter == nil {
		return false, 0
	}
	key, ok := clientAddressKey(r, trustedProxyRanges)
	if !ok {
		return true, 0
	}
	allowed, retryAfter := limiter.Allow(key)
	return !allowed, retryAfter
}

func clientAddressKey(r *http.Request, trustedProxyRanges []*net.IPNet) (string, bool) {
	peerIP, ok := parseIPFromRemoteAddr(r.RemoteAddr)
	if !ok {
		return "", false
	}

	clientIP := peerIP
	if ipInTrustedRanges(peerIP, trustedProxyRanges) {
		if forwardedIP, ok := forwardedClientIP(
			r.Header.Values("X-Forwarded-For"),
			trustedProxyRanges,
		); ok {
			clientIP = forwardedIP
		}
	}

	hash := sha256.Sum256([]byte(clientIP.String()))
	return hex.EncodeToString(hash[:]), true
}

func parseIPFromRemoteAddr(remoteAddr string) (net.IP, bool) {
	host := strings.TrimSpace(remoteAddr)
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	}
	ip := net.ParseIP(host)
	return ip, ip != nil
}

func forwardedClientIP(values []string, trustedProxyRanges []*net.IPNet) (net.IP, bool) {
	var chain []net.IP
	for _, value := range values {
		for _, entry := range strings.Split(value, ",") {
			candidate := net.ParseIP(strings.TrimSpace(entry))
			if candidate == nil {
				return nil, false
			}
			chain = append(chain, candidate)
		}
	}
	for index := len(chain) - 1; index >= 0; index-- {
		if !ipInTrustedRanges(chain[index], trustedProxyRanges) {
			return chain[index], true
		}
	}
	return nil, false
}

func ipInTrustedRanges(ip net.IP, trustedProxyRanges []*net.IPNet) bool {
	for _, trustedProxyRange := range trustedProxyRanges {
		if trustedProxyRange.Contains(ip) {
			return true
		}
	}
	return false
}

func isLoginRequest(r *http.Request) bool {
	if r.Method != http.MethodGet {
		return false
	}
	return r.URL.Path == loginStartPath || r.URL.Path == loginCallbackPath
}

func requestTimeoutFor(
	r *http.Request,
	requestTimeout time.Duration,
	uploadTimeout time.Duration,
) time.Duration {
	if r.Method == http.MethodPost && r.URL.Path == documentUploadPath {
		contentType := r.Header.Get("Content-Type")
		if strings.HasPrefix(contentType, "multipart/form-data") {
			return uploadTimeout
		}
	}
	return requestTimeout
}

func writeRateLimitResponse(w http.ResponseWriter, retryAfter time.Duration) {
	if retryAfter > 0 {
		seconds := int(math.Ceil(retryAfter.Seconds()))
		if seconds < 1 {
			seconds = 1
		}
		w.Header().Set("Retry-After", strconv.Itoa(seconds))
	}
	writeError(w, http.StatusTooManyRequests, "rate_limited")
}

func sameSiteRequest(r *http.Request, publicURL *url.URL) bool {
	if strings.EqualFold(strings.TrimSpace(r.Header.Get("Sec-Fetch-Site")), "cross-site") {
		return false
	}
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	if publicURL == nil {
		return false
	}
	requestOrigin, ok := normalizeOrigin(origin)
	if !ok {
		return false
	}
	publicOrigin, ok := normalizeOrigin(publicURL.String())
	return ok && requestOrigin == publicOrigin
}

func normalizeOrigin(raw string) (string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" ||
		parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" ||
		(parsed.Path != "" && parsed.Path != "/") {
		return "", false
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", false
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "" {
		return "", false
	}
	port := parsed.Port()
	if (scheme == "http" && port == "80") ||
		(scheme == "https" && port == "443") {
		port = ""
	}
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	if port != "" {
		host += ":" + port
	}
	return scheme + "://" + host, true
}

func logRateLimitRejection(logger *log.Logger, scope string, r *http.Request) {
	if logger == nil {
		return
	}
	logger.Printf(
		"rate limit rejected scope=%s method=%s path=%s",
		scope,
		r.Method,
		r.URL.Path,
	)
}
