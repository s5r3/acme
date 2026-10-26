package middleware

import (
	"strings"

	"github.com/labstack/echo/v5"

	"example.com/acme/internal/platform/auth"
)

// BearerAuthenticator authenticates API requests from
// "Authorization: Bearer <token>".
func BearerAuthenticator(tokens *auth.Tokens) Authenticator {
	return AuthenticatorFunc(func(c *echo.Context) (*auth.Principal, error) {
		h := c.Request().Header.Get(echo.HeaderAuthorization)
		raw, ok := strings.CutPrefix(h, "Bearer ")
		if !ok || raw == "" {
			return nil, nil
		}
		return tokens.Verify(strings.TrimSpace(raw))
	})
}
