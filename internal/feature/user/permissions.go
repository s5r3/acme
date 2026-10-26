package user

import "example.com/acme/internal/platform/auth"

// Permissions owned by the user feature.
const (
	PermUserRead auth.Permission = "user:read" // list and view other users
)

// RegisterPermissions grants this feature's permissions to roles.
// Customers can always see their own profile through /users/me.
func (m *Module) RegisterPermissions(r *auth.Roles) {
	r.Grant(auth.RoleAdmin, PermUserRead)
}
