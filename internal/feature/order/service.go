package order

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"example.com/acme/internal/platform/auth"
	"example.com/acme/internal/platform/database"
	"example.com/acme/internal/platform/httpx"
	"example.com/acme/internal/platform/logger"
	"example.com/acme/internal/shared/id"
	"example.com/acme/internal/shared/page"
)

const maxItems = 50

// repository is what the service needs from storage.
type repository interface {
	Create(ctx context.Context, o *Order) error
	FindByID(ctx context.Context, id uuid.UUID) (*Order, error)
	FindByIDForUpdate(ctx context.Context, id uuid.UUID) (*Order, error)
	List(ctx context.Context, f ListFilter, q page.Query) ([]Order, int64, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	AddItem(ctx context.Context, item *Item, newTotalCents int64) error
}

// UserLookup is the part of the user feature this feature needs. It is
// declared here (by the consumer) and satisfied by *user.Service.
type UserLookup interface {
	Exists(ctx context.Context, userID uuid.UUID) (bool, error)
}

// Service holds every order business rule. The API and web handlers both
// call it, so an order behaves the same whichever front-end created it.
type Service struct {
	repo  repository
	users UserLookup
	tx    database.TxManager
}

// newService is unexported on purpose: the feature's public API is New(Deps)
// and Module.Service(), so other packages always get a fully wired Service.
// Same-package tests call it directly with a fake repository.
func newService(repo repository, users UserLookup, tx database.TxManager) *Service {
	return &Service{repo: repo, users: users, tx: tx}
}

// CreateOrderInput is the service's own input type. The API's
// CreateOrderRequest and the web's CreateOrderForm are both mapped to it.
type CreateOrderInput struct {
	CustomerID *uuid.UUID // nil = the caller; another customer needs PermOrderAdmin
	Note       string
	Items      []ItemInput
}

type ItemInput struct {
	SKU            string
	Name           string
	Quantity       int
	UnitPriceCents int64
}

// ListFilter narrows a list. Non-admins are always limited to their own orders.
type ListFilter struct {
	Status     string
	CustomerID *uuid.UUID
}

// Create places a new open order.
func (s *Service) Create(ctx context.Context, in CreateOrderInput) (*Order, error) {
	p, err := principal(ctx)
	if err != nil {
		return nil, err
	}
	customerID := p.UserID
	if in.CustomerID != nil && *in.CustomerID != p.UserID {
		if !p.Has(PermOrderAdmin) {
			return nil, ErrOtherCustomer
		}
		customerID = *in.CustomerID
	}
	if err := validateItems(in.Items); err != nil {
		return nil, err
	}

	o := &Order{ID: id.New(), CustomerID: customerID, Status: StatusOpen, Note: strings.TrimSpace(in.Note)}
	for _, it := range in.Items {
		item := toItem(o.ID, it)
		o.Items = append(o.Items, item)
		o.TotalCents += item.LineTotalCents()
	}

	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		ok, err := s.users.Exists(ctx, customerID) // other feature, same transaction
		if err != nil {
			return err
		}
		if !ok {
			return ErrCustomerNotFound
		}
		return s.repo.Create(ctx, o)
	})
	if err != nil {
		return nil, err
	}
	logger.From(ctx).Info("order created", "order_id", o.ID.String(), "customer_id", customerID.String(), "total_cents", o.TotalCents)
	return o, nil
}

// Get returns an order the caller may see. Someone else's order is reported
// as not found rather than forbidden, so IDs cannot be probed.
func (s *Service) Get(ctx context.Context, orderID uuid.UUID) (*Order, error) {
	p, err := principal(ctx)
	if err != nil {
		return nil, err
	}
	o, err := s.repo.FindByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if !canAccess(p, o) {
		return nil, ErrOrderNotFound
	}
	return o, nil
}

// List returns a page of orders visible to the caller.
func (s *Service) List(ctx context.Context, f ListFilter, q page.Query) (page.Result[Order], error) {
	p, err := principal(ctx)
	if err != nil {
		return page.Result[Order]{}, err
	}
	if !p.Has(PermOrderAdmin) {
		f.CustomerID = &p.UserID
	}
	items, total, err := s.repo.List(ctx, f, q)
	if err != nil {
		return page.Result[Order]{}, err
	}
	return page.Result[Order]{Items: items, Page: q.Page, PageSize: q.PageSize, Total: total}, nil
}

// Cancel cancels an open order.
func (s *Service) Cancel(ctx context.Context, orderID uuid.UUID) (*Order, error) {
	p, err := principal(ctx)
	if err != nil {
		return nil, err
	}
	var o *Order
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		o, err = s.repo.FindByIDForUpdate(ctx, orderID)
		if err != nil {
			return err
		}
		if !canAccess(p, o) {
			return ErrOrderNotFound
		}
		if !o.IsOpen() {
			return ErrOrderClosed
		}
		o.Status = StatusCancelled
		return s.repo.UpdateStatus(ctx, o.ID, StatusCancelled)
	})
	if err != nil {
		return nil, err
	}
	logger.From(ctx).Info("order cancelled", "order_id", o.ID.String())
	return o, nil
}

// AddItem adds a line to an open order and updates its total atomically.
func (s *Service) AddItem(ctx context.Context, orderID uuid.UUID, in ItemInput) (*Item, error) {
	p, err := principal(ctx)
	if err != nil {
		return nil, err
	}
	if err := validateItems([]ItemInput{in}); err != nil {
		return nil, err
	}
	var item Item
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		o, err := s.repo.FindByIDForUpdate(ctx, orderID)
		if err != nil {
			return err
		}
		if !canAccess(p, o) {
			return ErrOrderNotFound
		}
		if !o.IsOpen() {
			return ErrOrderClosed
		}
		if len(o.Items) >= maxItems {
			return httpx.Validation(map[string]string{"items": fmt.Sprintf("an order can have at most %d items", maxItems)})
		}
		item = toItem(o.ID, in)
		return s.repo.AddItem(ctx, &item, o.TotalCents+item.LineTotalCents())
	})
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// CanAdminister reports whether the caller may act on any customer's orders.
func CanAdminister(ctx context.Context) bool {
	p, ok := auth.FromContext(ctx)
	return ok && p.Has(PermOrderAdmin)
}

func principal(ctx context.Context) (*auth.Principal, error) {
	p, ok := auth.FromContext(ctx)
	if !ok {
		return nil, httpx.ErrUnauthenticated
	}
	return p, nil
}

func canAccess(p *auth.Principal, o *Order) bool {
	return o.CustomerID == p.UserID || p.Has(PermOrderAdmin)
}

// validateItems holds the business rules for lines. Handlers validate the
// wire format; this protects the rules whatever the caller.
func validateItems(items []ItemInput) error {
	if len(items) == 0 {
		return httpx.Validation(map[string]string{"items": "add at least one item"})
	}
	if len(items) > maxItems {
		return httpx.Validation(map[string]string{"items": fmt.Sprintf("an order can have at most %d items", maxItems)})
	}
	fields := map[string]string{}
	for i, it := range items {
		key := fmt.Sprintf("items[%d]", i)
		switch {
		case strings.TrimSpace(it.SKU) == "":
			fields[key+".sku"] = "is required"
		case strings.TrimSpace(it.Name) == "":
			fields[key+".name"] = "is required"
		case it.Quantity <= 0:
			fields[key+".quantity"] = "must be greater than 0"
		case it.UnitPriceCents < 0:
			fields[key+".unitPriceCents"] = "must not be negative"
		}
	}
	if len(fields) > 0 {
		return httpx.Validation(fields)
	}
	return nil
}

func toItem(orderID uuid.UUID, in ItemInput) Item {
	return Item{
		ID:             id.New(),
		OrderID:        orderID,
		SKU:            strings.TrimSpace(in.SKU),
		Name:           strings.TrimSpace(in.Name),
		Quantity:       in.Quantity,
		UnitPriceCents: in.UnitPriceCents,
	}
}
