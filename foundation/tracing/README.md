# foundation/tracing

> OpenTelemetry setup for Gin services: traces, metrics and W3C propagation over OTLP.

[![Go Reference](https://pkg.go.dev/badge/github.com/brizenchi/go-modules/foundation/tracing.svg)](https://pkg.go.dev/github.com/brizenchi/go-modules/foundation/tracing)

It provides:

- `Setup(Config)`: global TracerProvider, optional MeterProvider, TraceContext +
  Baggage propagators, OTLP HTTP/gRPC exporters, SDK errors routed to slog
- `Middleware(MiddlewareConfig)` / `Trace(serviceName)`: inbound Gin
  instrumentation built on the official
  [`otelgin`](https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin)
- `TraceID(ctx)` / `SpanID(ctx)`

Pair it with `foundation/httpx` (`Tracing: true`), `foundation/pgx`
(`Tracing: true`) and `foundation/rdx` (`Tracing: true`) for outbound HTTP,
SQL and Redis spans.

## Install

```bash
go get github.com/brizenchi/go-modules/foundation/tracing
```

## Quick start

```go
flog.Setup(flog.Config{Format: flog.FormatJSON}) // first: OTel errors are logged through slog

shutdown, err := tracing.Setup(tracing.Config{
    ServiceName: "billing-api",
    Project:     "acme",
    Environment: "prod",
    Endpoint:    "otel-collector:4318", // "http://..." / "https://..." also accepted
    SampleRate:  0.2,
    Metrics:     true,
})
if err != nil {
    log.Fatal(err)
}
defer tracing.Shutdown(context.Background(), shutdown)

skip := []string{"/health"}
r := gin.New()
r.Use(
    ginx.RequestID(),
    tracing.Middleware(tracing.MiddlewareConfig{ServiceName: "billing-api", SkipPaths: skip}),
    ginx.AccessLog(ginx.AccessLogConfig{SkipPaths: skip}),
    ginx.Recover(),
)
```

`Recover` is innermost on purpose: a panic is recovered inside the server
span, so the span ends with status Error and `http.response.status_code=500`,
and the access log still records the request.

## What gets recorded

| Signal | Content |
|--------|---------|
| Server span | name `GET /users/:id`; `http.request.method`, `http.route`, `url.path`, `url.scheme`, `http.response.status_code`, `client.address`, `user_agent.original`, `http.request_id`; status Error for 5xx and `c.Error` |
| Metrics | `http.server.request.duration`, request/response body size (when `Metrics` is on) |
| Resource | `service.name`, `service.version`, `service.namespace`, `deployment.environment.name`, `telemetry.sdk.*`, `host.name`, `process.runtime.*` |

`OTEL_SERVICE_NAME` and `OTEL_RESOURCE_ATTRIBUTES` override the resource
attributes from `Config`.

## Sampling

`ParentBased(TraceIDRatioBased(SampleRate))`: new traces are sampled at
`SampleRate`; requests with an incoming `traceparent` follow the caller's
decision, so a distributed trace is never cut in the middle. Spans are created
even when `Endpoint` is empty, so `trace_id` still correlates logs.

## Configuration

| Field | Default | Notes |
|-------|---------|-------|
| `ServiceName` | required | `service.name` |
| `ServiceVersion` | — | `service.version` |
| `Project` | — | `service.namespace` |
| `Environment` | — | `deployment.environment.name` |
| `Endpoint` | — | `host:port`; empty disables export |
| `Protocol` | `http` | `http` or `grpc` |
| `Insecure` | `false` | plain-text OTLP |
| `SampleRate` | `0` | fraction of new traces |
| `Headers` | — | e.g. `Authorization` for OpenObserve |
| `URLPath` | `/v1/traces` | HTTP only |
| `Metrics` | `false` | OTLP metrics; requires `Endpoint` |
| `MetricsURLPath` | derived | `URLPath` with `/v1/traces` → `/v1/metrics` |
| `MetricsInterval` | `60s` | export period |

## Testing

```bash
go test -race ./...
```

## Changelog

See [CHANGELOG.md](./CHANGELOG.md).
