// Package logger builds the application's slog.Logger. lumberjack is only an
// io.Writer used underneath; nothing outside this package imports it.
package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"

	"example.com/acme/internal/platform/config"
)

// redactedKeys are never written to logs, whatever level.
var redactedKeys = map[string]bool{
	"password":      true,
	"password_hash": true,
	"token":         true,
	"access_token":  true,
	"authorization": true,
	"secret":        true,
	"jwt_secret":    true,
	"dsn":           true,
}

// New returns the base logger and a function that closes the log file.
func New(cfg config.LogConfig, service, version string) (*slog.Logger, func()) {
	var writers []io.Writer
	if cfg.Stdout {
		writers = append(writers, os.Stdout)
	}

	var rot *lumberjack.Logger
	if cfg.File.Path != "" {
		rot = &lumberjack.Logger{
			Filename:   cfg.File.Path,
			MaxSize:    cfg.File.MaxSizeMB,
			MaxBackups: cfg.File.MaxBackups,
			MaxAge:     cfg.File.MaxAgeDays,
			Compress:   cfg.File.Compress,
		}
		writers = append(writers, rot)
	}
	if len(writers) == 0 {
		writers = append(writers, io.Discard)
	}

	h := slog.NewJSONHandler(io.MultiWriter(writers...), &slog.HandlerOptions{
		Level:       ParseLevel(cfg.Level),
		ReplaceAttr: redact,
	})
	log := slog.New(h).With("service", service, "version", version)

	return log, func() {
		if rot != nil {
			_ = rot.Close()
		}
	}
}

// ParseLevel converts a config string to a slog level (default info).
func ParseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func redact(_ []string, a slog.Attr) slog.Attr {
	if redactedKeys[strings.ToLower(a.Key)] {
		return slog.String(a.Key, "[REDACTED]")
	}
	return a
}

type ctxKey struct{}

// WithContext stores a request-scoped logger in ctx.
func WithContext(ctx context.Context, log *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, log)
}

// From returns the request-scoped logger, or slog.Default when none is set.
// Services use this so their logs carry request_id and user_id.
func From(ctx context.Context) *slog.Logger {
	if log, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok {
		return log
	}
	return slog.Default()
}
