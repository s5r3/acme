package database

import (
	"context"

	"gorm.io/gorm"
)

// TxManager runs a function inside a transaction. Services start
// transactions because one business operation can span several repositories
// or features; repositories join it through Conn.
type TxManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type txKey struct{}

type txManager struct {
	db *gorm.DB
}

// NewTxManager returns a TxManager backed by db.
func NewTxManager(db *gorm.DB) TxManager {
	return &txManager{db: db}
}

// WithinTx commits when fn returns nil and rolls back otherwise. A nested
// call joins the outer transaction instead of opening a new one.
func (m *txManager) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return fn(ctx)
	}
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, txKey{}, tx))
	})
}

// Conn returns the transaction stored in ctx, or base when there is none.
// Every repository method starts with Conn(ctx, r.db).
func Conn(ctx context.Context, base *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return base.WithContext(ctx)
}

// NoTx runs fn directly; use it in unit tests that fake repositories.
type NoTx struct{}

func (NoTx) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error { return fn(ctx) }
