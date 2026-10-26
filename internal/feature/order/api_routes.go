package order

import (
	"github.com/labstack/echo/v5"

	"example.com/acme/internal/platform/middleware"
)

// RegisterAPI declares every JSON route of the feature, with its permission.
func (m *Module) RegisterAPI(api *echo.Group, guard middleware.Guards) {
	g := api.Group("/orders", guard.Authenticated)

	g.GET("", m.api.List, guard.Require(PermOrderRead))
	g.POST("", m.api.Create, guard.Require(PermOrderWrite))
	g.GET("/:orderID", m.api.Get, guard.Require(PermOrderRead))
	g.POST("/:orderID/cancel", m.api.Cancel, guard.Require(PermOrderWrite)) // action route

	items := g.Group("/:orderID/items")
	items.GET("", m.api.ListItems, guard.Require(PermOrderRead))
	items.POST("", m.api.AddItem, guard.Require(PermOrderWrite))
}
