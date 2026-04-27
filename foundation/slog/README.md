# foundation/slog

> Boot-time setup helper around the standard library's `log/slog`.

[![Go Reference](https://pkg.go.dev/badge/github.com/brizenchi/go-modules/foundation/slog.svg)](https://pkg.go.dev/github.com/brizenchi/go-modules/foundation/slog)

This package does **not** wrap or replace `log/slog` — it just standardizes
how every service in the org configures the global logger at boot. All
business code keeps calling `log/slog` directly.

## Install

```bash
go get github.com/brizenchi/go-modules/foundation/slog
```

## Quick start

```go
import (
    "log/slog"
    flog "github.com/brizenchi/go-modules/foundation/slog"
)

func main() {
    flog.Setup(flog.Config{
        Level:  "info",         // debug | info | warn | error
        Format: flog.FormatJSON, // FormatJSON or FormatText
        Defaults: map[string]any{
            "service": "auth-svc",
            "env":     "prod",
        },
    })
    slog.Info("ready", "port", 8080) // every record now carries service+env
}
```

## Configuration

| Field        | Type                     | Default        | Notes |
|--------------|--------------------------|----------------|-------|
| `Level`      | `string`                 | `"info"`       | `debug` / `info` / `warn` / `error` |
| `Format`     | `Format`                 | `FormatJSON`   | `FormatJSON` for log shippers, `FormatText` for humans |
| `AddSource`  | `bool`                   | `false`        | Attach `file:line` to every record |
| `Output`     | `io.Writer`              | `os.Stdout`    | Override for tests |
| `Defaults`   | `map[string]any`         | nil            | Attributes attached to every record |

## Gin integration

Use [`foundation/ginx`](../ginx/) for `Recover` / `RequestID` / `AccessLog`
middleware — they emit `slog` records with the request id automatically.

## Testing

```bash
go test -race ./...
```

Coverage: 83.8%.

## Changelog

See [CHANGELOG.md](./CHANGELOG.md).
