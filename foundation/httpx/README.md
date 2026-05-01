# foundation/httpx

> Configurable `http.Client` builder with retry, circuit-breaker, default headers, and timeout middleware.

[![Go Reference](https://pkg.go.dev/badge/github.com/brizenchi/go-modules/foundation/httpx.svg)](https://pkg.go.dev/github.com/brizenchi/go-modules/foundation/httpx)

This module is for outbound HTTP. It wraps a `http.RoundTripper` chain around
the standard library client so callers can get sane defaults without rewriting
retry and breaker glue in every service.

Under the hood it integrates [`foundation/resilience`](../resilience/).

## Install

```bash
go get github.com/brizenchi/go-modules/foundation/httpx
```

## Quick start

```go
import (
    "time"

    "github.com/brizenchi/go-modules/foundation/httpx"
    "github.com/brizenchi/go-modules/foundation/resilience"
)

retry := resilience.Exponential(4, 100*time.Millisecond)
breaker := resilience.NewBreaker(resilience.BreakerConfig{
    FailureThreshold: 5,
    OpenDuration:     30 * time.Second,
})

client := httpx.NewClient(httpx.Config{
    Timeout: 10 * time.Second,
    Retry:   &retry,
    Breaker: breaker,
    Headers: map[string]string{
        "User-Agent": "my-service/1.0",
    },
})
```

## Middleware order

Outermost to innermost:

- default headers
- circuit breaker
- retry
- underlying transport

Headers are re-applied on every retry attempt. The breaker wraps retry so an
open circuit short-circuits the entire call immediately.

## Retry semantics

- Retries `408`, `425`, `429`, and all `5xx` responses.
- Retries transport errors unless the request context is already canceled or timed out.
- Buffers the request body once in memory so it can be replayed on retry.
- Returns the final HTTP response when all retry attempts still end on a retryable status.

If you need custom retry cadence, pass a tailored `resilience.Policy`.

## Transport defaults

`DefaultTransport()` clones `http.DefaultTransport` and tunes the common pool
settings for service-to-service traffic:

- `MaxIdleConns = 100`
- `MaxIdleConnsPerHost = 10`
- `IdleConnTimeout = 90s`
- `TLSHandshakeTimeout = 10s`
- `ExpectContinueTimeout = 1s`

## Testing

```bash
go test -race ./...
```

## Changelog

See [CHANGELOG.md](./CHANGELOG.md).
