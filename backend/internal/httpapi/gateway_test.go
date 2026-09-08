package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const testGatewaySecret = "01234567890123456789012345678901"

func TestGatewayValidSecretForwardsAndRemovesPrivateHeaders(t *testing.T) {
	var received *http.Request
	handler := Gateway(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = r
		w.WriteHeader(http.StatusNoContent)
	}), testGatewaySecret)
	request := gatewayRequest(http.MethodGet, "/api/v1/session", "192.0.2.10")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
	if received.RemoteAddr != "192.0.2.10:0" {
		t.Fatalf("remote address = %q, want validated client IP", received.RemoteAddr)
	}
	if received.Header.Get(GatewayTokenHeader) != "" ||
		received.Header.Get(GatewayClientIPHeader) != "" {
		t.Fatal("private gateway headers reached the application handler")
	}
}

func TestGatewayRejectsMissingOrIncorrectSecret(t *testing.T) {
	for _, token := range []string{"", "incorrect"} {
		request := httptest.NewRequest(http.MethodGet, "/api/v1/session", nil)
		request.Header.Set(GatewayTokenHeader, token)
		request.Header.Set(GatewayClientIPHeader, "192.0.2.10")
		response := httptest.NewRecorder()

		Gateway(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			t.Fatal("invalid token reached the application handler")
		}), testGatewaySecret).ServeHTTP(response, request)

		if response.Code != http.StatusNotFound {
			t.Fatalf("token %q returned %d, want 404", token, response.Code)
		}
	}
}

func TestGatewayAllowsHealthCheckWithoutPrivateHeaders(t *testing.T) {
	handler := Gateway(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), testGatewaySecret)
	response := httptest.NewRecorder()

	handler.ServeHTTP(
		response,
		httptest.NewRequest(http.MethodGet, "/api/v1/health", nil),
	)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
}

func TestGatewayRejectsMissingMalformedOrMultipleClientAddresses(t *testing.T) {
	for _, address := range []string{"", "not-an-ip", "192.0.2.1, 192.0.2.2"} {
		request := httptest.NewRequest(http.MethodGet, "/api/v1/session", nil)
		request.Header.Set(GatewayTokenHeader, testGatewaySecret)
		request.Header.Set(GatewayClientIPHeader, address)
		response := httptest.NewRecorder()

		Gateway(http.NotFoundHandler(), testGatewaySecret).ServeHTTP(response, request)

		if response.Code != http.StatusBadRequest {
			t.Fatalf("address %q returned %d, want 400", address, response.Code)
		}
	}
}

func TestGatewayClientAddressReachesIPLimiter(t *testing.T) {
	secured := Secure(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), SecurityOptions{
		IPLimiter: NewLimiter(1, time.Hour, time.Now),
	})
	handler := Gateway(secured, testGatewaySecret)

	for attempt, want := range []int{http.StatusNoContent, http.StatusTooManyRequests} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(
			response,
			gatewayRequest(http.MethodGet, "/api/v1/session", "2001:db8::1"),
		)
		if response.Code != want {
			t.Fatalf("attempt %d status = %d, want %d", attempt+1, response.Code, want)
		}
	}
}

func TestGatewayDisabledPreservesLocalRequests(t *testing.T) {
	handler := Gateway(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RemoteAddr != "192.0.2.20:4321" {
			t.Fatalf("remote address changed to %q", r.RemoteAddr)
		}
		if r.Header.Get(GatewayTokenHeader) != "" ||
			r.Header.Get(GatewayClientIPHeader) != "" {
			t.Fatal("disabled gateway retained private headers")
		}
		w.WriteHeader(http.StatusNoContent)
	}), "")
	request := httptest.NewRequest(http.MethodGet, "/api/v1/session", nil)
	request.RemoteAddr = "192.0.2.20:4321"
	request.Header.Set(GatewayTokenHeader, "local-value")
	request.Header.Set(GatewayClientIPHeader, "198.51.100.1")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
}

func gatewayRequest(method, path, clientIP string) *http.Request {
	request := httptest.NewRequest(method, path, nil)
	request.Header.Set(GatewayTokenHeader, testGatewaySecret)
	request.Header.Set(GatewayClientIPHeader, clientIP)
	return request
}
