package user

import (
	"github.com/labstack/echo/v5"

	"example.com/acme/internal/platform/middleware"
)

// RegisterWeb declares every HTML route of the feature. These pages are
// public by design (you cannot log in while logged out otherwise).
func (m *Module) RegisterWeb(g *echo.Group, _ middleware.Guards) {
	g.GET("/login", m.web.LoginPage)
	g.POST("/login", m.web.Login)
	g.GET("/signup", m.web.SignupPage)
	g.POST("/signup", m.web.Signup)
	g.POST("/logout", m.web.Logout)
}
