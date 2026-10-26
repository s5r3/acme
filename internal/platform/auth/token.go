package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"example.com/acme/internal/platform/config"
	"example.com/acme/internal/shared/clock"
)

// ErrInvalidToken is returned for any malformed, expired or forged token.
var ErrInvalidToken = errors.New("invalid token")

type claims struct {
	Email string   `json:"email"`
	Name  string   `json:"name"`
	Roles []string `json:"roles"`
	jwt.RegisteredClaims
}

// Tokens issues and verifies the API's bearer tokens (HS256 JWTs).
type Tokens struct {
	secret []byte
	issuer string
	ttl    time.Duration
	roles  *Roles
	clock  clock.Clock
}

// NewTokens builds the token service from config.
func NewTokens(cfg config.AuthConfig, roles *Roles, clk clock.Clock) *Tokens {
	return &Tokens{secret: []byte(cfg.JWTSecret), issuer: cfg.Issuer, ttl: cfg.TokenTTL, roles: roles, clock: clk}
}

// Issue returns a signed token for the user and its expiry time.
func (t *Tokens) Issue(userID uuid.UUID, email, name string, roles []string) (string, time.Time, error) {
	now := t.clock.Now()
	exp := now.Add(t.ttl)
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		Email: email,
		Name:  name,
		Roles: roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			Issuer:    t.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	})
	signed, err := tok.SignedString(t.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign token: %w", err)
	}
	return signed, exp, nil
}

// Verify parses raw and returns the Principal it carries.
func (t *Tokens) Verify(raw string) (*Principal, error) {
	var c claims
	_, err := jwt.ParseWithClaims(raw, &c, func(*jwt.Token) (any, error) { return t.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(t.issuer),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(t.clock.Now),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	userID, err := uuid.Parse(c.Subject)
	if err != nil {
		return nil, fmt.Errorf("%w: bad subject", ErrInvalidToken)
	}
	return t.roles.NewPrincipal(userID, c.Email, c.Name, c.Roles), nil
}
