package __name__

import (
	"github.com/labstack/echo/v5"

	"example.com/acme/internal/platform/middleware"
)

func (m *Module) RegisterAPI(api *echo.Group, guard middleware.Guards) {
	g := api.Group("/__name__s", guard.Authenticated)

	g.GET("", m.api.List, guard.Require(Perm__Name__Read))
	g.POST("", m.api.Create, guard.Require(Perm__Name__Write))
	g.GET("/:__name__ID", m.api.Get, guard.Require(Perm__Name__Read))
}

// For HTML pages, add web_routes.go (RegisterWeb), web_handler.go, form.go
// and views/ following internal/feature/order.
