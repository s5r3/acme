//go:build integration

package order

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"example.com/acme/internal/platform/auth"
	"example.com/acme/internal/platform/database"
	"example.com/acme/internal/platform/database/dbtest"
	"example.com/acme/internal/shared/id"
	"example.com/acme/internal/shared/page"
)

// These tests run the real service + repository against PostgreSQL.

// insertUser creates a customer row directly, so this feature's tests don't
// import the user feature.
func insertUser(t *testing.T, db *gorm.DB) uuid.UUID {
	t.Helper()
	uid := id.New()
	err := db.Exec(`INSERT INTO users (id, email, name, password_hash, role) VALUES (?, ?, 'Test', 'x', 'customer')`,
		uid, uid.String()+"@example.com").Error
	if err != nil {
		t.Fatal(err)
	}
	return uid
}

type allUsersExist struct{}

func (allUsersExist) Exists(context.Context, uuid.UUID) (bool, error) { return true, nil }

func TestRepositoryRoundTrip(t *testing.T) {
	db := dbtest.Open(t)
	ctx := context.Background()

	uid := insertUser(t, db)

	repo := newGormRepository(db)
	o := &Order{ID: id.New(), CustomerID: uid, Status: StatusOpen, TotalCents: 300,
		Items: []Item{{ID: id.New(), SKU: "A", Name: "Apple", Quantity: 2, UnitPriceCents: 150}}}
	for i := range o.Items {
		o.Items[i].OrderID = o.ID
	}
	if err := repo.Create(ctx, o); err != nil {
		t.Fatal(err)
	}

	got, err := repo.FindByID(ctx, o.ID)
	if err != nil || len(got.Items) != 1 || got.TotalCents != 300 {
		t.Fatalf("find: %v %+v", err, got)
	}
	if _, err := repo.FindByID(ctx, id.New()); !errors.Is(err, ErrOrderNotFound) {
		t.Fatalf("missing order: err = %v", err)
	}

	list, total, err := repo.List(ctx, ListFilter{CustomerID: &uid, Status: StatusOpen}, page.New(1, 10))
	if err != nil || total != 1 || len(list) != 1 {
		t.Fatalf("list: %v total=%d", err, total)
	}
}

func TestServiceTransactionRollsBack(t *testing.T) {
	db := dbtest.Open(t)
	tx := database.NewTxManager(db)
	svc := newService(newGormRepository(db), allUsersExist{}, tx)

	uid := insertUser(t, db)
	roles := auth.NewRoles()
	(&Module{}).RegisterPermissions(roles)
	ctx := auth.WithPrincipal(context.Background(), roles.NewPrincipal(uid, "tx@example.com", "Tx", []string{auth.RoleCustomer}))

	o, err := svc.Create(ctx, CreateOrderInput{Items: []ItemInput{{SKU: "A", Name: "Apple", Quantity: 1, UnitPriceCents: 100}}})
	if err != nil {
		t.Fatal(err)
	}

	// A failure inside WithinTx must undo the earlier write.
	boom := errors.New("boom")
	err = tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := newGormRepository(db).UpdateStatus(ctx, o.ID, StatusCancelled); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
	got, _ := svc.Get(ctx, o.ID)
	if got.Status != StatusOpen {
		t.Fatalf("status = %q after rollback, want open", got.Status)
	}
}
