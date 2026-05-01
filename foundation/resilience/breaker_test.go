package resilience

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// fakeClock returns a deterministic clock controlled by the test.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func TestBreaker_TripsAfterThreshold(t *testing.T) {
	clk := newFakeClock()
	b := NewBreaker(BreakerConfig{
		FailureThreshold: 3,
		OpenDuration:     10 * time.Second,
		Now:              clk.Now,
	})

	boom := errors.New("boom")
	for i := 0; i < 3; i++ {
		_ = b.Do(context.Background(), func(ctx context.Context) error { return boom })
	}
	if got := b.State(); got != "open" {
		t.Fatalf("state = %s, want open after 3 failures", got)
	}

	err := b.Do(context.Background(), func(ctx context.Context) error { return nil })
	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestBreaker_HalfOpenSuccessCloses(t *testing.T) {
	clk := newFakeClock()
	b := NewBreaker(BreakerConfig{
		FailureThreshold: 2,
		OpenDuration:     5 * time.Second,
		Now:              clk.Now,
	})

	boom := errors.New("boom")
	for i := 0; i < 2; i++ {
		_ = b.Do(context.Background(), func(ctx context.Context) error { return boom })
	}
	if b.State() != "open" {
		t.Fatal("expected open")
	}

	clk.Advance(6 * time.Second)
	// First call after timeout transitions to half-open and runs the probe.
	err := b.Do(context.Background(), func(ctx context.Context) error { return nil })
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if got := b.State(); got != "closed" {
		t.Errorf("state = %s, want closed after successful probe", got)
	}
}

func TestBreaker_HalfOpenFailureReopens(t *testing.T) {
	clk := newFakeClock()
	b := NewBreaker(BreakerConfig{
		FailureThreshold: 2,
		OpenDuration:     5 * time.Second,
		Now:              clk.Now,
	})

	boom := errors.New("boom")
	for i := 0; i < 2; i++ {
		_ = b.Do(context.Background(), func(ctx context.Context) error { return boom })
	}
	clk.Advance(6 * time.Second)

	// Probe fails → reopen.
	_ = b.Do(context.Background(), func(ctx context.Context) error { return boom })
	if got := b.State(); got != "open" {
		t.Errorf("state = %s, want open after probe failure", got)
	}
}

func TestBreaker_IsFailureFilters(t *testing.T) {
	clk := newFakeClock()
	userErr := errors.New("user input bad")
	b := NewBreaker(BreakerConfig{
		FailureThreshold: 2,
		OpenDuration:     5 * time.Second,
		Now:              clk.Now,
		IsFailure:        func(err error) bool { return !errors.Is(err, userErr) },
	})

	for i := 0; i < 5; i++ {
		_ = b.Do(context.Background(), func(ctx context.Context) error { return userErr })
	}
	if got := b.State(); got != "closed" {
		t.Errorf("state = %s, want closed (user errors don't trip)", got)
	}
}

func TestBreaker_DefaultsApplied(t *testing.T) {
	b := NewBreaker(BreakerConfig{})
	if b.cfg.FailureThreshold != 5 || b.cfg.OpenDuration != 30*time.Second || b.cfg.HalfOpenMax != 1 {
		t.Errorf("defaults not applied: %+v", b.cfg)
	}
}
