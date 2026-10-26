// Package database opens the GORM connection and provides transaction
// propagation through context. Only repositories and this package use GORM.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"example.com/acme/internal/platform/config"
)

// Open connects to PostgreSQL, applies pool settings and pings the server.
func Open(ctx context.Context, cfg config.DatabaseConfig, log *slog.Logger) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{
		TranslateError: true, // unique violations -> gorm.ErrDuplicatedKey, etc.
		PrepareStmt:    true,
		Logger:         newGormLogger(log, cfg.SlowQueryThreshold),
		NowFunc:        func() time.Time { return time.Now().UTC() },
	})
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("sql db: %w", err)
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return db, nil
}

// Close closes the underlying connection pool.
func Close(db *gorm.DB) {
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}
}

// SQLDB returns the *sql.DB behind db (used by the migration runner).
func SQLDB(db *gorm.DB) (*sql.DB, error) {
	return db.DB()
}

// Timestamps is embedded in every entity.
type Timestamps struct {
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Paginate is a reusable GORM scope for offset pagination.
func Paginate(offset, limit int) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Offset(offset).Limit(limit)
	}
}
