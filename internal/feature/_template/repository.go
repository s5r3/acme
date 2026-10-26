package __name__

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

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

func (r *gormRepository) Create(ctx context.Context, x *__Name__) error {
	return r.db(ctx).Create(x).Error
}

func (r *gormRepository) FindByID(ctx context.Context, id uuid.UUID) (*__Name__, error) {
	var x __Name__
	if err := r.db(ctx).Where("id = ?", id).Take(&x).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, Err__Name__NotFound
		}
		return nil, err
	}
	return &x, nil
}

func (r *gormRepository) List(ctx context.Context, q page.Query) ([]__Name__, int64, error) {
	var total int64
	if err := r.db(ctx).Model(&__Name__{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []__Name__
	err := r.db(ctx).Order("created_at DESC, id DESC").
		Scopes(database.Paginate(q.Offset(), q.PageSize)).Find(&items).Error
	return items, total, err
}
