package httpapi

import (
	"crypto/sha256"
	"crypto/subtle"
	"net"
	"net/http"
	"net/netip"
)

const (
	// GatewayTokenHeader carries the shared secret injected by the trusted edge.
	// Gateway removes this header before application handlers run.
	GatewayTokenHeader = "X-Aether-Gateway-Token"
	// GatewayClientIPHeader carries the single client address validated by the
	// trusted edge. Gateway removes this header before application handlers run.
	GatewayClientIPHeader = "X-Aether-Client-IP"
)

// Gateway restricts a backend handler to requests authenticated by the edge.
//
// A GET health check remains public for platform probes. Other requests must
// carry the configured secret and exactly one valid client IP. The validated
// address replaces RemoteAddr so downstream IP limits do not trust caller-
// controlled forwarding headers. An empty secret disables this boundary for
// local development.
func Gateway(next http.Handler, secret string) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	if secret == "" {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			removeGatewayHeaders(r.Header)
			next.ServeHTTP(w, r)
		})
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/health" {
			removeGatewayHeaders(r.Header)
			next.ServeHTTP(w, r)
			return
		}
		if !equalGatewaySecret(r.Header.Get(GatewayTokenHeader), secret) {
			writeError(w, http.StatusNotFound, "not_found")
			return
		}

		clientIP, ok := gatewayClientIP(r.Header.Values(GatewayClientIPHeader))
		if !ok {
			writeError(w, http.StatusBadRequest, "invalid_request")
			return
		}
		removeGatewayHeaders(r.Header)
		r.RemoteAddr = net.JoinHostPort(clientIP.String(), "0")
		next.ServeHTTP(w, r)
	})
}

func removeGatewayHeaders(header http.Header) {
	header.Del(GatewayTokenHeader)
	header.Del(GatewayClientIPHeader)
}

func equalGatewaySecret(candidate, expected string) bool {
	candidateHash := sha256.Sum256([]byte(candidate))
	expectedHash := sha256.Sum256([]byte(expected))
	return subtle.ConstantTimeCompare(candidateHash[:], expectedHash[:]) == 1
}

func gatewayClientIP(values []string) (netip.Addr, bool) {
	if len(values) != 1 {
		return netip.Addr{}, false
	}
	address, err := netip.ParseAddr(values[0])
	if err != nil || address.Zone() != "" {
		return netip.Addr{}, false
	}
	return address.Unmap(), true
}
