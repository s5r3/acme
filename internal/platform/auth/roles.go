package auth

import (
	"sync"

	"github.com/google/uuid"
)

// Role names stored on users.
const (
	RoleAdmin    = "admin"
	RoleCustomer = "customer"
)

// Roles maps role names to permissions. Features add their own grants at
// startup (see each feature's permissions.go), so this package never needs to
// know feature permission strings.
type Roles struct {
	mu     sync.RWMutex
	grants map[string]map[Permission]struct{}
}

// NewRoles returns an empty registry.
func NewRoles() *Roles {
	return &Roles{grants: map[string]map[Permission]struct{}{}}
}

// Grant gives perms to role.
func (r *Roles) Grant(role string, perms ...Permission) {
	r.mu.Lock()
	defer r.mu.Unlock()
	set, ok := r.grants[role]
	if !ok {
		set = map[Permission]struct{}{}
		r.grants[role] = set
	}
	for _, p := range perms {
		set[p] = struct{}{}
	}
}

// Expand returns the union of permissions held by roles.
func (r *Roles) Expand(roles []string) map[Permission]struct{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := map[Permission]struct{}{}
	for _, role := range roles {
		for p := range r.grants[role] {
			out[p] = struct{}{}
		}
	}
	return out
}

// NewPrincipal builds a Principal with permissions expanded from roles.
func (r *Roles) NewPrincipal(userID uuid.UUID, email, name string, roles []string) *Principal {
	return &Principal{UserID: userID, Email: email, Name: name, Roles: roles, Permissions: r.Expand(roles)}
}
