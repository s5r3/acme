package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadPrecedence(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "config.yaml")
	write(t, base, `
database:
  dsn: "postgres://file"
auth:
  jwt_secret: "0123456789abcdef0123456789abcdef"
api:
  address: ":9000"
`)
	write(t, filepath.Join(dir, "config.local.yaml"), `
api:
  address: ":9100"
`)
	t.Setenv("ACME_DATABASE_DSN", "postgres://env")
	t.Setenv("ACME_WEB_SESSION_TTL", "2h")

	cfg, err := Load(base)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Database.DSN != "postgres://env" {
		t.Errorf("env should win over file: dsn = %q", cfg.Database.DSN)
	}
	if cfg.API.Address != ":9100" {
		t.Errorf("local file should win over base: address = %q", cfg.API.Address)
	}
	if cfg.Web.SessionTTL != 2*time.Hour || cfg.Web.Address != ":8081" {
		t.Errorf("web = %+v", cfg.Web)
	}
	if cfg.Database.SlowQueryThreshold != 200*time.Millisecond {
		t.Errorf("default not applied: %v", cfg.Database.SlowQueryThreshold)
	}
}

func TestLoadRejectsInvalidConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	write(t, path, `
database:
  dsn: "postgres://x"
auth:
  jwt_secret: "too-short"
`)
	if _, err := Load(path); err == nil {
		t.Fatal("expected a validation error for a short jwt_secret")
	}
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}
