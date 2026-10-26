package order

import "example.com/acme/internal/platform/auth"

// Permissions owned by the order feature.
const (
	PermOrderRead  auth.Permission = "order:read"  // see own orders
	PermOrderWrite auth.Permission = "order:write" // create and change own orders
	PermOrderAdmin auth.Permission = "order:admin" // see and change everyone's orders
)

// RegisterPermissions grants this feature's permissions to roles.
func (m *Module) RegisterPermissions(r *auth.Roles) {
	r.Grant(auth.RoleCustomer, PermOrderRead, PermOrderWrite)
	r.Grant(auth.RoleAdmin, PermOrderRead, PermOrderWrite, PermOrderAdmin)
}
