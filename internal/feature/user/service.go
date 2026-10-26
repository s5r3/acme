package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"example.com/acme/internal/platform/auth"
	"example.com/acme/internal/platform/httpx"
	"example.com/acme/internal/platform/logger"
	"example.com/acme/internal/shared/id"
	"example.com/acme/internal/shared/page"
)

// repository is what the service needs from storage. It is declared here, by
// the consumer, and implemented by gormRepository in repository.go.
type repository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	Create(ctx context.Context, u *User) error
	List(ctx context.Context, q page.Query) ([]User, int64, error)
}

// Service holds the user business rules. Both the API and the web handlers
// call it; it knows nothing about HTTP.
type Service struct {
	repo repository
}

// newService is unexported on purpose: the feature's public API is New(Deps)
// and Module.Service(), so other packages always get a fully wired Service.
// Same-package tests call it directly with a fake repository.
func newService(repo repository) *Service {
	return &Service{repo: repo}
}

// RegisterInput is the service's own input type: the API DTO and the web
// form are both mapped to it.
type RegisterInput struct {
	Email    string
	Name     string
	Password string
	Role     string // empty means customer; only seeding sets admin
}

// Register creates an account. Self sign-up always creates a customer.
func (s *Service) Register(ctx context.Context, in RegisterInput) (*User, error) {
	email := normalizeEmail(in.Email)
	if email == "" || strings.TrimSpace(in.Name) == "" || len(in.Password) < 8 {
		return nil, httpx.Validation(map[string]string{"email": "email, name and an 8+ character password are required"})
	}
	role := in.Role
	if role == "" {
		role = auth.RoleCustomer
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	u := &User{ID: id.New(), Email: email, Name: strings.TrimSpace(in.Name), PasswordHash: hash, Role: role}
	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}
	logger.From(ctx).Info("user registered", "new_user_id", u.ID.String(), "role", role)
	return u, nil
}

// Authenticate checks credentials. The same error is returned for an unknown
// email and a wrong password so accounts cannot be enumerated.
func (s *Service) Authenticate(ctx context.Context, email, password string) (*User, error) {
	u, err := s.repo.FindByEmail(ctx, normalizeEmail(email))
	if errors.Is(err, ErrUserNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	if !auth.CheckPassword(u.PasswordHash, password) {
		return nil, ErrInvalidCredentials
	}
	return u, nil
}

// Get returns a user. Callers may read themselves; reading others needs
// PermUserRead. Without it the user is reported as not found.
func (s *Service) Get(ctx context.Context, userID uuid.UUID) (*User, error) {
	p, ok := auth.FromContext(ctx)
	if !ok {
		return nil, httpx.ErrUnauthenticated
	}
	if p.UserID != userID && !p.Has(PermUserRead) {
		return nil, ErrUserNotFound
	}
	return s.repo.FindByID(ctx, userID)
}

// List returns a page of users (admin only).
func (s *Service) List(ctx context.Context, q page.Query) (page.Result[User], error) {
	p, ok := auth.FromContext(ctx)
	if !ok || !p.Has(PermUserRead) {
		return page.Result[User]{}, httpx.ErrForbidden
	}
	items, total, err := s.repo.List(ctx, q)
	if err != nil {
		return page.Result[User]{}, err
	}
	return page.Result[User]{Items: items, Page: q.Page, PageSize: q.PageSize, Total: total}, nil
}

// Exists reports whether a user exists. Other features call it through an
// interface they declare (see order.UserLookup); it does no authorization.
func (s *Service) Exists(ctx context.Context, userID uuid.UUID) (bool, error) {
	_, err := s.repo.FindByID(ctx, userID)
	if errors.Is(err, ErrUserNotFound) {
		return false, nil
	}
	return err == nil, err
}

// EnsureUser creates the user if the email is not taken yet (used by seeding).
func (s *Service) EnsureUser(ctx context.Context, in RegisterInput) (*User, bool, error) {
	if u, err := s.repo.FindByEmail(ctx, normalizeEmail(in.Email)); err == nil {
		return u, false, nil
	} else if !errors.Is(err, ErrUserNotFound) {
		return nil, false, err
	}
	u, err := s.Register(ctx, in)
	return u, err == nil, err
}

func normalizeEmail(e string) string {
	return strings.ToLower(strings.TrimSpace(e))
}
