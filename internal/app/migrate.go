package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"example.com/acme/internal/feature/user"
	"example.com/acme/internal/platform/auth"
	"example.com/acme/internal/platform/database"
	"example.com/acme/migrations"
)

// RunMigrate runs a migration command: up, down, status or seed.
// Migrations run as their own step before a deploy, never on API start-up.
func RunMigrate(ctx context.Context, cfgPath, command string) error {
	core, err := LoadCore(ctx, cfgPath, "migrate")
	if err != nil {
		return err
	}
	defer core.Close()

	m, err := database.NewMigrator(core.DB, migrations.FS)
	if err != nil {
		return err
	}
	switch command {
	case "up":
		return m.Up(ctx, core.Log)
	case "down":
		return m.Down(ctx, core.Log)
	case "status":
		lines, err := m.Status(ctx)
		for _, l := range lines {
			fmt.Println(l)
		}
		return err
	case "seed":
		return seed(ctx, core)
	default:
		return fmt.Errorf("unknown command %q (want up, down, status, seed, create)", command)
	}
}

// seed creates demo accounts for local development. Never run it in production.
func seed(ctx context.Context, c *Core) error {
	if c.Config.App.Env == "production" {
		return fmt.Errorf("refusing to seed a production environment")
	}
	svc := user.New(user.Deps{DB: c.DB}).Service()
	accounts := []user.RegisterInput{
		{Email: "admin@example.com", Name: "Ada Admin", Password: "admin12345", Role: auth.RoleAdmin},
		{Email: "customer@example.com", Name: "Carl Customer", Password: "customer123", Role: auth.RoleCustomer},
	}
	for _, in := range accounts {
		_, created, err := svc.EnsureUser(ctx, in)
		if err != nil {
			return fmt.Errorf("seed %s: %w", in.Email, err)
		}
		c.Log.Info("seed account", "email", in.Email, "created", created)
	}
	return nil
}

// CreateMigration writes an empty, timestamped goose SQL file into dir.
func CreateMigration(dir, name string) (string, error) {
	file := filepath.Join(dir, time.Now().UTC().Format("20060102150405")+"_"+name+".sql")
	body := "-- +goose Up\n-- +goose StatementBegin\n\n-- +goose StatementEnd\n\n-- +goose Down\n-- +goose StatementBegin\n\n-- +goose StatementEnd\n"
	if err := os.WriteFile(file, []byte(body), 0o644); err != nil {
		return "", err
	}
	return file, nil
}
