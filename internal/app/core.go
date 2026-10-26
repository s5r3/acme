// Package app is the composition root: the only package that knows every
// concrete type. It builds what both binaries share (Core) and the Echo
// instance for each binary.
package app

import (
	"context"
	"fmt"
	"log/slog"

	"gorm.io/gorm"

	"example.com/acme/internal/platform/auth"
	"example.com/acme/internal/platform/config"
	"example.com/acme/internal/platform/database"
	"example.com/acme/internal/platform/logger"
	"example.com/acme/internal/platform/web"
	"example.com/acme/internal/shared/clock"
)

// Version is set at build time: -ldflags "-X example.com/acme/internal/app.Version=1.2.3".
var Version = "dev"

// Core is everything both binaries share.
type Core struct {
	Config   *config.Config
	Log      *slog.Logger
	DB       *gorm.DB
	Tx       database.TxManager
	Roles    *auth.Roles
	Tokens   *auth.Tokens
	Sessions *web.Sessions
	Modules  []Module

	closers []func()
}

// NewCore opens the database, builds the platform services and every
// feature module. service names the binary in log lines ("api", "web").
func NewCore(ctx context.Context, cfg *config.Config, service string) (*Core, error) {
	log, closeLog := logger.New(cfg.Log, cfg.App.Name+"-"+service, Version)
	slog.SetDefault(log)

	db, err := database.Open(ctx, cfg.Database, log)
	if err != nil {
		closeLog()
		return nil, fmt.Errorf("open database: %w", err)
	}

	clk := clock.System{}
	roles := auth.NewRoles()
	c := &Core{
		Config:   cfg,
		Log:      log,
		DB:       db,
		Tx:       database.NewTxManager(db),
		Roles:    roles,
		Tokens:   auth.NewTokens(cfg.Auth, roles, clk),
		Sessions: web.NewSessions(db, roles, cfg.Web, clk),
		closers:  []func(){func() { database.Close(db) }, closeLog},
	}
	c.Modules = buildModules(c)
	for _, m := range c.Modules {
		if pm, ok := m.(PermissionModule); ok {
			pm.RegisterPermissions(roles)
		}
	}
	return c, nil
}

// LoadCore loads config from cfgPath and builds Core.
func LoadCore(ctx context.Context, cfgPath, service string) (*Core, error) {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	return NewCore(ctx, cfg, service)
}

// Close releases the database and the log file.
func (c *Core) Close() {
	for _, fn := range c.closers {
		fn()
	}
}
