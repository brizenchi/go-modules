# foundation/tracing

> OpenTelemetry tracing setup for Gin services, with request-id-aware middleware.

[![Go Reference](https://pkg.go.dev/badge/github.com/brizenchi/go-modules/foundation/tracing.svg)](https://pkg.go.dev/github.com/brizenchi/go-modules/foundation/tracing)

This module standardizes distributed tracing across services that
already use `foundation/ginx`, `foundation/slog`, and `foundation/httpx`.

It provides:

- `Setup(Config)` for boot-time tracer-provider initialization
- `Trace(serviceName)` for inbound Gin request tracing

## Install

```bash
go get github.com/brizenchi/go-modules/foundation/tracing
```

## Quick start

```go
import (
    "context"
    "log"

    "github.com/brizenchi/go-modules/foundation/ginx"
    "github.com/brizenchi/go-modules/foundation/tracing"
    "github.com/gin-gonic/gin"
)

func main() {
    shutdown, err := tracing.Setup(tracing.Config{
        ServiceName: "api",
        Endpoint:    "localhost:4318",
        Protocol:    "http",
        Insecure:    true,
        SampleRate:  1,
    })
    if err != nil { log.Fatal(err) }
    defer tracing.Shutdown(context.Background(), shutdown)

    r := gin.New()
    r.Use(
        ginx.Recover(),
        ginx.RequestID(),
        tracing.Trace("api"),
        ginx.AccessLog(ginx.AccessLogConfig{}),
    )
}
```

## Middleware order

Recommended order:

1. `ginx.Recover()`
2. `ginx.RequestID()`
3. `tracing.Trace(serviceName)`
4. `ginx.AccessLog(...)`

That order ensures `request_id` is tagged onto the active span and
`trace_id` / `span_id` are available to the access log.

## Config

| Field | Meaning |
|---|---|
| `ServiceName` | Required service name reported to the trace backend |
| `Project` | Optional SaaS/project grouping; exported as `service.namespace` |
| `Environment` | Optional environment; exported as `deployment.environment` |
| `Endpoint` | OTLP collector host:port; empty disables exporter |
| `Protocol` | `"http"` (default) or `"grpc"` |
| `Insecure` | Disable TLS for local collectors |
| `SampleRate` | Fraction of traces to sample, `0..1` |

When `Endpoint` is empty, tracing is disabled cleanly and `Setup`
returns a no-op shutdown function. This keeps tracing boot code stable
across all services even when some environments do not export spans.

## Outbound propagation

Pair this module with [`foundation/httpx`](../httpx/) and set
`httpx.Config{Tracing: true}` to inject W3C Trace Context headers into
outgoing HTTP requests.

## Testing

```bash
go test -race ./...
```

## Changelog

See [CHANGELOG.md](./CHANGELOG.md).
