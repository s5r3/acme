package app

import (
	"github.com/labstack/echo/v5"

	"example.com/acme/internal/feature/order"
	"example.com/acme/internal/feature/user"
	"example.com/acme/internal/platform/auth"
	"example.com/acme/internal/platform/middleware"
)

// buildModules is the single list of features, shared by both binaries.
// Adding a feature = one new directory + one line here.
func buildModules(c *Core) []Module {
	users := user.New(user.Deps{DB: c.DB, Tokens: c.Tokens, Sessions: c.Sessions})
	orders := order.New(order.Deps{DB: c.DB, Tx: c.Tx, Users: users.Service()})

	return []Module{users, orders}
}

// Module is implemented by every feature.
type Module interface {
	Name() string
}

// APIModule is a feature with JSON routes (mounted by cmd/api).
type APIModule interface {
	Module
	RegisterAPI(g *echo.Group, guards middleware.Guards)
}

// WebModule is a feature with HTML routes (mounted by cmd/web).
type WebModule interface {
	Module
	RegisterWeb(g *echo.Group, guards middleware.Guards)
}

// PermissionModule is a feature that grants its permissions to roles.
type PermissionModule interface {
	RegisterPermissions(r *auth.Roles)
}
