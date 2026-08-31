package identity

import (
	"context"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

type tokenVerifier interface {
	Verify(context.Context, string) (*oidc.IDToken, error)
}

type GoogleProvider struct {
	oauth2Config oauth2.Config
	verifier     tokenVerifier
}

// NewGoogleProvider discovers Google's endpoints and configures exact redirect
// URI and ID-token verification for the configured OAuth client.
func NewGoogleProvider(ctx context.Context, clientID, clientSecret, redirectURL string) (*GoogleProvider, error) {
	provider, err := oidc.NewProvider(ctx, "https://accounts.google.com")
	if err != nil {
		return nil, fmt.Errorf("discover Google OIDC provider: %w", err)
	}
	return &GoogleProvider{
		oauth2Config: oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Endpoint:     provider.Endpoint(),
			Scopes:       []string{oidc.ScopeOpenID, "email", "profile"},
		},
		verifier: provider.Verifier(&oidc.Config{ClientID: clientID}),
	}, nil
}

// AuthorizationURL returns a Google authorization URL protected by state,
// nonce, and an S256 PKCE challenge.
func (p *GoogleProvider) AuthorizationURL(state, nonce, codeChallenge string) string {
	return p.oauth2Config.AuthCodeURL(state, oidc.Nonce(nonce), oauth2.SetAuthURLParam("code_challenge", codeChallenge), oauth2.SetAuthURLParam("code_challenge_method", "S256"))
}

// ExchangeAndVerify exchanges a code with its PKCE verifier and validates the
// ID token signature, audience, issuer, expiry, and nonce.
func (p *GoogleProvider) ExchangeAndVerify(ctx context.Context, code, codeVerifier, nonce string) (Identity, error) {
	token, err := p.oauth2Config.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", codeVerifier))
	if err != nil {
		return Identity{}, fmt.Errorf("exchange Google authorization code: %w", err)
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return Identity{}, fmt.Errorf("Google token response omitted id_token")
	}
	idToken, err := p.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return Identity{}, fmt.Errorf("verify Google ID token: %w", err)
	}
	if idToken.Nonce != nonce {
		return Identity{}, fmt.Errorf("Google ID token nonce mismatch")
	}
	var claims struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return Identity{}, fmt.Errorf("decode Google ID token claims: %w", err)
	}
	return Identity{Subject: idToken.Subject, Email: claims.Email, EmailVerified: claims.EmailVerified, DisplayName: claims.Name}, nil
}
