// Package user owns accounts: sign-up, login (API tokens and web sessions)
// and user lookup for other features.
package user

import (
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"gorm.io/gorm"
)

// TokenIssuer issues API bearer tokens (implemented by platform/auth.Tokens).
type TokenIssuer interface {
	Issue(userID uuid.UUID, email, name string, roles []string) (string, time.Time, error)
}

// SessionManager starts and ends web sessions (implemented by platform/web.Sessions).
type SessionManager interface {
	Start(c *echo.Context, userID uuid.UUID, email, name string, roles []string) error
	End(c *echo.Context) error
}

// Deps are everything the feature needs from the outside.
type Deps struct {
	DB       *gorm.DB
	Tokens   TokenIssuer
	Sessions SessionManager
}

// Module wires the feature's repository, service and both front-ends.
type Module struct {
	svc *Service
	api *apiHandler
	web *webHandler
}

// New builds the module.
func New(d Deps) *Module {
	svc := newService(newGormRepository(d.DB))
	return &Module{
		svc: svc,
		api: &apiHandler{svc: svc, tokens: d.Tokens},
		web: &webHandler{svc: svc, sessions: d.Sessions},
	}
}

func (m *Module) Name() string { return "user" }

// Service is exposed so other features can depend on it through their own
// small interfaces (see order.UserLookup).
func (m *Module) Service() *Service { return m.svc }
