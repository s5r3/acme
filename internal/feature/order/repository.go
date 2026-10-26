package order

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"example.com/acme/internal/platform/database"
	"example.com/acme/internal/shared/page"
)

type gormRepository struct {
	base *gorm.DB
}

// gormRepository must satisfy the interface the service declares.
var _ repository = (*gormRepository)(nil)

// newGormRepository returns the GORM implementation of repository.
func newGormRepository(db *gorm.DB) *gormRepository {
	return &gormRepository{base: db}
}

func (r *gormRepository) db(ctx context.Context) *gorm.DB {
	return database.Conn(ctx, r.base)
}

// Create inserts the order and its items (GORM saves the association).
func (r *gormRepository) Create(ctx context.Context, o *Order) error {
	return r.db(ctx).Create(o).Error
}

func (r *gormRepository) FindByID(ctx context.Context, id uuid.UUID) (*Order, error) {
	var o Order
	err := r.db(ctx).
		Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order("created_at, id") }).
		Where("id = ?", id).
		Take(&o).Error
	if err != nil {
		return nil, translate(err)
	}
	return &o, nil
}

// FindByIDForUpdate locks the order row until the transaction ends, so two
// concurrent changes to one order are serialized.
func (r *gormRepository) FindByIDForUpdate(ctx context.Context, id uuid.UUID) (*Order, error) {
	var o Order
	err := r.db(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", id).
		Take(&o).Error
	if err != nil {
		return nil, translate(err)
	}
	if err := r.db(ctx).Where("order_id = ?", id).Order("created_at, id").Find(&o.Items).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *gormRepository) List(ctx context.Context, f ListFilter, q page.Query) ([]Order, int64, error) {
	base := r.db(ctx).Model(&Order{}).Scopes(filter(f))

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var orders []Order
	err := r.db(ctx).
		Scopes(filter(f), database.Paginate(q.Offset(), q.PageSize)).
		Order("created_at DESC, id DESC").
		Find(&orders).Error
	return orders, total, err
}

func (r *gormRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	res := r.db(ctx).Model(&Order{}).Where("id = ?", id).Update("status", status)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrOrderNotFound
	}
	return nil
}

// AddItem inserts the line and stores the new total. Call it inside a
// transaction (the service does).
func (r *gormRepository) AddItem(ctx context.Context, item *Item, newTotalCents int64) error {
	if err := r.db(ctx).Create(item).Error; err != nil {
		return err
	}
	return r.db(ctx).Model(&Order{}).Where("id = ?", item.OrderID).Update("total_cents", newTotalCents).Error
}

// filter is a reusable scope for list queries.
func filter(f ListFilter) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if f.Status != "" {
			db = db.Where("status = ?", f.Status)
		}
		if f.CustomerID != nil {
			db = db.Where("customer_id = ?", *f.CustomerID)
		}
		return db
	}
}

func translate(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrOrderNotFound
	}
	return err
}
