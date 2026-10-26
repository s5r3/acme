package httpx

import (
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v5"

	"example.com/acme/internal/platform/logger"
)

// Problem is an RFC 9457 problem details body extended with code, requestId
// and per-field errors.
type Problem struct {
	Type      string            `json:"type"`
	Title     string            `json:"title"`
	Status    int               `json:"status"`
	Code      string            `json:"code"`
	RequestID string            `json:"requestId,omitempty"`
	Errors    map[string]string `json:"errors,omitempty"`
}

// ProblemFor converts any error into a Problem. Unknown errors become a
// generic 500 so internal details never reach clients.
func ProblemFor(err error) Problem {
	if appErr, ok := AsError(err); ok {
		return Problem{Type: "about:blank", Title: appErr.Message, Status: appErr.Kind.Status(), Code: appErr.Code, Errors: appErr.Fields}
	}
	// Echo's own errors (route not found, method not allowed, body too large...).
	// echo.StatusCode also matches the v5 sentinel errors, which are no longer
	// *echo.HTTPError values.
	if status := echo.StatusCode(err); status != 0 {
		return Problem{Type: "about:blank", Title: http.StatusText(status), Status: status, Code: codeForStatus(status)}
	}
	return Problem{Type: "about:blank", Title: "Internal server error", Status: http.StatusInternalServerError, Code: "internal"}
}

// ErrorHandler is the API binary's echo.HTTPErrorHandler: one place that turns
// errors into problem+json responses and logs them once.
func ErrorHandler(base *slog.Logger) echo.HTTPErrorHandler {
	return func(c *echo.Context, err error) {
		if resp, _ := echo.UnwrapResponse(c.Response()); resp != nil && resp.Committed {
			return // already written (e.g. by the access logger's error handling)
		}

		p := ProblemFor(err)
		p.RequestID = c.Response().Header().Get(echo.HeaderXRequestID)
		LogError(c, base, p.Status, err)

		c.Response().Header().Set(echo.HeaderContentType, "application/problem+json")
		if c.Request().Method == http.MethodHead {
			_ = c.NoContent(p.Status)
			return
		}
		_ = c.JSON(p.Status, p)
	}
}

// LogError logs server errors at Error level and client errors at Debug.
// Shared by the API and web error handlers so both log the same way.
func LogError(c *echo.Context, base *slog.Logger, status int, err error) {
	log := logger.From(c.Request().Context())
	if log == slog.Default() {
		log = base
	}
	attrs := []any{"status", status, "method", c.Request().Method, "path", c.Request().URL.Path, "err", err}
	if status >= http.StatusInternalServerError {
		log.Error("request failed", attrs...)
		return
	}
	log.Debug("request rejected", attrs...)
}

func codeForStatus(status int) string {
	switch status {
	case http.StatusNotFound:
		return "route.not_found"
	case http.StatusMethodNotAllowed:
		return "route.method_not_allowed"
	case http.StatusRequestEntityTooLarge:
		return "request.too_large"
	case http.StatusUnauthorized:
		return "auth.required"
	case http.StatusForbidden:
		return "request.forbidden" // e.g. Echo's CSRF check
	case http.StatusServiceUnavailable:
		return "service.unavailable"
	default:
		if status >= 500 {
			return "internal"
		}
		return "request.invalid"
	}
}
