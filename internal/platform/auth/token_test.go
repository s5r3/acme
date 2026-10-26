package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"example.com/acme/internal/platform/config"
	"example.com/acme/internal/shared/clock"
)

func TestTokenRoundTrip(t *testing.T) {
	roles := NewRoles()
	roles.Grant(RoleCustomer, "order:read")
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	cfg := config.AuthConfig{JWTSecret: "0123456789abcdef0123456789abcdef", Issuer: "test", TokenTTL: time.Hour}
	tokens := NewTokens(cfg, roles, clock.Fixed{T: now})

	uid := uuid.New()
	raw, exp, err := tokens.Issue(uid, "a@example.com", "A", []string{RoleCustomer})
	if err != nil || !exp.Equal(now.Add(time.Hour)) {
		t.Fatalf("issue: %v exp=%v", err, exp)
	}
	p, err := tokens.Verify(raw)
	if err != nil || p.UserID != uid || !p.Has("order:read") || p.Has("order:admin") {
		t.Fatalf("verify: %v %+v", err, p)
	}

	expired := NewTokens(cfg, roles, clock.Fixed{T: now.Add(2 * time.Hour)})
	if _, err := expired.Verify(raw); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expired token accepted: %v", err)
	}
	other := NewTokens(config.AuthConfig{JWTSecret: "ffffffffffffffffffffffffffffffff", Issuer: "test", TokenTTL: time.Hour}, roles, clock.Fixed{T: now})
	if _, err := other.Verify(raw); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("token with wrong key accepted: %v", err)
	}
}
