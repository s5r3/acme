package user

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

// db joins the transaction in ctx, if any.
func (r *gormRepository) db(ctx context.Context) *gorm.DB {
	return database.Conn(ctx, r.base)
}

func (r *gormRepository) FindByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var u User
	if err := r.db(ctx).Where("id = ?", id).Take(&u).Error; err != nil {
		return nil, translate(err)
	}
	return &u, nil
}

func (r *gormRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	if err := r.db(ctx).Where("email = ?", email).Take(&u).Error; err != nil {
		return nil, translate(err)
	}
	return &u, nil
}

func (r *gormRepository) Create(ctx context.Context, u *User) error {
	return translate(r.db(ctx).Create(u).Error)
}

func (r *gormRepository) List(ctx context.Context, q page.Query) ([]User, int64, error) {
	var total int64
	if err := r.db(ctx).Model(&User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var users []User
	err := r.db(ctx).
		Order("created_at DESC, id DESC").
		Scopes(database.Paginate(q.Offset(), q.PageSize)).
		Find(&users).Error
	return users, total, err
}

// translate maps GORM errors to feature errors at the repository boundary,
// so services never see GORM types.
func translate(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, gorm.ErrRecordNotFound):
		return ErrUserNotFound
	case errors.Is(err, gorm.ErrDuplicatedKey):
		return ErrEmailTaken.Wrap(err)
	default:
		return err
	}
}
