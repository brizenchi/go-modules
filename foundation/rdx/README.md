# foundation/rdx

> Redis client setup + a small distributed `Lock` primitive.

[![Go Reference](https://pkg.go.dev/badge/github.com/brizenchi/go-modules/foundation/rdx.svg)](https://pkg.go.dev/github.com/brizenchi/go-modules/foundation/rdx)

Wraps `go-redis/v9` with sane pool defaults, a key-prefix convention, and
a SET-NX-PX-based lock for "make sure exactly one worker does X for the
next N seconds" workflows. **Not** a replacement for Redlock or full
coordination — intentionally minimal.

## Install

```bash
go get github.com/brizenchi/go-modules/foundation/rdx
```

## Quick start

```go
import (
    "context"
    "time"
    "github.com/brizenchi/go-modules/foundation/rdx"
)

client, err := rdx.Open(context.Background(), rdx.Config{
    Addr:      "localhost:6379",
    KeyPrefix: "myapp:",
})
if err != nil { log.Fatal(err) }

if err := rdx.HealthCheck(context.Background(), client); err != nil {
    log.Fatal(err)
}

// Distributed lock:
lock, ok, err := rdx.Acquire(ctx, client, "send-daily-report", 30*time.Second)
if err != nil { return err }
if !ok { return nil } // someone else has it
defer lock.Unlock(ctx)

// ... do the work that should run at most once per minute ...
```

## Configuration

| Field           | Default | Notes |
|-----------------|---------|-------|
| `Addr`          | —       | Required |
| `Password`      | —       |  |
| `DB`            | 0       |  |
| `PoolSize`      | 10      |  |
| `MinIdleConns`  | 2       |  |
| `DialTimeout`   | 5s      |  |
| `ReadTimeout`   | 3s      |  |
| `WriteTimeout`  | 3s      |  |
| `KeyPrefix`     | —       | Useful when one Redis serves many envs |

## Lock semantics

- Acquire: SET NX PX — atomic, returns `(nil, false, nil)` if already held
- Unlock: Lua-scripted compare-and-delete — won't release a lock that
  was already taken over by another caller after TTL expiry
- **Not re-entrant**. **Not high-contention**. For those: real coordinator.

## Testing

Run against a local Redis (or skipped in CI):

```bash
go test -race ./...
```

## Changelog

See [CHANGELOG.md](./CHANGELOG.md).
