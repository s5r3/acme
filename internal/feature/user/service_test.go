package user

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"example.com/acme/internal/platform/auth"
	"example.com/acme/internal/shared/page"
)

type fakeRepo struct {
	byEmail map[string]*User
}

func (r *fakeRepo) FindByID(_ context.Context, id uuid.UUID) (*User, error) {
	for _, u := range r.byEmail {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, ErrUserNotFound
}

func (r *fakeRepo) FindByEmail(_ context.Context, email string) (*User, error) {
	if u, ok := r.byEmail[email]; ok {
		return u, nil
	}
	return nil, ErrUserNotFound
}

func (r *fakeRepo) Create(_ context.Context, u *User) error {
	if _, ok := r.byEmail[u.Email]; ok {
		return ErrEmailTaken
	}
	r.byEmail[u.Email] = u
	return nil
}

func (r *fakeRepo) List(context.Context, page.Query) ([]User, int64, error) { return nil, 0, nil }

func TestRegisterAndAuthenticate(t *testing.T) {
	ctx := context.Background()
	svc := newService(&fakeRepo{byEmail: map[string]*User{}})

	u, err := svc.Register(ctx, RegisterInput{Email: "  Ada@Example.com ", Name: "Ada", Password: "correct horse"})
	if err != nil {
		t.Fatal(err)
	}
	if u.Email != "ada@example.com" || u.Role != auth.RoleCustomer || u.PasswordHash == "correct horse" {
		t.Fatalf("unexpected user %+v", u)
	}

	if _, err := svc.Register(ctx, RegisterInput{Email: "ada@example.com", Name: "Ada 2", Password: "12345678"}); !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("duplicate email: err = %v", err)
	}
	if _, err := svc.Authenticate(ctx, "ADA@example.com", "correct horse"); err != nil {
		t.Fatalf("login: %v", err)
	}
	for _, tc := range []struct{ email, pw string }{{"ada@example.com", "wrong"}, {"nobody@example.com", "x"}} {
		if _, err := svc.Authenticate(ctx, tc.email, tc.pw); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("%s: err = %v, want ErrInvalidCredentials", tc.email, err)
		}
	}
}

func TestGetHidesOtherUsersWithoutPermission(t *testing.T) {
	repo := &fakeRepo{byEmail: map[string]*User{}}
	svc := newService(repo)
	a, _ := svc.Register(context.Background(), RegisterInput{Email: "a@example.com", Name: "A", Password: "12345678"})
	b, _ := svc.Register(context.Background(), RegisterInput{Email: "b@example.com", Name: "B", Password: "12345678"})

	roles := auth.NewRoles()
	(&Module{}).RegisterPermissions(roles)
	asA := auth.WithPrincipal(context.Background(), roles.NewPrincipal(a.ID, a.Email, a.Name, []string{auth.RoleCustomer}))
	asAdmin := auth.WithPrincipal(context.Background(), roles.NewPrincipal(uuid.New(), "admin@example.com", "Admin", []string{auth.RoleAdmin}))

	if _, err := svc.Get(asA, a.ID); err != nil {
		t.Fatalf("self: %v", err)
	}
	if _, err := svc.Get(asA, b.ID); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("other user as customer: err = %v, want not found", err)
	}
	if _, err := svc.Get(asAdmin, b.ID); err != nil {
		t.Fatalf("admin: %v", err)
	}
}
