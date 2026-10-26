package __name__

import "example.com/acme/internal/platform/auth"

const (
	Perm__Name__Read  auth.Permission = "__name__:read"
	Perm__Name__Write auth.Permission = "__name__:write"
)

func (m *Module) RegisterPermissions(r *auth.Roles) {
	r.Grant(auth.RoleAdmin, Perm__Name__Read, Perm__Name__Write)
	r.Grant(auth.RoleCustomer, Perm__Name__Read)
}
