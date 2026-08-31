// Command aether-gdrive-auth obtains a Google Drive refresh token through a
// short-lived loopback OAuth callback. It prints the token once and never
// writes it to disk.
package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

const driveScope = "https://www.googleapis.com/auth/drive.file"

func main() {
	clientID := flag.String(
		"client-id", os.Getenv("AETHER_GDRIVE_CLIENT_ID"), "Google OAuth client ID",
	)
	clientSecret := flag.String(
		"client-secret", os.Getenv("AETHER_GDRIVE_CLIENT_SECRET"),
		"Google OAuth client secret",
	)
	redirectURL := flag.String(
		"redirect-url", os.Getenv("AETHER_GDRIVE_REDIRECT_URL"), "loopback callback URL",
	)
	flag.Parse()
	if *clientID == "" || *clientSecret == "" || *redirectURL == "" {
		fatal("client-id, client-secret, and redirect-url are required")
	}
	parsed, err := url.Parse(*redirectURL)
	if err != nil || parsed.Scheme != "http" || parsed.Host == "" ||
		parsed.Path != "/auth/gdrive/callback" || !isLoopback(parsed.Hostname()) {
		fatal("redirect-url must be an HTTP loopback URL ending in /auth/gdrive/callback")
	}
	state, err := randomState()
	if err != nil {
		fatal(err.Error())
	}
	config := &oauth2.Config{
		ClientID:     *clientID,
		ClientSecret: *clientSecret,
		RedirectURL:  *redirectURL,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		},
		Scopes: []string{driveScope},
	}
	callback := make(chan callbackResult, 1)
	server := &http.Server{ReadHeaderTimeout: 5 * time.Second}
	mux := http.NewServeMux()
	mux.HandleFunc(parsed.Path, func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Query().Get("state") != state {
			callback <- callbackResult{err: errors.New("OAuth state mismatch")}
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		if providerError := request.URL.Query().Get("error"); providerError != "" {
			callback <- callbackResult{err: fmt.Errorf("Google authorization failed: %s", providerError)}
			_, _ = writer.Write([]byte("Authorization failed; you may close this window."))
			return
		}
		callback <- callbackResult{code: request.URL.Query().Get("code")}
		_, _ = writer.Write([]byte("Authorization complete; you may close this window."))
	})
	server.Handler = mux
	listener, err := net.Listen("tcp", parsed.Host)
	if err != nil {
		fatal(err.Error())
	}
	go func() { _ = server.Serve(listener) }()
	fmt.Println("Open this URL in the vault-owner browser:")
	fmt.Println(config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce))
	var result callbackResult
	select {
	case result = <-callback:
	case <-time.After(5 * time.Minute):
		fatal("authorization timed out")
	}
	shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownContext)
	if result.err != nil {
		fatal(result.err.Error())
	}
	if result.code == "" {
		fatal("authorization callback omitted code")
	}
	token, err := config.Exchange(context.Background(), result.code)
	if err != nil {
		fatal("exchange authorization code: " + err.Error())
	}
	if token.RefreshToken == "" {
		fatal("Google did not return a refresh token; revoke prior consent and retry")
	}
	fmt.Println("AETHER_GDRIVE_REFRESH_TOKEN=" + token.RefreshToken)
}

type callbackResult struct {
	code string
	err  error
}

func randomState() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func isLoopback(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func fatal(message string) {
	_, _ = fmt.Fprintln(os.Stderr, strings.TrimSpace(message))
	os.Exit(1)
}
