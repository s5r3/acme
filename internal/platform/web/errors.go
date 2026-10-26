package web

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v5"

	"example.com/acme/internal/platform/httpx"
	"example.com/acme/internal/platform/web/layout"
)

// ErrorHandler is the web binary's echo.HTTPErrorHandler. It maps the same
// error kinds as the API to the same status codes, but renders HTML.
func ErrorHandler(base *slog.Logger) echo.HTTPErrorHandler {
	return func(c *echo.Context, err error) {
		if resp, _ := echo.UnwrapResponse(c.Response()); resp != nil && resp.Committed {
			return
		}
		p := httpx.ProblemFor(err)
		httpx.LogError(c, base, p.Status, err)

		if p.Status == http.StatusUnauthorized {
			_ = RedirectToLogin(c)
			return
		}

		// The request deadline context may already be cancelled here (the
		// error handler runs after the handler chain returned).
		c.SetRequest(c.Request().WithContext(context.WithoutCancel(c.Request().Context())))
		RefreshUser(c) // the error may have stopped the chain before PageData ran

		title, message := pageText(p)
		if renderErr := Render(c, p.Status, layout.ErrorPage(p.Status, title, message)); renderErr != nil {
			base.Error("render error page", "err", renderErr)
			_ = c.String(http.StatusInternalServerError, "Internal server error")
		}
	}
}

func pageText(p httpx.Problem) (string, string) {
	switch p.Status {
	case http.StatusNotFound:
		return "Page not found", "The page you asked for does not exist or you cannot see it."
	case http.StatusForbidden:
		if p.Code == ErrCSRF.Code {
			return "Form expired", "Your form has expired. Go back, reload the page and try again."
		}
		return "Not allowed", "You do not have permission to do this."
	case http.StatusInternalServerError:
		return "Something went wrong", "An unexpected error happened. Please try again."
	default:
		return http.StatusText(p.Status), p.Title
	}
}
