package order

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"example.com/acme/internal/platform/auth"
	"example.com/acme/internal/platform/database"
	"example.com/acme/internal/platform/httpx"
	"example.com/acme/internal/shared/page"
)

// fakeRepo is a hand-written in-memory Repository.
type fakeRepo struct {
	orders map[uuid.UUID]*Order
}

func newFakeRepo() *fakeRepo { return &fakeRepo{orders: map[uuid.UUID]*Order{}} }

func (r *fakeRepo) Create(_ context.Context, o *Order) error {
	cp := *o
	r.orders[o.ID] = &cp
	return nil
}

func (r *fakeRepo) FindByID(_ context.Context, id uuid.UUID) (*Order, error) {
	o, ok := r.orders[id]
	if !ok {
		return nil, ErrOrderNotFound
	}
	cp := *o
	return &cp, nil
}

func (r *fakeRepo) FindByIDForUpdate(ctx context.Context, id uuid.UUID) (*Order, error) {
	return r.FindByID(ctx, id)
}

func (r *fakeRepo) List(_ context.Context, f ListFilter, q page.Query) ([]Order, int64, error) {
	var out []Order
	for _, o := range r.orders {
		if f.CustomerID != nil && o.CustomerID != *f.CustomerID {
			continue
		}
		if f.Status != "" && o.Status != f.Status {
			continue
		}
		out = append(out, *o)
	}
	return out, int64(len(out)), nil
}

func (r *fakeRepo) UpdateStatus(_ context.Context, id uuid.UUID, status string) error {
	r.orders[id].Status = status
	return nil
}

func (r *fakeRepo) AddItem(_ context.Context, item *Item, total int64) error {
	o := r.orders[item.OrderID]
	o.Items = append(o.Items, *item)
	o.TotalCents = total
	return nil
}

type fakeUsers map[uuid.UUID]bool

func (f fakeUsers) Exists(_ context.Context, id uuid.UUID) (bool, error) { return f[id], nil }

var (
	customerID = uuid.MustParse("00000000-0000-7000-8000-000000000001")
	otherID    = uuid.MustParse("00000000-0000-7000-8000-000000000002")
	adminID    = uuid.MustParse("00000000-0000-7000-8000-000000000003")
)

func ctxAs(userID uuid.UUID, role string) context.Context {
	roles := auth.NewRoles()
	(&Module{}).RegisterPermissions(roles)
	return auth.WithPrincipal(context.Background(), roles.NewPrincipal(userID, "x@example.com", "X", []string{role}))
}

func newTestService() (*Service, *fakeRepo) {
	repo := newFakeRepo()
	users := fakeUsers{customerID: true, otherID: true, adminID: true}
	return newService(repo, users, database.NoTx{}), repo
}

func oneItem() []ItemInput {
	return []ItemInput{{SKU: "A", Name: "Apple", Quantity: 3, UnitPriceCents: 120}}
}

func TestCreate(t *testing.T) {
	tests := []struct {
		name      string
		ctx       context.Context
		in        CreateOrderInput
		wantErr   error
		wantOwner uuid.UUID
		wantTotal int64
	}{
		{
			name:      "customer creates own order",
			ctx:       ctxAs(customerID, auth.RoleCustomer),
			in:        CreateOrderInput{Items: oneItem()},
			wantOwner: customerID,
			wantTotal: 360,
		},
		{
			name:    "customer cannot order for someone else",
			ctx:     ctxAs(customerID, auth.RoleCustomer),
			in:      CreateOrderInput{CustomerID: &otherID, Items: oneItem()},
			wantErr: ErrOtherCustomer,
		},
		{
			name:      "admin orders for a customer",
			ctx:       ctxAs(adminID, auth.RoleAdmin),
			in:        CreateOrderInput{CustomerID: &otherID, Items: oneItem()},
			wantOwner: otherID,
			wantTotal: 360,
		},
		{
			name:    "unknown customer",
			ctx:     ctxAs(adminID, auth.RoleAdmin),
			in:      CreateOrderInput{CustomerID: new(uuid.New()), Items: oneItem()},
			wantErr: ErrCustomerNotFound,
		},
		{
			name:    "no items",
			ctx:     ctxAs(customerID, auth.RoleCustomer),
			in:      CreateOrderInput{},
			wantErr: httpx.Validation(nil),
		},
		{
			name:    "unauthenticated",
			ctx:     context.Background(),
			in:      CreateOrderInput{Items: oneItem()},
			wantErr: httpx.ErrUnauthenticated,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := newTestService()
			o, err := svc.Create(tt.ctx, tt.in)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if o.CustomerID != tt.wantOwner || o.TotalCents != tt.wantTotal || o.Status != StatusOpen {
				t.Fatalf("got owner=%s total=%d status=%s", o.CustomerID, o.TotalCents, o.Status)
			}
		})
	}
}

func TestCancel(t *testing.T) {
	svc, _ := newTestService()
	ctx := ctxAs(customerID, auth.RoleCustomer)
	o, err := svc.Create(ctx, CreateOrderInput{Items: oneItem()})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := svc.Cancel(ctxAs(otherID, auth.RoleCustomer), o.ID); !errors.Is(err, ErrOrderNotFound) {
		t.Fatalf("other customer: err = %v, want not found", err)
	}
	got, err := svc.Cancel(ctx, o.ID)
	if err != nil || got.Status != StatusCancelled {
		t.Fatalf("cancel: %v, status %q", err, got.Status)
	}
	if _, err := svc.Cancel(ctx, o.ID); !errors.Is(err, ErrOrderClosed) {
		t.Fatalf("second cancel: err = %v, want ErrOrderClosed", err)
	}
	if _, err := svc.AddItem(ctx, o.ID, oneItem()[0]); !errors.Is(err, ErrOrderClosed) {
		t.Fatalf("add to cancelled: err = %v, want ErrOrderClosed", err)
	}
}

func TestAddItemUpdatesTotal(t *testing.T) {
	svc, repo := newTestService()
	ctx := ctxAs(customerID, auth.RoleCustomer)
	o, _ := svc.Create(ctx, CreateOrderInput{Items: oneItem()})

	if _, err := svc.AddItem(ctx, o.ID, ItemInput{SKU: "B", Name: "Bread", Quantity: 2, UnitPriceCents: 250}); err != nil {
		t.Fatal(err)
	}
	if got := repo.orders[o.ID].TotalCents; got != 360+500 {
		t.Fatalf("total = %d, want 860", got)
	}
}

func TestListScopesCustomersToOwnOrders(t *testing.T) {
	svc, _ := newTestService()
	_, _ = svc.Create(ctxAs(customerID, auth.RoleCustomer), CreateOrderInput{Items: oneItem()})
	_, _ = svc.Create(ctxAs(otherID, auth.RoleCustomer), CreateOrderInput{Items: oneItem()})

	res, err := svc.List(ctxAs(customerID, auth.RoleCustomer), ListFilter{}, page.New(1, 10))
	if err != nil || res.Total != 1 {
		t.Fatalf("customer list: total=%d err=%v, want 1", res.Total, err)
	}
	// A customer cannot widen the filter to another customer.
	res, _ = svc.List(ctxAs(customerID, auth.RoleCustomer), ListFilter{CustomerID: &otherID}, page.New(1, 10))
	if res.Total != 1 || res.Items[0].CustomerID != customerID {
		t.Fatalf("filter override leaked other customer's orders")
	}
	res, _ = svc.List(ctxAs(adminID, auth.RoleAdmin), ListFilter{}, page.New(1, 10))
	if res.Total != 2 {
		t.Fatalf("admin list: total=%d, want 2", res.Total)
	}
}
