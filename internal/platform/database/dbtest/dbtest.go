// Package dbtest gives integration tests a migrated, empty PostgreSQL
// database. Tests using it carry the "integration" build tag and are skipped
// when ACME_TEST_DATABASE_DSN is unset.
package dbtest

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"gorm.io/gorm"

	"example.com/acme/internal/platform/config"
	"example.com/acme/internal/platform/database"
	"example.com/acme/migrations"
)

// EnvDSN names the variable holding the test database DSN. Use a dedicated
// database: every table is truncated between tests.
const EnvDSN = "ACME_TEST_DATABASE_DSN"

var migrateOnce sync.Once

// DSN returns the test DSN or skips the test.
func DSN(t testing.TB) string {
	t.Helper()
	dsn := os.Getenv(EnvDSN)
	if dsn == "" {
		t.Skipf("%s not set; skipping integration test", EnvDSN)
	}
	return dsn
}

// Open connects, runs the real migrations once per process and truncates all
// tables so each test starts empty.
func Open(t testing.TB) *gorm.DB {
	t.Helper()
	ctx := context.Background()
	quiet := slog.New(slog.NewTextHandler(io.Discard, nil))

	db, err := database.Open(ctx, config.DatabaseConfig{
		DSN: DSN(t), MaxOpenConns: 5, MaxIdleConns: 2, ConnMaxLifetime: time.Minute,
	}, quiet)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() { database.Close(db) })

	var migErr error
	migrateOnce.Do(func() {
		m, err := database.NewMigrator(db, migrations.FS)
		if err == nil {
			err = m.Up(ctx, quiet)
		}
		migErr = err
	})
	if migErr != nil {
		t.Fatalf("migrate test database: %v", migErr)
	}

	Truncate(t, db)
	return db
}

// Truncate empties every application table.
func Truncate(t testing.TB, db *gorm.DB) {
	t.Helper()
	if err := db.Exec("TRUNCATE order_items, orders, sessions, users RESTART IDENTITY CASCADE").Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}
}
