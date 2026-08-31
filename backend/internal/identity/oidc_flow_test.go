package identity

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/shxntanu/aether/backend/internal/domain"
)

func TestOIDCFlowUsesStateNonceAndPKCEAndConsumesFlow(t *testing.T) {
	now := time.Date(2026, 8, 31, 11, 0, 0, 0, time.UTC)
	repository := &memoryFlowRepository{}
	provider := &recordingProvider{identity: Identity{Subject: "google-1", Email: "member@example.com", EmailVerified: true}}
	service := NewOIDCService(repository, provider, func() time.Time { return now }, sequenceSecrets("state", "nonce", "verifier"))

	location, err := service.Start(context.Background())
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if location != "https://accounts.example/auth?state=state&nonce=nonce&challenge=iMnq5o6zALKXGivsnlom_0F5_WYda32GHkxlV7mq7hQ" {
		t.Fatalf("Start() location = %q", location)
	}
	if repository.flow.StateHash == "state" || repository.flow.Nonce != "nonce" || repository.flow.PKCEVerifier != "verifier" {
		t.Fatalf("stored flow = %#v, want hashed state and server-side nonce/verifier", repository.flow)
	}

	identity, err := service.Complete(context.Background(), "state", "authorization-code")
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if identity.Email != "member@example.com" || !repository.consumed {
		t.Fatalf("Complete() = %#v, consumed=%v", identity, repository.consumed)
	}
	if provider.code != "authorization-code" || provider.verifier != "verifier" || provider.nonce != "nonce" {
		t.Fatalf("provider received code/verifier/nonce %q/%q/%q", provider.code, provider.verifier, provider.nonce)
	}
}

func TestOIDCFlowRejectsUnknownExpiredAndReplayedState(t *testing.T) {
	now := time.Date(2026, 8, 31, 11, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		prepare func(*memoryFlowRepository)
	}{
		{name: "unknown", prepare: func(r *memoryFlowRepository) {}},
		{name: "expired", prepare: func(r *memoryFlowRepository) {
			r.flow = domain.AuthFlow{StateHash: tokenHash("state"), ExpiresAt: now.Add(-time.Second)}
		}},
		{name: "replayed", prepare: func(r *memoryFlowRepository) {
			r.flow = domain.AuthFlow{StateHash: tokenHash("state"), ExpiresAt: now.Add(time.Minute)}
			r.consumed = true
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &memoryFlowRepository{}
			test.prepare(repository)
			service := NewOIDCService(repository, &recordingProvider{}, func() time.Time { return now }, func() string { return "unused" })
			_, err := service.Complete(context.Background(), "state", "code")
			if !errors.Is(err, ErrInvalidOIDCFlow) {
				t.Fatalf("Complete() error = %v, want ErrInvalidOIDCFlow", err)
			}
		})
	}
}

type memoryFlowRepository struct {
	flow     domain.AuthFlow
	consumed bool
}

func (r *memoryFlowRepository) CreateAuthFlow(_ context.Context, flow domain.AuthFlow) error {
	r.flow = flow
	return nil
}
func (r *memoryFlowRepository) ConsumeAuthFlow(_ context.Context, stateHash string) (domain.AuthFlow, error) {
	if r.flow.StateHash != stateHash || r.consumed {
		return domain.AuthFlow{}, domain.ErrNotFound
	}
	r.consumed = true
	return r.flow, nil
}

type recordingProvider struct {
	identity              Identity
	code, verifier, nonce string
}

func (p *recordingProvider) AuthorizationURL(state, nonce, challenge string) string {
	return "https://accounts.example/auth?state=" + state + "&nonce=" + nonce + "&challenge=" + challenge
}
func (p *recordingProvider) ExchangeAndVerify(_ context.Context, code, verifier, nonce string) (Identity, error) {
	p.code, p.verifier, p.nonce = code, verifier, nonce
	return p.identity, nil
}

func sequenceSecrets(values ...string) func() string {
	index := 0
	return func() string { value := values[index]; index++; return value }
}

func TestPKCEChallengeIsBase64URLSHA256WithoutPadding(t *testing.T) {
	challenge := pkceChallenge("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk")
	if strings.Contains(challenge, "=") || challenge != "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM" {
		t.Fatalf("pkceChallenge() = %q", challenge)
	}
}
