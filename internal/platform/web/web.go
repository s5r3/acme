// Package web holds the HTML front-end's plumbing: sessions, CSRF, flash
// messages, security headers, rendering and the HTML error handler.
package web

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v5"
	echomw "github.com/labstack/echo/v5/middleware"

	"example.com/acme/internal/platform/auth"
	"example.com/acme/internal/platform/httpx"
	"example.com/acme/internal/platform/web/layout"
)

const (
	csrfCookie     = "acme_csrf"
	csrfContextKey = "csrf"
	flashCookie    = "acme_flash"
)

// Render renders a templ component with the given status. The page is
// rendered into a buffer first so a template error never sends half a page.
func Render(c *echo.Context, status int, comp templ.Component) error {
	var buf bytes.Buffer
	if err := comp.Render(c.Request().Context(), &buf); err != nil {
		return err
	}
	return c.Blob(status, echo.MIMETextHTMLCharsetUTF8, buf.Bytes())
}

// RedirectSeeOther is the post/redirect/get response after a successful form.
func RedirectSeeOther(c *echo.Context, to string) error {
	return c.Redirect(http.StatusSeeOther, to)
}

// RedirectToLogin is the web binary's onMissing guard handler.
func RedirectToLogin(c *echo.Context) error {
	next := c.Request().URL.RequestURI()
	return c.Redirect(http.StatusSeeOther, "/login?next="+url.QueryEscape(next))
}

// SafeNext returns next if it is a local path, else fallback. It prevents
// open redirects through ?next=.
func SafeNext(next, fallback string) string {
	if strings.HasPrefix(next, "/") && !strings.HasPrefix(next, "//") && !strings.HasPrefix(next, "/\\") {
		return next
	}
	return fallback
}

// ErrCSRF is returned for a missing or wrong CSRF token.
var ErrCSRF = httpx.New(httpx.KindForbidden, "csrf.invalid", "Your form has expired. Reload the page and try again.")

// CSRF uses Echo's double-submit-cookie middleware; forms send the token in
// the "_csrf" field (layout.CSRFField).
func CSRF(secure bool) echo.MiddlewareFunc {
	return echomw.CSRFWithConfig(echomw.CSRFConfig{
		TokenLookup:    "form:_csrf,header:X-CSRF-Token",
		ContextKey:     csrfContextKey,
		CookieName:     csrfCookie,
		CookiePath:     "/",
		CookieHTTPOnly: true,
		CookieSecure:   secure,
		CookieSameSite: http.SameSiteLaxMode,
		ErrorHandler:   func(_ *echo.Context, err error) error { return ErrCSRF.Wrap(err) },
	})
}

// SetFlash stores a one-time message shown on the next page.
func SetFlash(c *echo.Context, kind, message string) {
	val := base64.RawURLEncoding.EncodeToString([]byte(kind + "|" + message))
	c.SetCookie(&http.Cookie{Name: flashCookie, Value: val, Path: "/", MaxAge: 60, HttpOnly: true, SameSite: http.SameSiteLaxMode})
}

// PageData is middleware that builds layout.Page (user, CSRF token, flash)
// for templates. It must run after Sessions.Load and CSRF.
func PageData(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		p := layout.Page{Path: c.Request().URL.Path}
		if tok, ok := c.Get(csrfContextKey).(string); ok {
			p.CSRFToken = tok
		}
		if ck, err := c.Cookie(flashCookie); err == nil && ck.Value != "" {
			if raw, err := base64.RawURLEncoding.DecodeString(ck.Value); err == nil {
				kind, msg, _ := strings.Cut(string(raw), "|")
				p.Flash = &layout.Flash{Kind: kind, Message: msg}
			}
			c.SetCookie(&http.Cookie{Name: flashCookie, Value: "", Path: "/", MaxAge: -1})
		}
		ctx := c.Request().Context()
		if pr, ok := auth.FromContext(ctx); ok {
			p.User = pr
		}
		c.SetRequest(c.Request().WithContext(layout.WithPage(ctx, p)))
		return next(c)
	}
}

// RefreshUser updates layout.Page after login/logout changed the Principal
// within the same request (e.g. when re-rendering instead of redirecting).
func RefreshUser(c *echo.Context) {
	ctx := c.Request().Context()
	p := layout.PageFrom(ctx)
	p.User, _ = auth.FromContext(ctx)
	c.SetRequest(c.Request().WithContext(layout.WithPage(ctx, p)))
}

// SecureHeaders sets browser security headers for HTML responses.
func SecureHeaders(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		h := c.Response().Header()
		h.Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self'; form-action 'self'; frame-ancestors 'none'")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		return next(c)
	}
}
