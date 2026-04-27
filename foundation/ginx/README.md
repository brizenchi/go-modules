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

// Order matters: Recover first, RequestID before AccessLog so the id lands
// in the access record.
r.Use(
    ginx.Recover(),
    ginx.RequestID(),
    ginx.AccessLog(ginx.AccessLogConfig{
        SkipPaths: []string{"/health"},
    }),
)

r.Use(ginx.CORS(ginx.CORSConfig{
    AllowedOrigins: []string{"https://app.example.com"},
    AllowedMethods: []string{"GET", "POST"},
}))

r.Use(ginx.NoCache(), ginx.Secure(ginx.SecureConfig{
    HSTS: "max-age=31536000; includeSubDomains; preload",
}))
```

## Middleware reference

| Middleware    | Purpose                                                           |
|---------------|-------------------------------------------------------------------|
| `Recover()`   | Catches panics, logs stack via slog, responds 500 envelope        |
| `RequestID()` | Generates / reads `X-Request-ID`, makes it available to handlers  |
| `AccessLog()` | Structured slog record per request: method, path, status, dur    |
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

The `RequestID()` middleware also calls `c.Set(...)`, so any logger that
pulls from the Gin context (e.g. `foundation/slog.Ctx`) picks it up
automatically.

## Testing

```bash
go test -race ./...
```

## Changelog

See [CHANGELOG.md](./CHANGELOG.md).
