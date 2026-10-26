package database

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/pressly/goose/v3"
	"gorm.io/gorm"
)

// Migrator runs the versioned SQL migrations with goose. It is used by
// cmd/migrate and by integration tests; the API and web binaries never
// migrate on startup.
type Migrator struct {
	p *goose.Provider
}

// NewMigrator builds a goose provider over the embedded migrations FS.
func NewMigrator(db *gorm.DB, migrations fs.FS) (*Migrator, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	p, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations)
	if err != nil {
		return nil, fmt.Errorf("goose provider: %w", err)
	}
	return &Migrator{p: p}, nil
}

// Up applies all pending migrations.
func (m *Migrator) Up(ctx context.Context, log *slog.Logger) error {
	results, err := m.p.Up(ctx)
	for _, r := range results {
		log.Info("migration applied", "migration", r.Source.Version, "file", r.Source.Path, "duration_ms", r.Duration.Milliseconds())
	}
	return err
}

// Down rolls back the most recent migration.
func (m *Migrator) Down(ctx context.Context, log *slog.Logger) error {
	r, err := m.p.Down(ctx)
	if r != nil {
		log.Info("migration rolled back", "migration", r.Source.Version, "file", r.Source.Path)
	}
	return err
}

// Status prints each migration and whether it is applied.
func (m *Migrator) Status(ctx context.Context) ([]string, error) {
	statuses, err := m.p.Status(ctx)
	if err != nil {
		return nil, err
	}
	lines := make([]string, 0, len(statuses))
	for _, s := range statuses {
		applied := "pending"
		if s.State == goose.StateApplied {
			applied = "applied " + s.AppliedAt.Format("2006-01-02 15:04:05")
		}
		lines = append(lines, fmt.Sprintf("%-14d %-45s %s", s.Source.Version, s.Source.Path, applied))
	}
	return lines, nil
}
