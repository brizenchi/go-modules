# foundation/resilience

> Transport-agnostic retry/backoff and circuit-breaker primitives for external dependencies.

[![Go Reference](https://pkg.go.dev/badge/github.com/brizenchi/go-modules/foundation/resilience.svg)](https://pkg.go.dev/github.com/brizenchi/go-modules/foundation/resilience)

Use this module when the thing you are calling is not plain HTTP: Stripe SDK,
Redis ops, DB writes, queue publishes, or any other function you want to wrap
with retry/backoff and circuit-breaking.

HTTP callers will usually prefer [`foundation/httpx`](../httpx/), which wires
the same primitives into a `http.Client`.

## Install

```bash
go get github.com/brizenchi/go-modules/foundation/resilience
```

## Quick start

```go
import (
    "context"
    "time"

    "github.com/brizenchi/go-modules/foundation/resilience"
)

err := resilience.Do(context.Background(), func(ctx context.Context) error {
    return callStripe(ctx)
}, resilience.Exponential(4, 100*time.Millisecond))
```

## Retry API

- `Do(ctx, fn, policy)` retries a function until it succeeds, the policy is exhausted, or the context is canceled.
- `Constant(attempts, interval)` retries on a fixed interval.
- `Exponential(attempts, base)` doubles the wait each retry, caps at 30s, and adds jitter to avoid retry storms.

`Policy.Retryable` lets callers stop retrying on terminal errors such as
validation failures or permanent upstream rejections.

## Circuit breaker API

```go
breaker := resilience.NewBreaker(resilience.BreakerConfig{
    FailureThreshold: 5,
    OpenDuration:     30 * time.Second,
})

err := breaker.Do(context.Background(), func(ctx context.Context) error {
    return callRedis(ctx)
})
```

State machine:

- `closed` -> normal traffic
- `open` -> fail fast with `ErrCircuitOpen`
- `half-open` -> allow a small number of probe requests before closing again

Use `BreakerConfig.IsFailure` when only some errors should count toward the
failure threshold.

## Testing

```bash
go test -race ./...
```

## Changelog

See [CHANGELOG.md](./CHANGELOG.md).
