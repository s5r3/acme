package user

import (
	"github.com/labstack/echo/v5"

	"example.com/acme/internal/platform/middleware"
)

// RegisterAPI declares every JSON route of the feature.
func (m *Module) RegisterAPI(api *echo.Group, guard middleware.Guards) {
	// Public: no Authenticated guard, on purpose.
	api.POST("/auth/token", m.api.IssueToken)
	api.POST("/users", m.api.Create)

	g := api.Group("/users", guard.Authenticated)
	g.GET("/me", m.api.Me)
	g.GET("", m.api.List, guard.Require(PermUserRead))
	g.GET("/:userID", m.api.Get, guard.Require(PermUserRead))
}
