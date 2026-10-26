package app

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v5"
	echomw "github.com/labstack/echo/v5/middleware"

	"example.com/acme/api/openapi"
	"example.com/acme/internal/platform/httpx"
	"example.com/acme/internal/platform/middleware"
)

// RunAPI runs the JSON API until ctx is cancelled, then shuts down gracefully.
func RunAPI(ctx context.Context, cfgPath string) error {
	core, err := LoadCore(ctx, cfgPath, "api")
	if err != nil {
		return err
	}
	defer core.Close()

	cfg := core.Config.API
	core.Log.Info("api starting", "address", cfg.Address)
	sc := echo.StartConfig{Address: cfg.Address, GracefulTimeout: cfg.ShutdownTimeout, HideBanner: true, HidePort: true}
	err = sc.Start(ctx, NewAPIServer(core))
	core.Log.Info("api stopped", "err", err)
	return err
}

// NewAPIServer builds the API's Echo instance. Integration tests call it
// directly and drive it with httptest.
func NewAPIServer(c *Core) *echo.Echo {
	cfg := c.Config.API
	e := echo.New()
	e.Logger = c.Log
	e.Validator = httpx.NewValidator()
	e.HTTPErrorHandler = httpx.ErrorHandler(c.Log)

	e.Use(
		middleware.RequestContext(c.Log),
		middleware.AccessLog(c.Log),
		middleware.Recover(),
	)
	if len(cfg.CORSOrigins) > 0 {
		e.Use(echomw.CORS(cfg.CORSOrigins...))
	}
	e.Use(
		echomw.BodyLimit(cfg.BodyLimitBytes),
		middleware.Timeout(cfg.RequestTimeout),
	)

	e.GET("/healthz", healthHandler(c))
	e.StaticFS("/api/openapi", openapi.FS) // spec files, $refs resolve relative to openapi.yaml

	v1 := e.Group("/api/v1")
	guards := middleware.NewGuards(middleware.BearerAuthenticator(c.Tokens), middleware.Unauthorized)
	for _, m := range c.Modules {
		if am, ok := m.(APIModule); ok {
			am.RegisterAPI(v1, guards)
		}
	}
	return e
}

func healthHandler(c *Core) echo.HandlerFunc {
	return func(ec *echo.Context) error {
		sqlDB, err := c.DB.DB()
		if err == nil {
			err = sqlDB.PingContext(ec.Request().Context())
		}
		if err != nil {
			c.Log.Error("health check failed", "err", err)
			return ec.JSON(http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
		}
		return ec.JSON(http.StatusOK, map[string]string{"status": "ok", "version": Version})
	}
}
