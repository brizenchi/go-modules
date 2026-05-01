// Package resilience provides retry/backoff and circuit-breaker
// primitives that are transport-agnostic. They can wrap any function:
// HTTP calls, SDK calls (stripe-go, redis), DB writes, etc.
//
// HTTP users typically don't reach for this package directly — use
// foundation/httpx, which has these primitives built into the
// RoundTripper chain. Reach for resilience when wrapping a non-HTTP
// dependency.
//
// Two primary entry points:
//
//   - Do(ctx, fn, Policy)   retry with backoff
//   - Breaker.Do(ctx, fn)   circuit-break failing dependencies
//
// They compose: a Breaker wrapping a Do gives "retry, but stop trying
// when the dependency is clearly down."
package resilience

import (
	"context"
	"errors"
	"math/rand/v2"
	"time"
)

// ErrInvalidPolicy is returned by Do when the supplied Policy is empty
// or has no Backoff.
var ErrInvalidPolicy = errors.New("resilience: invalid policy")

// Backoff returns the wait time before the *next* attempt. `attempt`
// is 1-indexed and counts the number of retries already performed
// (so attempt=1 means "we tried once and failed; how long before we
// try again?").
type Backoff interface {
	Next(attempt int) time.Duration
}

// Policy describes how Do should retry.
type Policy struct {
	// MaxAttempts is the total number of attempts including the first.
	// 1 = try once, no retry. <=0 is treated as 1.
	MaxAttempts int

	// Backoff returns the delay before each retry. Required.
	Backoff Backoff

	// Retryable inspects an error and returns true if it should be
	// retried. nil means "retry all non-nil errors".
	Retryable func(error) bool
}

// Do invokes fn with retry. fn must respect ctx — its current attempt
// is cancelled when ctx is. The returned error is the last attempt's
// error, or ctx.Err() if the context was cancelled mid-wait.
//
// Do does NOT mutate fn's input or output beyond invoking it. Callers
// passing mutable state (e.g. http.Request bodies) are responsible
// for replaying it.
func Do(ctx context.Context, fn func(context.Context) error, p Policy) error {
	if p.Backoff == nil {
		return ErrInvalidPolicy
	}
	attempts := p.MaxAttempts
	if attempts <= 0 {
		attempts = 1
	}

	var lastErr error
	for n := 1; n <= attempts; n++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := fn(ctx)
		if err == nil {
			return nil
		}
		lastErr = err
		if p.Retryable != nil && !p.Retryable(err) {
			return err
		}
		if n == attempts {
			break
		}
		wait := p.Backoff.Next(n)
		if wait <= 0 {
			continue
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return lastErr
}

// Constant returns a Policy that waits a fixed interval between attempts.
func Constant(attempts int, interval time.Duration) Policy {
	return Policy{
		MaxAttempts: attempts,
		Backoff:     constantBackoff{interval: interval},
	}
}

// Exponential returns a Policy that doubles the wait each attempt
// starting from `base`, capped at 30s, with ±20% jitter to spread
// retry storms across many concurrent callers.
func Exponential(attempts int, base time.Duration) Policy {
	return Policy{
		MaxAttempts: attempts,
		Backoff: exponentialBackoff{
			base:      base,
			max:       30 * time.Second,
			jitterPct: 0.2,
		},
	}
}

type constantBackoff struct {
	interval time.Duration
}

func (c constantBackoff) Next(int) time.Duration { return c.interval }

type exponentialBackoff struct {
	base      time.Duration
	max       time.Duration
	jitterPct float64 // 0..1; 0 disables jitter
}

func (e exponentialBackoff) Next(attempt int) time.Duration {
	if e.base <= 0 {
		return 0
	}
	// 1<<n grows fast; cap before overflowing.
	shift := attempt - 1
	if shift < 0 {
		shift = 0
	}
	if shift > 30 {
		shift = 30
	}
	d := e.base << shift
	if d <= 0 || d > e.max {
		d = e.max
	}
	if e.jitterPct > 0 {
		delta := float64(d) * e.jitterPct
		// rand.Float64() in [0,1); scale to [-delta, +delta].
		d = time.Duration(float64(d) - delta + rand.Float64()*delta*2) //nolint:gosec // math/rand for jitter is fine
		if d < 0 {
			d = 0
		}
	}
	return d
}
