// Package middleware holds HTTP middleware shared by both binaries: request
// IDs, access logs, panic recovery and the authentication/permission guards.
package middleware

import (
	"log/slog"
	"regexp"
	"time"

	"github.com/labstack/echo/v5"
	echomw "github.com/labstack/echo/v5/middleware"

	"example.com/acme/internal/platform/logger"
	"example.com/acme/internal/shared/id"
)

var validRequestID = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

// RequestContext assigns a request ID (reusing a sane incoming X-Request-Id),
// echoes it in the response and stores a request-scoped logger in the
// request context. It must be the first middleware.
func RequestContext(base *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			req := c.Request()
			rid := req.Header.Get(echo.HeaderXRequestID)
			if !validRequestID.MatchString(rid) {
				rid = id.New().String()
			}
			c.Response().Header().Set(echo.HeaderXRequestID, rid)

			ctx := logger.WithContext(req.Context(), base.With("request_id", rid))
			c.SetRequest(req.WithContext(ctx))
			return next(c)
		}
	}
}

// Recover turns panics into 500 errors handled by the binary's error handler.
func Recover() echo.MiddlewareFunc {
	return echomw.Recover()
}

// Timeout puts a deadline on the request context; services and GORM honour it.
func Timeout(d time.Duration) echo.MiddlewareFunc {
	return echomw.ContextTimeout(d)
}

// AccessLog writes one line per request. HandleError runs the binary's error
// handler first, so the logged status is the one the client received.
func AccessLog(base *slog.Logger) echo.MiddlewareFunc {
	return echomw.RequestLoggerWithConfig(echomw.RequestLoggerConfig{
		HandleError:  true,
		LogLatency:   true,
		LogMethod:    true,
		LogURIPath:   true,
		LogRoutePath: true,
		LogStatus:    true,
		LogRemoteIP:  true,
		LogUserAgent: true,
		LogRequestID: true,
		LogValuesFunc: func(c *echo.Context, v echomw.RequestLoggerValues) error {
			log := logger.From(c.Request().Context())
			if log == slog.Default() {
				log = base
			}
			attrs := []any{
				"method", v.Method,
				"path", v.URIPath,
				"route", v.RoutePath,
				"status", v.Status,
				"latency_ms", v.Latency.Milliseconds(),
				"remote_ip", v.RemoteIP,
				"user_agent", v.UserAgent,
			}
			// user_id is already on the request logger (see SetPrincipal).
			log.Info("http request", attrs...)
			return nil
		},
	})
}
