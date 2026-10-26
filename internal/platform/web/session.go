package web

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"gorm.io/gorm"

	"example.com/acme/internal/platform/auth"
	"example.com/acme/internal/platform/config"
	"example.com/acme/internal/platform/database"
	"example.com/acme/internal/platform/middleware"
	"example.com/acme/internal/shared/clock"
)

// SessionCookie is the name of the web session cookie. It holds only an
// opaque random ID; the session data lives in the sessions table.
const SessionCookie = "acme_session"

// Session is a logged-in browser session.
type Session struct {
	ID        string    `gorm:"primaryKey"`
	UserID    uuid.UUID `gorm:"type:uuid"`
	Email     string
	Name      string
	Roles     []string `gorm:"serializer:json;type:jsonb"`
	ExpiresAt time.Time
	CreatedAt time.Time
}

func (Session) TableName() string { return "sessions" }

// Sessions manages server-side sessions and their cookie.
type Sessions struct {
	db     *gorm.DB
	roles  *auth.Roles
	ttl    time.Duration
	secure bool
	clock  clock.Clock
}

// NewSessions builds the session manager used by the web binary.
func NewSessions(db *gorm.DB, roles *auth.Roles, cfg config.WebConfig, clk clock.Clock) *Sessions {
	return &Sessions{db: db, roles: roles, ttl: cfg.SessionTTL, secure: cfg.CookieSecure, clock: clk}
}

// Start creates a session for the user and sets the cookie. Call it after a
// successful login or sign-up.
func (s *Sessions) Start(c *echo.Context, userID uuid.UUID, email, name string, roles []string) error {
	sid, err := randomID()
	if err != nil {
		return err
	}
	now := s.clock.Now()
	sess := Session{ID: sid, UserID: userID, Email: email, Name: name, Roles: roles, ExpiresAt: now.Add(s.ttl), CreatedAt: now}
	if err := database.Conn(c.Request().Context(), s.db).Create(&sess).Error; err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	c.SetCookie(s.cookie(sid, int(s.ttl.Seconds())))
	middleware.SetPrincipal(c, s.roles.NewPrincipal(userID, email, name, roles))
	return nil
}

// End deletes the current session and clears the cookie.
func (s *Sessions) End(c *echo.Context) error {
	if ck, err := c.Cookie(SessionCookie); err == nil && ck.Value != "" {
		if err := database.Conn(c.Request().Context(), s.db).Delete(&Session{}, "id = ?", ck.Value).Error; err != nil {
			return fmt.Errorf("delete session: %w", err)
		}
	}
	c.SetCookie(s.cookie("", -1))
	return nil
}

// Load is middleware that attaches the Principal of a valid session to the
// request context. Pages stay public unless their route adds the
// Authenticated guard.
func (s *Sessions) Load(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		ck, err := c.Cookie(SessionCookie)
		if err != nil || ck.Value == "" {
			return next(c)
		}
		var sess Session
		err = database.Conn(c.Request().Context(), s.db).
			Where("id = ? AND expires_at > ?", ck.Value, s.clock.Now()).
			Take(&sess).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			c.SetCookie(s.cookie("", -1)) // stale cookie
		case err != nil:
			return fmt.Errorf("load session: %w", err)
		default:
			middleware.SetPrincipal(c, s.roles.NewPrincipal(sess.UserID, sess.Email, sess.Name, sess.Roles))
		}
		return next(c)
	}
}

// DeleteExpired removes expired sessions; run it periodically.
func (s *Sessions) DeleteExpired(ctx context.Context) (int64, error) {
	res := database.Conn(ctx, s.db).Where("expires_at <= ?", s.clock.Now()).Delete(&Session{})
	return res.RowsAffected, res.Error
}

func (s *Sessions) cookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     SessionCookie,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   s.secure,
		SameSite: http.SameSiteLaxMode,
	}
}

func randomID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
