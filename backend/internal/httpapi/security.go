package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"math"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
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
	// AccountLimiter is reserved for later authenticated account throttling.
	AccountLimiter *Limiter
	// IPLimiter throttles requests by resolved client address.
	IPLimiter *Limiter
	// RequestTimeout bounds ordinary request handling.
	RequestTimeout time.Duration
	// UploadTimeout bounds document upload request handling.
	UploadTimeout time.Duration
	// Logger records limiter rejections without logging raw client addresses.
	Logger *log.Logger
}

// Secure applies response hardening, safe client-IP rate limiting, and request
// deadlines around handler.
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

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setSecurityHeaders(w.Header())

		if denied, retryAfter := rateLimitRequest(
			r,
			options.TrustedProxyRanges,
			options.IPLimiter,
		); denied {
			logRateLimitRejection(options.Logger, "ip", r)
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
				writeRateLimitResponse(w, retryAfter)
				return
			}
		}

		timeout := requestTimeoutFor(r, requestTimeout, uploadTimeout)
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()
		handler.ServeHTTP(w, r.WithContext(ctx))
	})
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
	http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
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
