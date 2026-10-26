package order

import (
	"github.com/labstack/echo/v5"

	"example.com/acme/internal/platform/middleware"
)

// RegisterWeb declares every HTML route of the feature. HTML forms cannot
// send PATCH/DELETE, so changes are POST action routes.
func (m *Module) RegisterWeb(root *echo.Group, guard middleware.Guards) {
	g := root.Group("/orders", guard.Authenticated)

	g.GET("", m.web.Index, guard.Require(PermOrderRead))
	g.GET("/new", m.web.New, guard.Require(PermOrderWrite))
	g.POST("", m.web.Create, guard.Require(PermOrderWrite))
	g.GET("/:orderID", m.web.Show, guard.Require(PermOrderRead))
	g.POST("/:orderID/cancel", m.web.Cancel, guard.Require(PermOrderWrite))
}
