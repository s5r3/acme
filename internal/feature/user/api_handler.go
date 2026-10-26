package user

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"

	"example.com/acme/internal/platform/auth"
	"example.com/acme/internal/platform/httpx"
	"example.com/acme/internal/shared/page"
)

// apiService is the subset of Service the JSON handlers use; declared here so
// handler tests can pass a fake.
type apiService interface {
	Register(ctx context.Context, in RegisterInput) (*User, error)
	Authenticate(ctx context.Context, email, password string) (*User, error)
	Get(ctx context.Context, userID uuid.UUID) (*User, error)
	List(ctx context.Context, q page.Query) (page.Result[User], error)
}

type apiHandler struct {
	svc    apiService
	tokens TokenIssuer
}

// IssueToken exchanges email + password for a bearer token.
func (h *apiHandler) IssueToken(c *echo.Context) error {
	var req TokenRequest
	if err := httpx.BindAndValidate(c, &req); err != nil {
		return err
	}
	u, err := h.svc.Authenticate(c.Request().Context(), req.Email, req.Password)
	if err != nil {
		return err
	}
	tok, exp, err := h.tokens.Issue(u.ID, u.Email, u.Name, u.Roles())
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, TokenResponse{AccessToken: tok, TokenType: "Bearer", ExpiresAt: exp})
}

// Create is public self sign-up.
func (h *apiHandler) Create(c *echo.Context) error {
	var req CreateUserRequest
	if err := httpx.BindAndValidate(c, &req); err != nil {
		return err
	}
	u, err := h.svc.Register(c.Request().Context(), req.toInput())
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, toUserResponse(*u))
}

// Me returns the caller.
func (h *apiHandler) Me(c *echo.Context) error {
	p, _ := auth.FromContext(c.Request().Context())
	u, err := h.svc.Get(c.Request().Context(), p.UserID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, toUserResponse(*u))
}

func (h *apiHandler) Get(c *echo.Context) error {
	userID, err := echo.PathParam[uuid.UUID](c, "userID")
	if err != nil {
		return ErrUserNotFound
	}
	u, err := h.svc.Get(c.Request().Context(), userID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, toUserResponse(*u))
}

func (h *apiHandler) List(c *echo.Context) error {
	var q ListUsersQuery
	if err := httpx.BindAndValidate(c, &q); err != nil {
		return err
	}
	res, err := h.svc.List(c.Request().Context(), page.New(q.Page, q.PageSize))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, httpx.NewList(res, toUserResponse))
}
