package middleware

import (
	"github.com/labstack/echo/v5"

	"example.com/acme/internal/platform/auth"
	"example.com/acme/internal/platform/httpx"
	"example.com/acme/internal/platform/logger"
)

// Authenticator turns a request into a Principal. The API binary uses a
// bearer-token authenticator, the web binary a session authenticator.
// It returns (nil, nil) when the request carries no credentials.
type Authenticator interface {
	Authenticate(c *echo.Context) (*auth.Principal, error)
}

// AuthenticatorFunc adapts a function to Authenticator.
type AuthenticatorFunc func(c *echo.Context) (*auth.Principal, error)

func (f AuthenticatorFunc) Authenticate(c *echo.Context) (*auth.Principal, error) { return f(c) }

// Guards are attached to route groups and routes in each feature's
// api_routes.go / web_routes.go.
type Guards struct {
	// Authenticated rejects requests without a valid identity and stores the
	// Principal in the request context.
	Authenticated echo.MiddlewareFunc
	authn         Authenticator
	onMissing     func(c *echo.Context) error
}

// NewGuards builds guards around authn. onMissing decides what an
// unauthenticated request gets: a 401 for the API, a redirect for the web.
func NewGuards(authn Authenticator, onMissing func(c *echo.Context) error) Guards {
	g := Guards{authn: authn, onMissing: onMissing}
	g.Authenticated = g.authenticated
	return g
}

func (g Guards) authenticated(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		if _, ok := auth.FromContext(c.Request().Context()); ok {
			return next(c) // already authenticated by an outer group
		}
		p, err := g.authn.Authenticate(c)
		if err != nil || p == nil {
			return g.onMissing(c)
		}
		SetPrincipal(c, p)
		return next(c)
	}
}

// Require rejects the request with 403 unless the caller holds every perm.
// It assumes Authenticated ran earlier in the chain.
func (g Guards) Require(perms ...auth.Permission) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			p, ok := auth.FromContext(c.Request().Context())
			if !ok {
				return g.onMissing(c)
			}
			if !p.Has(perms...) {
				return httpx.ErrForbidden
			}
			return next(c)
		}
	}
}

// SetPrincipal stores p in the request context and tags the request logger
// with user_id.
func SetPrincipal(c *echo.Context, p *auth.Principal) {
	ctx := c.Request().Context()
	ctx = auth.WithPrincipal(ctx, p)
	ctx = logger.WithContext(ctx, logger.From(ctx).With("user_id", p.UserID.String()))
	c.SetRequest(c.Request().WithContext(ctx))
}

// Unauthorized is the API's onMissing: a 401 problem response.
func Unauthorized(*echo.Context) error {
	return httpx.ErrUnauthenticated
}
