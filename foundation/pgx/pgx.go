// Package pgx is a thin GORM-over-Postgres helper that standardizes
// connection setup, pool sizing, and slow-query logging.
//
// Use Open(cfg) at boot, get a *gorm.DB, share it across all repos.
// HealthCheck(db) is for Kubernetes /healthz handlers.
//
// With Tracing enabled the database/sql driver is wrapped by otelsql:
// every statement becomes an OpenTelemetry client span
// (db.system.name=postgresql, db.query.text with placeholders — bind
// values are never recorded) and connection-pool metrics are reported
// through the global MeterProvider.
//
// Stdlib + GORM-postgres + otelsql.
package pgx

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/XSAM/otelsql"
	pgxdriver "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
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

	// LogSQLParams keeps bind values in logged SQL. Default false: SQL is
	// logged with "?" / "$n" placeholders so passwords, tokens and
	// personal data never reach the log pipeline.
	LogSQLParams bool

	// LogRecordNotFound logs gorm.ErrRecordNotFound at ERROR. Default
	// false, because a miss is normal control flow for most lookups.
	LogRecordNotFound bool

	// Tracing wraps the driver with otelsql: one client span per
	// statement (without bind values) plus sql.DBStats pool metrics.
	Tracing bool

	// Project and Environment are copied onto DB log records so external
	// log backends can partition data without guessing from service names.
	Project     string
	Environment string
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

	dialector := postgres.Open(dsn)
	var traced *sql.DB
	if cfg.Tracing {
		var err error
		if traced, err = openTraced(dsn); err != nil {
			return nil, err
		}
		dialector = postgres.New(postgres.Config{Conn: traced})
	}

	db, err := gorm.Open(dialector, gormCfg)
	if err != nil {
		if traced != nil {
			_ = traced.Close()
		}
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

// openTraced builds the same pgx stdlib connection GORM would, wrapped by
// otelsql. Row iteration and session resets are not traced to keep one
// span per statement.
func openTraced(dsn string) (*sql.DB, error) {
	connCfg, err := pgxdriver.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("pgx: parse dsn: %w", err)
	}
	opts := otelOptions()
	db := otelsql.OpenDB(stdlib.GetConnector(*connCfg), opts...)
	if _, err := otelsql.RegisterDBStatsMetrics(db, opts...); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("pgx: register pool metrics: %w", err)
	}
	return db, nil
}

func otelOptions() []otelsql.Option {
	return []otelsql.Option{
		otelsql.WithAttributes(semconv.DBSystemNamePostgreSQL),
		otelsql.WithSpanOptions(otelsql.SpanOptions{
			OmitConnResetSession: true,
			OmitRows:             true,
		}),
	}
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
		level:       level,
		threshold:   threshold,
		project:     cfg.Project,
		env:         cfg.Environment,
		logParams:   cfg.LogSQLParams,
		logNotFound: cfg.LogRecordNotFound,
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
	level       gormlogger.LogLevel
	threshold   time.Duration
	project     string
	env         string
	logParams   bool
	logNotFound bool
}

// ParamsFilter implements gorm.ParamsFilter. Returning nil params makes
// GORM render SQL with placeholders instead of interpolated values.
func (l *slogLogger) ParamsFilter(_ context.Context, sql string, params ...any) (string, []any) {
	if l.logParams {
		return sql, params
	}
	return sql, nil
}

func (l *slogLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	cp := *l
	cp.level = level
	return &cp
}

func (l *slogLogger) Info(ctx context.Context, msg string, args ...any) {
	if l.level >= gormlogger.Info {
		slog.InfoContext(ctx, msg, l.appendCommonAttrs("db_log", "info", "args", args)...)
	}
}
func (l *slogLogger) Warn(ctx context.Context, msg string, args ...any) {
	if l.level >= gormlogger.Warn {
		slog.WarnContext(ctx, msg, l.appendCommonAttrs("db_log", "warn", "args", args)...)
	}
}
func (l *slogLogger) Error(ctx context.Context, msg string, args ...any) {
	if l.level >= gormlogger.Error {
		slog.ErrorContext(ctx, msg, l.appendCommonAttrs("db_log", "error", "args", args)...)
	}
}

func (l *slogLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	switch {
	case err != nil && l.level >= gormlogger.Error &&
		(l.logNotFound || !errors.Is(err, gorm.ErrRecordNotFound)):
		sql, rows := fc()
		slog.ErrorContext(ctx, "gorm error",
			l.appendCommonAttrs("error", "error",
				"error", err,
				"sql", sql,
				"rows", rows,
				"elapsed_ms", elapsed.Milliseconds(),
			)...)
	case elapsed > l.threshold && l.threshold > 0 && l.level >= gormlogger.Warn:
		sql, rows := fc()
		slog.WarnContext(ctx, "gorm slow query",
			l.appendCommonAttrs("slow_query", "warn",
				"sql", sql,
				"rows", rows,
				"elapsed_ms", elapsed.Milliseconds(),
				"threshold_ms", l.threshold.Milliseconds(),
			)...)
	case l.level >= gormlogger.Info:
		sql, rows := fc()
		slog.InfoContext(ctx, "gorm query",
			l.appendCommonAttrs("query", "info",
				"sql", sql,
				"rows", rows,
				"elapsed_ms", elapsed.Milliseconds(),
			)...)
	}
}

func (l *slogLogger) appendCommonAttrs(operation, severity string, attrs ...any) []any {
	base := []any{
		"component", "db",
		"db_system", "postgres",
		"operation", operation,
		"severity", severity,
	}
	if l.project != "" {
		base = append(base, "project", l.project)
	}
	if l.env != "" {
		base = append(base, "env", l.env)
	}
	return append(base, attrs...)
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
