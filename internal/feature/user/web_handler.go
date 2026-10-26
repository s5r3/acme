package user

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v5"

	"example.com/acme/internal/feature/user/views"
	"example.com/acme/internal/platform/httpx"
	"example.com/acme/internal/platform/web"
)

// webService is the subset of Service the HTML handlers use.
type webService interface {
	Register(ctx context.Context, in RegisterInput) (*User, error)
	Authenticate(ctx context.Context, email, password string) (*User, error)
}

type webHandler struct {
	svc      webService
	sessions SessionManager
}

func (h *webHandler) LoginPage(c *echo.Context) error {
	return web.Render(c, http.StatusOK, views.Login(views.LoginForm{Next: c.QueryParam("next")}, nil, ""))
}

func (h *webHandler) Login(c *echo.Context) error {
	var f LoginForm
	render := func(fields map[string]string, msg string) error {
		vm := views.LoginForm{Email: f.Email, Next: f.Next}
		return web.Render(c, http.StatusUnprocessableEntity, views.Login(vm, fields, msg))
	}
	if err := httpx.BindAndValidate(c, &f); err != nil {
		return web.HandleFormError(err, render)
	}
	u, err := h.svc.Authenticate(c.Request().Context(), f.Email, f.Password)
	if err != nil {
		return web.HandleFormError(err, render)
	}
	if err := h.sessions.Start(c, u.ID, u.Email, u.Name, u.Roles()); err != nil {
		return err
	}
	web.SetFlash(c, "success", "Welcome back, "+u.Name+".")
	return web.RedirectSeeOther(c, web.SafeNext(f.Next, "/orders"))
}

func (h *webHandler) SignupPage(c *echo.Context) error {
	return web.Render(c, http.StatusOK, views.Signup(views.SignupForm{}, nil, ""))
}

func (h *webHandler) Signup(c *echo.Context) error {
	var f SignupForm
	vm := func() views.SignupForm { return views.SignupForm{Email: f.Email, Name: f.Name} }
	render := func(fields map[string]string, msg string) error {
		return web.Render(c, http.StatusUnprocessableEntity, views.Signup(vm(), fields, msg))
	}
	if err := httpx.BindAndValidate(c, &f); err != nil {
		return web.HandleFormError(err, render)
	}
	u, err := h.svc.Register(c.Request().Context(), f.toInput())
	if err != nil {
		return web.HandleFormError(err, render)
	}
	if err := h.sessions.Start(c, u.ID, u.Email, u.Name, u.Roles()); err != nil {
		return err
	}
	web.SetFlash(c, "success", "Your account is ready.")
	return web.RedirectSeeOther(c, "/orders")
}

func (h *webHandler) Logout(c *echo.Context) error {
	if err := h.sessions.End(c); err != nil {
		return err
	}
	web.SetFlash(c, "info", "You have been logged out.")
	return web.RedirectSeeOther(c, "/login")
}
