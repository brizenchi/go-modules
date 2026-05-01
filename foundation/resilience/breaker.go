package resilience

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrCircuitOpen is returned by Breaker.Do when the breaker is open
// or has reached its half-open concurrency limit.
var ErrCircuitOpen = errors.New("resilience: circuit breaker is open")

// BreakerConfig configures a Breaker.
type BreakerConfig struct {
	// FailureThreshold is the number of consecutive failures in the
	// closed state that trips the breaker. Default 5.
	FailureThreshold int

	// OpenDuration is how long the breaker stays open before
	// transitioning to half-open. Default 30s.
	OpenDuration time.Duration

	// HalfOpenMax is the maximum number of concurrent probe calls
	// allowed in the half-open state. Default 1.
	HalfOpenMax int

	// IsFailure reports whether a returned error counts as a failure
	// for breaker accounting. nil means "any non-nil error is a failure".
	// Use this to avoid tripping on user-input errors (e.g. HTTP 4xx).
	IsFailure func(error) bool

	// Now is a clock injection point for tests. Defaults to time.Now.
	Now func() time.Time
}

// Breaker is a half-open circuit breaker.
//
// State machine:
//
//	closed     --N consecutive failures-->     open
//	open       --OpenDuration elapsed-->       half-open
//	half-open  --probe success-->              closed
//	half-open  --probe failure-->              open
//
// Concurrency-safe.
type Breaker struct {
	cfg BreakerConfig

	mu               sync.Mutex
	state            breakerState
	consecFailures   int
	openUntil        time.Time
	halfOpenInflight int
}

type breakerState int

const (
	stateClosed breakerState = iota
	stateOpen
	stateHalfOpen
)

// NewBreaker constructs a breaker. Zero-value fields take defaults.
func NewBreaker(cfg BreakerConfig) *Breaker {
	if cfg.FailureThreshold <= 0 {
		cfg.FailureThreshold = 5
	}
	if cfg.OpenDuration <= 0 {
		cfg.OpenDuration = 30 * time.Second
	}
	if cfg.HalfOpenMax <= 0 {
		cfg.HalfOpenMax = 1
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Breaker{cfg: cfg}
}

// Do runs fn through the breaker.
//
// Returns ErrCircuitOpen immediately when blocked. Otherwise returns
// fn's error (the breaker's state update happens regardless).
func (b *Breaker) Do(ctx context.Context, fn func(context.Context) error) error {
	if err := b.beforeCall(); err != nil {
		return err
	}
	err := fn(ctx)
	b.afterCall(err)
	return err
}

// State returns the current state for observability.
// Returns "closed", "open", or "half-open".
func (b *Breaker) State() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.state.String()
}

func (s breakerState) String() string {
	switch s {
	case stateClosed:
		return "closed"
	case stateOpen:
		return "open"
	case stateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

func (b *Breaker) beforeCall() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case stateClosed:
		return nil
	case stateOpen:
		if !b.cfg.Now().Before(b.openUntil) {
			b.state = stateHalfOpen
			b.halfOpenInflight = 1
			return nil
		}
		return ErrCircuitOpen
	case stateHalfOpen:
		if b.halfOpenInflight >= b.cfg.HalfOpenMax {
			return ErrCircuitOpen
		}
		b.halfOpenInflight++
		return nil
	}
	return nil
}

func (b *Breaker) afterCall(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	failed := err != nil
	if failed && b.cfg.IsFailure != nil {
		failed = b.cfg.IsFailure(err)
	}

	if b.state == stateHalfOpen && b.halfOpenInflight > 0 {
		b.halfOpenInflight--
	}

	if failed {
		b.consecFailures++
		switch b.state {
		case stateClosed:
			if b.consecFailures >= b.cfg.FailureThreshold {
				b.state = stateOpen
				b.openUntil = b.cfg.Now().Add(b.cfg.OpenDuration)
			}
		case stateHalfOpen:
			b.state = stateOpen
			b.openUntil = b.cfg.Now().Add(b.cfg.OpenDuration)
		}
		return
	}

	// Success.
	b.consecFailures = 0
	if b.state == stateHalfOpen {
		b.state = stateClosed
	}
}
