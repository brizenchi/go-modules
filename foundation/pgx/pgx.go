// Package pgx is a thin GORM-over-Postgres helper that standardizes
// connection setup, pool sizing, and slow-query logging.
//
// Use Open(cfg) at boot, get a *gorm.DB, share it across all repos.
// HealthCheck(db) is for Kubernetes /healthz handlers.
//
// Stdlib + GORM-postgres only.
package pgx

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Config configures the Postgres connection.
type Config struct {
	// DSN: "postgres://user:pass@host:5432/db?sslmode=disable"
	// or key=value: "host=... user=... password=... dbname=... port=5432 sslmode=disable"
	// One of DSN or the discrete fields below must be set.
	DSN string

	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string // disable | require | verify-ca | verify-full
	TimeZone string // e.g. "UTC"

	// Pool tuning. Sane defaults applied if zero.
	MaxOpenConns    int           // default 25
	MaxIdleConns    int           // default 5
	ConnMaxLifetime time.Duration // default 30m
	ConnMaxIdleTime time.Duration // default 5m

	// SlowQueryThreshold: queries slower than this are logged at WARN.
	// Set to 0 to disable slow logging. Default 200ms.
	SlowQueryThreshold time.Duration

	// LogLevel: silent | error | warn | info. Default warn.
	LogLevel string
}

func (c Config) effectiveDSN() string {
	if c.DSN != "" {
		return c.DSN
	}
	tz := c.TimeZone
	if tz == "" {
		tz = "UTC"
	}
	ssl := c.SSLMode
	if ssl == "" {
		ssl = "disable"
	}
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=%s",
		c.Host, c.User, c.Password, c.Database, c.Port, ssl, tz)
}

// Open establishes the connection and applies pool tuning. Returns a
// *gorm.DB ready to share across repositories.
func Open(cfg Config) (*gorm.DB, error) {
	dsn := cfg.effectiveDSN()
	if dsn == "" {
		return nil, fmt.Errorf("pgx: dsn or host/user/database required")
	}

	gormCfg := &gorm.Config{
		Logger: buildLogger(cfg),
	}

	db, err := gorm.Open(postgres.Open(dsn), gormCfg)
	if err != nil {
		return nil, fmt.Errorf("pgx: open: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("pgx: get *sql.DB: %w", err)
	}
	sqlDB.SetMaxOpenConns(nonZeroInt(cfg.MaxOpenConns, 25))
	sqlDB.SetMaxIdleConns(nonZeroInt(cfg.MaxIdleConns, 5))
	sqlDB.SetConnMaxLifetime(nonZeroDur(cfg.ConnMaxLifetime, 30*time.Minute))
	sqlDB.SetConnMaxIdleTime(nonZeroDur(cfg.ConnMaxIdleTime, 5*time.Minute))

	return db, nil
}

// HealthCheck pings the underlying *sql.DB with a short timeout.
func HealthCheck(ctx context.Context, db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return sqlDB.PingContext(pingCtx)
}

// buildLogger wires GORM's logger into slog with the configured level
// and slow-query threshold.
func buildLogger(cfg Config) gormlogger.Interface {
	threshold := cfg.SlowQueryThreshold
	if threshold == 0 {
		threshold = 200 * time.Millisecond
	}
	level := parseLevel(cfg.LogLevel)
	return &slogLogger{
		level:     level,
		threshold: threshold,
	}
}

func parseLevel(s string) gormlogger.LogLevel {
	switch s {
	case "silent":
		return gormlogger.Silent
	case "error":
		return gormlogger.Error
	case "info":
		return gormlogger.Info
	default:
		return gormlogger.Warn
	}
}

type slogLogger struct {
	level     gormlogger.LogLevel
	threshold time.Duration
}

func (l *slogLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	cp := *l
	cp.level = level
	return &cp
}

func (l *slogLogger) Info(ctx context.Context, msg string, args ...any) {
	if l.level >= gormlogger.Info {
		slog.InfoContext(ctx, msg, "args", args)
	}
}
func (l *slogLogger) Warn(ctx context.Context, msg string, args ...any) {
	if l.level >= gormlogger.Warn {
		slog.WarnContext(ctx, msg, "args", args)
	}
}
func (l *slogLogger) Error(ctx context.Context, msg string, args ...any) {
	if l.level >= gormlogger.Error {
		slog.ErrorContext(ctx, msg, "args", args)
	}
}

func (l *slogLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	switch {
	case err != nil && l.level >= gormlogger.Error:
		sql, rows := fc()
		slog.ErrorContext(ctx, "gorm error",
			"error", err, "sql", sql, "rows", rows, "elapsed_ms", elapsed.Milliseconds())
	case elapsed > l.threshold && l.threshold > 0 && l.level >= gormlogger.Warn:
		sql, rows := fc()
		slog.WarnContext(ctx, "gorm slow query",
			"sql", sql, "rows", rows, "elapsed_ms", elapsed.Milliseconds(),
			"threshold_ms", l.threshold.Milliseconds())
	case l.level >= gormlogger.Info:
		sql, rows := fc()
		slog.InfoContext(ctx, "gorm query",
			"sql", sql, "rows", rows, "elapsed_ms", elapsed.Milliseconds())
	}
}

func nonZeroInt(v, def int) int {
	if v > 0 {
		return v
	}
	return def
}
func nonZeroDur(v, def time.Duration) time.Duration {
	if v > 0 {
		return v
	}
	return def
}
