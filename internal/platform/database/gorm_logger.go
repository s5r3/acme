package database

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"example.com/acme/internal/platform/logger"
)

// gormLogger sends GORM's logs to slog. It uses the request-scoped logger
// from ctx so query logs carry request_id.
type gormLogger struct {
	base *slog.Logger
	slow time.Duration
}

func newGormLogger(base *slog.Logger, slow time.Duration) gormlogger.Interface {
	return &gormLogger{base: base.With("component", "gorm"), slow: slow}
}

func (l *gormLogger) LogMode(gormlogger.LogLevel) gormlogger.Interface { return l }

// ParamsFilter drops bound parameter values from logged SQL, so passwords,
// tokens and personal data never reach the logs.
func (l *gormLogger) ParamsFilter(_ context.Context, sql string, _ ...any) (string, []any) {
	return sql, nil
}

func (l *gormLogger) log(ctx context.Context) *slog.Logger {
	if log := logger.From(ctx); log != slog.Default() {
		return log.With("component", "gorm")
	}
	return l.base
}

func (l *gormLogger) Info(ctx context.Context, msg string, args ...any) {
	l.log(ctx).Info(fmt.Sprintf(msg, args...))
}

func (l *gormLogger) Warn(ctx context.Context, msg string, args ...any) {
	l.log(ctx).Warn(fmt.Sprintf(msg, args...))
}

func (l *gormLogger) Error(ctx context.Context, msg string, args ...any) {
	l.log(ctx).Error(fmt.Sprintf(msg, args...))
}

// Trace logs failed queries at Error, slow queries at Warn and everything
// else at Debug (off in production).
func (l *gormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	log := l.log(ctx)
	switch {
	case err != nil && !errors.Is(err, gorm.ErrRecordNotFound) && !errors.Is(err, gorm.ErrDuplicatedKey):
		// not found / duplicate key are expected outcomes the repository
		// translates into feature errors; anything else is a real failure.
		sql, rows := fc()
		log.Error("query failed", "sql", sql, "rows", rows, "elapsed_ms", elapsed.Milliseconds(), "err", err)
	case l.slow > 0 && elapsed > l.slow:
		sql, rows := fc()
		log.Warn("slow query", "sql", sql, "rows", rows, "elapsed_ms", elapsed.Milliseconds())
	case log.Enabled(ctx, slog.LevelDebug):
		sql, rows := fc()
		log.Debug("query", "sql", sql, "rows", rows, "elapsed_ms", elapsed.Milliseconds())
	}
}
