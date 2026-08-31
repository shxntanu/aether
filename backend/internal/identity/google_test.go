package identity

import (
	"net/url"
	"testing"

	"golang.org/x/oauth2"
)

func TestGoogleAuthorizationURLContainsOIDCSecurityParametersAndExactRedirect(t *testing.T) {
	provider := &GoogleProvider{oauth2Config: oauth2.Config{ClientID: "client-id", RedirectURL: "https://vault.example/auth/google/callback", Endpoint: oauth2.Endpoint{AuthURL: "https://accounts.google.test/auth"}, Scopes: []string{"openid", "email", "profile"}}}
	location := provider.AuthorizationURL("state-value", "nonce-value", "challenge-value")
	parsed, err := url.Parse(location)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	wants := map[string]string{"state": "state-value", "nonce": "nonce-value", "code_challenge": "challenge-value", "code_challenge_method": "S256", "redirect_uri": "https://vault.example/auth/google/callback", "scope": "openid email profile"}
	for key, want := range wants {
		if got := query.Get(key); got != want {
			t.Errorf("query %s = %q, want %q", key, got, want)
		}
	}
}
