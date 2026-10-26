package __name__

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"example.com/acme/internal/platform/database"
	"example.com/acme/internal/platform/httpx"
	"example.com/acme/internal/shared/id"
	"example.com/acme/internal/shared/page"
)

// repository is declared by its consumer (this service).
type repository interface {
	Create(ctx context.Context, x *__Name__) error
	FindByID(ctx context.Context, id uuid.UUID) (*__Name__, error)
	List(ctx context.Context, q page.Query) ([]__Name__, int64, error)
}

// Service holds the business rules; no Echo, no GORM.
type Service struct {
	repo repository
	tx   database.TxManager
}

// newService is unexported on purpose: the feature's public API is New(Deps)
// and Module.Service(), so other packages always get a fully wired Service.
// Same-package tests call it directly with a fake repository.
func newService(repo repository, tx database.TxManager) *Service {
	return &Service{repo: repo, tx: tx}
}

// Create__Name__Input is the service's own input type.
type Create__Name__Input struct {
	Name string
}

func (s *Service) Create(ctx context.Context, in Create__Name__Input) (*__Name__, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, httpx.Validation(map[string]string{"name": "is required"})
	}
	x := &__Name__{ID: id.New(), Name: name}
	if err := s.repo.Create(ctx, x); err != nil {
		return nil, err
	}
	return x, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*__Name__, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) List(ctx context.Context, q page.Query) (page.Result[__Name__], error) {
	items, total, err := s.repo.List(ctx, q)
	if err != nil {
		return page.Result[__Name__]{}, err
	}
	return page.Result[__Name__]{Items: items, Page: q.Page, PageSize: q.PageSize, Total: total}, nil
}
