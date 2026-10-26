// Package __name__ is generated from internal/feature/_template by
// `make feature name=__name__`. The Go tool ignores directories starting with
// "_", so this template never compiles on its own.
package __name__

import (
	"gorm.io/gorm"

	"example.com/acme/internal/platform/database"
)

// Deps are everything the feature needs from the outside.
type Deps struct {
	DB *gorm.DB
	Tx database.TxManager
}

// Module wires the feature's repository, service and front-ends.
type Module struct {
	svc *Service
	api *apiHandler
}

func New(d Deps) *Module {
	svc := newService(newGormRepository(d.DB), d.Tx)
	return &Module{svc: svc, api: &apiHandler{svc: svc}}
}

func (m *Module) Name() string { return "__name__" }

// Service lets other features depend on this one through their own interface.
func (m *Module) Service() *Service { return m.svc }
