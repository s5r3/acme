// Package auth defines the identity that services see (Principal), the
// role -> permission registry, API tokens and password hashing. It does not
// import Echo: HTTP extraction lives in platform/middleware and platform/web.
package auth

import (
	"context"
	"slices"

	"github.com/google/uuid"
)

// Permission is a "resource:action" string declared by the feature that owns
// the resource, e.g. "order:write".
type Permission string

// Principal is the authenticated caller. Both front-ends produce the same
// type: the API from a bearer token, the web app from a session cookie.
type Principal struct {
	UserID      uuid.UUID
	Email       string
	Name        string
	Roles       []string
	Permissions map[Permission]struct{}
}

// Has reports whether p holds every listed permission.
func (p *Principal) Has(perms ...Permission) bool {
	if p == nil {
		return false
	}
	for _, perm := range perms {
		if _, ok := p.Permissions[perm]; !ok {
			return false
		}
	}
	return true
}

// HasRole is for display only (e.g. an "Admin" badge). Authorization code
// must check permissions, never role names.
func (p *Principal) HasRole(role string) bool {
	return p != nil && slices.Contains(p.Roles, role)
}

type principalKey struct{}

// WithPrincipal stores p in ctx.
func WithPrincipal(ctx context.Context, p *Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

// FromContext returns the caller, if authenticated.
func FromContext(ctx context.Context) (*Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(*Principal)
	return p, ok && p != nil
}
