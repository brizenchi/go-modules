# foundation/ginx

> Standard Gin middleware bundle: Recover / RequestID / AccessLog / CORS / NoCache / Secure.

[![Go Reference](https://pkg.go.dev/badge/github.com/brizenchi/go-modules/foundation/ginx.svg)](https://pkg.go.dev/github.com/brizenchi/go-modules/foundation/ginx)

Each middleware is independent — pick what you need. They all emit
`log/slog` records with the request id, so pair with
[`foundation/slog`](../slog/) for the best experience.

## Install

```bash
go get github.com/brizenchi/go-modules/foundation/ginx
```

## Quick start

```go
import "github.com/brizenchi/go-modules/foundation/ginx"

r := gin.New()

// Order matters:
//   CORS first so preflights are answered without logs or spans;
//   RequestID before tracing so the id lands on the server span;
//   AccessLog inside the span so records carry trace_id/span_id;
//   Recover innermost so a panic still yields a 500 span and access record.
r.Use(
    ginx.CORS(ginx.CORSConfig{
        AllowedOrigins: []string{"https://app.example.com"},
        AllowedMethods: []string{"GET", "POST"},
        ExposedHeaders: []string{ginx.HeaderRequestID},
    }),
    ginx.RequestID(),
    tracing.Middleware(tracing.MiddlewareConfig{ServiceName: "api", SkipPaths: []string{"/health"}}),
    ginx.AccessLog(ginx.AccessLogConfig{
        SkipPaths: []string{"/health"},
    }),
    ginx.Recover(),
)

r.Use(ginx.NoCache(), ginx.Secure(ginx.SecureConfig{
    HSTS: "max-age=31536000; includeSubDomains; preload",
}))
```

## Middleware reference

| Middleware    | Purpose                                                           |
|---------------|-------------------------------------------------------------------|
| `Recover()`   | Catches panics, logs full stack, marks the span failed, responds 500 |
| `RequestID()` | Reads (validated) or generates `X-Request-ID`; stores it in ctx for logs and spans |
| `AccessLog()` | One slog record per request; 5xx ERROR / 4xx WARN; never logs the query string |
| `CORS()`      | Allowlist origins / methods / headers; `["*"]` for any            |
| `NoCache()`   | Sets `Cache-Control: no-cache, no-store...` on dynamic responses  |
| `Secure()`    | `Strict-Transport-Security` + optional `Content-Security-Policy`  |

## Reading the request id from a handler

```go
func handler(c *gin.Context) {
    rid := ginx.RequestIDFromContext(c)
    slog.InfoContext(c.Request.Context(), "doing work", "request_id", rid)
}
```

You rarely need to pass it by hand: `RequestID()` stores the id under
`foundation/slog.RequestIDKey`, so with `foundation/slog` every
`slog.InfoContext(c.Request.Context(), ...)` record already carries
`request_id`, `trace_id` and `span_id`.

## Access-log schema

`component=http`, `operation=request`, `outcome`, `method`, `path`, `route`,
`status_code`, `duration_ms`, `client_ip`, `user_agent`, `request_size`,
`response_size`, `request_id`, `trace_id`, `span_id`, `errors` (when set).

## Testing

```bash
go test -race ./...
```

## Changelog

See [CHANGELOG.md](./CHANGELOG.md).
