// Package config loads the application configuration. Viper is used only in
// this package; the rest of the code receives typed structs.
package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

// Config is the root configuration shared by every binary.
type Config struct {
	App      AppConfig      `mapstructure:"app"`
	API      HTTPConfig     `mapstructure:"api"`
	Web      WebConfig      `mapstructure:"web"`
	Database DatabaseConfig `mapstructure:"database"`
	Log      LogConfig      `mapstructure:"log"`
	Auth     AuthConfig     `mapstructure:"auth"`
}

type AppConfig struct {
	Name string `mapstructure:"name" validate:"required"`
	Env  string `mapstructure:"env" validate:"oneof=development test production"`
}

// HTTPConfig describes one HTTP server (one per binary).
type HTTPConfig struct {
	Address         string        `mapstructure:"address" validate:"required"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
	RequestTimeout  time.Duration `mapstructure:"request_timeout"`
	BodyLimitBytes  int64         `mapstructure:"body_limit_bytes" validate:"gt=0"`
	CORSOrigins     []string      `mapstructure:"cors_origins"`
}

// WebConfig is the HTML front-end's server config plus session settings.
type WebConfig struct {
	HTTPConfig   `mapstructure:",squash"`
	SessionTTL   time.Duration `mapstructure:"session_ttl" validate:"gt=0"`
	CookieSecure bool          `mapstructure:"cookie_secure"`
}

type DatabaseConfig struct {
	DSN                string        `mapstructure:"dsn" validate:"required"`
	MaxOpenConns       int           `mapstructure:"max_open_conns"`
	MaxIdleConns       int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime    time.Duration `mapstructure:"conn_max_lifetime"`
	SlowQueryThreshold time.Duration `mapstructure:"slow_query_threshold"`
}

type LogConfig struct {
	Level  string        `mapstructure:"level" validate:"oneof=debug info warn error"`
	Stdout bool          `mapstructure:"stdout"`
	File   LogFileConfig `mapstructure:"file"`
}

// LogFileConfig configures the lumberjack rotating file writer.
// An empty Path disables file logging.
type LogFileConfig struct {
	Path       string `mapstructure:"path"`
	MaxSizeMB  int    `mapstructure:"max_size_mb"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAgeDays int    `mapstructure:"max_age_days"`
	Compress   bool   `mapstructure:"compress"`
}

type AuthConfig struct {
	JWTSecret string        `mapstructure:"jwt_secret" validate:"required,min=32"`
	Issuer    string        `mapstructure:"issuer" validate:"required"`
	TokenTTL  time.Duration `mapstructure:"token_ttl" validate:"gt=0"`
}

// EnvPrefix is prepended to every environment override: ACME_DATABASE_DSN.
const EnvPrefix = "ACME"

// Load reads defaults, the YAML file at path, an optional sibling
// "<name>.local.yaml", then environment variables, in that order.
func Load(path string) (*Config, error) {
	v := viper.New() // never the global instance
	setDefaults(v)

	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if err := mergeIfExists(v, localOverridePath(path)); err != nil {
		return nil, err
	}

	v.SetEnvPrefix(EnvPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}
	if err := validator.New().Struct(cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return &cfg, nil
}

// setDefaults registers every key. AutomaticEnv only overrides keys Viper
// already knows, so a key without a default cannot be set from the env.
func setDefaults(v *viper.Viper) {
	v.SetDefault("app.name", "acme")
	v.SetDefault("app.env", "development")

	for _, srv := range []string{"api", "web"} {
		v.SetDefault(srv+".shutdown_timeout", "15s")
		v.SetDefault(srv+".request_timeout", "30s")
		v.SetDefault(srv+".body_limit_bytes", 1<<20)
		v.SetDefault(srv+".cors_origins", []string{})
	}
	v.SetDefault("api.address", ":8080")
	v.SetDefault("web.address", ":8081")
	v.SetDefault("web.session_ttl", "12h")
	v.SetDefault("web.cookie_secure", false)

	v.SetDefault("database.dsn", "")
	v.SetDefault("database.max_open_conns", 20)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.conn_max_lifetime", "30m")
	v.SetDefault("database.slow_query_threshold", "200ms")

	v.SetDefault("log.level", "info")
	v.SetDefault("log.stdout", true)
	v.SetDefault("log.file.path", "")
	v.SetDefault("log.file.max_size_mb", 100)
	v.SetDefault("log.file.max_backups", 10)
	v.SetDefault("log.file.max_age_days", 30)
	v.SetDefault("log.file.compress", true)

	v.SetDefault("auth.jwt_secret", "")
	v.SetDefault("auth.issuer", "acme")
	v.SetDefault("auth.token_ttl", "1h")
}

func localOverridePath(path string) string {
	if ext := ".yaml"; strings.HasSuffix(path, ext) {
		return strings.TrimSuffix(path, ext) + ".local" + ext
	}
	return path + ".local"
}

func mergeIfExists(v *viper.Viper, path string) error {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()
	if err := v.MergeConfig(f); err != nil {
		return fmt.Errorf("merge %s: %w", path, err)
	}
	return nil
}
