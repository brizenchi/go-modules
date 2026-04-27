# foundation/pgx

> GORM-over-Postgres helper: standardized connection pool, slow-query logging, health check.

[![Go Reference](https://pkg.go.dev/badge/github.com/brizenchi/go-modules/foundation/pgx.svg)](https://pkg.go.dev/github.com/brizenchi/go-modules/foundation/pgx)

Use `Open(cfg)` once at boot and share the returned `*gorm.DB` across all
repositories. `HealthCheck(ctx, db)` is intended for `/healthz` handlers.

## Install

```bash
go get github.com/brizenchi/go-modules/foundation/pgx
```

## Quick start

```go
import (
    "context"
    "github.com/brizenchi/go-modules/foundation/pgx"
)

db, err := pgx.Open(pgx.Config{
    DSN:                "postgres://user:pass@host:5432/db?sslmode=require",
    MaxOpenConns:       50,
    MaxIdleConns:       10,
    SlowQueryThreshold: 200 * time.Millisecond,
    LogLevel:           "warn",
})
if err != nil { log.Fatal(err) }

if err := pgx.HealthCheck(context.Background(), db); err != nil {
    log.Fatal(err)
}
```

## Configuration

DSN form **OR** discrete fields — set one or the other:

```go
pgx.Config{
    DSN: "postgres://user:pass@host:5432/db?sslmode=disable",
}

// equivalent:
pgx.Config{
    Host:     "host",
    Port:     5432,
    User:     "user",
    Password: "pass",
    Database: "db",
    SSLMode:  "disable",
    TimeZone: "UTC",
}
```

| Field                 | Default | Notes |
|-----------------------|---------|-------|
| `MaxOpenConns`        | 25      |  |
| `MaxIdleConns`        | 5       |  |
| `ConnMaxLifetime`     | 30m     |  |
| `ConnMaxIdleTime`     | 5m      |  |
| `SlowQueryThreshold`  | 200ms   | Set 0 to disable slow log |
| `LogLevel`            | `warn`  | `silent` / `error` / `warn` / `info` |

## Slow query logging

Queries exceeding `SlowQueryThreshold` are logged at `warn` via slog with
the SQL, duration, and rows-affected. Pair with `foundation/slog` for
structured output.

## Testing

```bash
go test -race ./...
```

## Changelog

See [CHANGELOG.md](./CHANGELOG.md).
