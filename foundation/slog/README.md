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
| `RedactKeys` | `[]string`               | nil            | Extra secret keys on top of `DefaultRedactKeys` |

## Correlation fields

Records written with the `*Context` methods automatically include, when
present in ctx: `request_id`, `project`, `env`, `tenant_id`, `user_id`,
`trace_id`, `span_id`, `trace_flags`. A key the caller already passed (or
bound with `Logger.With`) is not written twice.

```go
slog.InfoContext(ctx, "invoice paid", "invoice_id", id) // always pass ctx
```

## Redaction

Values of `password`, `secret`, `token`, `access_token`, `refresh_token`,
`authorization`, `cookie`, `api_key`, `client_secret`, ... are replaced with
`[REDACTED]` (case-insensitive, `-` and `_` treated alike). Add project keys
with `RedactKeys`.

## Storage

Logs go to stdout only (12-factor). Rotation, shipping and retention belong to
the platform: Docker/Kubernetes log drivers plus Vector, Fluent Bit or the
OpenTelemetry Collector `filelog` receiver.

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
