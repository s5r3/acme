package app

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v5"
	echomw "github.com/labstack/echo/v5/middleware"

	"example.com/acme/internal/platform/auth"
	"example.com/acme/internal/platform/httpx"
	"example.com/acme/internal/platform/middleware"
	"example.com/acme/internal/platform/web"
	"example.com/acme/internal/platform/web/assets"
)

// RunWeb runs the HTML front-end until ctx is cancelled.
func RunWeb(ctx context.Context, cfgPath string) error {
	core, err := LoadCore(ctx, cfgPath, "web")
	if err != nil {
		return err
	}
	defer core.Close()

	go cleanupSessions(ctx, core)

	cfg := core.Config.Web
	core.Log.Info("web starting", "address", cfg.Address)
	sc := echo.StartConfig{Address: cfg.Address, GracefulTimeout: cfg.ShutdownTimeout, HideBanner: true, HidePort: true}
	err = sc.Start(ctx, NewWebServer(core))
	core.Log.Info("web stopped", "err", err)
	return err
}

// NewWebServer builds the web front-end's Echo instance.
func NewWebServer(c *Core) *echo.Echo {
	cfg := c.Config.Web
	e := echo.New()
	e.Logger = c.Log
	e.Validator = httpx.NewValidator()
	e.HTTPErrorHandler = web.ErrorHandler(c.Log)

	e.Use(
		middleware.RequestContext(c.Log),
		middleware.AccessLog(c.Log),
		middleware.Recover(),
		web.SecureHeaders,
		echomw.BodyLimit(cfg.BodyLimitBytes),
		middleware.Timeout(cfg.RequestTimeout),
	)

	// Static assets skip sessions and CSRF.
	e.StaticFS("/static", assets.FS, cacheFor(24*time.Hour))
	e.GET("/healthz", healthHandler(c))

	// Every page: CSRF token, logged-in user (if any), flash message.
	site := e.Group("", c.Sessions.Load, web.CSRF(cfg.CookieSecure), web.PageData)
	site.GET("/", home)

	// Sessions.Load already put the Principal in the context, so the guard
	// only has to check it is there.
	fromContext := middleware.AuthenticatorFunc(func(ec *echo.Context) (*auth.Principal, error) {
		p, _ := auth.FromContext(ec.Request().Context())
		return p, nil
	})
	guards := middleware.NewGuards(fromContext, web.RedirectToLogin)
	for _, m := range c.Modules {
		if wm, ok := m.(WebModule); ok {
			wm.RegisterWeb(site, guards)
		}
	}
	return e
}

func home(c *echo.Context) error {
	if _, ok := auth.FromContext(c.Request().Context()); ok {
		return c.Redirect(http.StatusSeeOther, "/orders")
	}
	return c.Redirect(http.StatusSeeOther, "/login")
}

func cacheFor(d time.Duration) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Response().Header().Set("Cache-Control", "public, max-age="+strconv.Itoa(int(d.Seconds())))
			return next(c)
		}
	}
}

// cleanupSessions deletes expired sessions every hour.
func cleanupSessions(ctx context.Context, c *Core) {
	t := time.NewTicker(time.Hour)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if n, err := c.Sessions.DeleteExpired(ctx); err != nil {
				c.Log.Error("delete expired sessions", "err", err)
			} else if n > 0 {
				c.Log.Info("expired sessions deleted", "count", n)
			}
		}
	}
}
