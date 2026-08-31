package identity

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"time"

	"github.com/shxntanu/aether/backend/internal/domain"
)

var ErrInvalidOIDCFlow = errors.New("OIDC login flow is invalid or expired")

type OIDCProvider interface {
	AuthorizationURL(state, nonce, codeChallenge string) string
	ExchangeAndVerify(ctx context.Context, code, codeVerifier, nonce string) (Identity, error)
}

type AuthFlowRepository interface {
	CreateAuthFlow(context.Context, domain.AuthFlow) error
	ConsumeAuthFlow(context.Context, string) (domain.AuthFlow, error)
}

type OIDCService struct {
	repository AuthFlowRepository
	provider   OIDCProvider
	now        func() time.Time
	newSecret  func() string
}

// NewOIDCService constructs the short-lived, server-side OIDC flow manager.
func NewOIDCService(repository AuthFlowRepository, provider OIDCProvider, now func() time.Time, newSecret func() string) *OIDCService {
	return &OIDCService{repository: repository, provider: provider, now: now, newSecret: newSecret}
}

// Start persists a state, nonce, and PKCE verifier and returns the provider URL.
func (s *OIDCService) Start(ctx context.Context) (string, error) {
	state, nonce, verifier := s.newSecret(), s.newSecret(), s.newSecret()
	now := s.now()
	flow := domain.AuthFlow{StateHash: tokenHash(state), Nonce: nonce, PKCEVerifier: verifier, CreatedAt: now, ExpiresAt: now.Add(10 * time.Minute)}
	if err := s.repository.CreateAuthFlow(ctx, flow); err != nil {
		return "", err
	}
	return s.provider.AuthorizationURL(state, nonce, pkceChallenge(verifier)), nil
}

// Complete atomically consumes state before exchanging and verifying the code.
func (s *OIDCService) Complete(ctx context.Context, state, code string) (Identity, error) {
	if state == "" || code == "" {
		return Identity{}, ErrInvalidOIDCFlow
	}
	flow, err := s.repository.ConsumeAuthFlow(ctx, tokenHash(state))
	if err != nil || !flow.ExpiresAt.After(s.now()) {
		return Identity{}, ErrInvalidOIDCFlow
	}
	return s.provider.ExchangeAndVerify(ctx, code, flow.PKCEVerifier, flow.Nonce)
}

func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
