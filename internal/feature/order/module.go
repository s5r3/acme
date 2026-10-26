// Package order owns orders and their items. Orders can be created through
// the JSON API and through the web pages; both go through Service.
package order

import (
	"gorm.io/gorm"

	"example.com/acme/internal/platform/database"
)

// Deps are everything the feature needs from the outside.
type Deps struct {
	DB    *gorm.DB
	Tx    database.TxManager
	Users UserLookup // satisfied by *user.Service, wired in app/modules.go
}

// Module wires the feature's repository, service and both front-ends.
type Module struct {
	svc *Service
	api *apiHandler
	web *webHandler
}

// New builds the module.
func New(d Deps) *Module {
	svc := newService(newGormRepository(d.DB), d.Users, d.Tx)
	return &Module{svc: svc, api: &apiHandler{svc: svc}, web: &webHandler{svc: svc}}
}

func (m *Module) Name() string { return "order" }
