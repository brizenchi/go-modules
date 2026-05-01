package resilience

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDo_SuccessFirstAttempt(t *testing.T) {
	calls := 0
	err := Do(context.Background(), func(ctx context.Context) error {
		calls++
		return nil
	}, Constant(3, time.Millisecond))
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1", calls)
	}
}

func TestDo_SuccessOnRetry(t *testing.T) {
	calls := 0
	err := Do(context.Background(), func(ctx context.Context) error {
		calls++
		if calls < 3 {
			return errors.New("transient")
		}
		return nil
	}, Constant(5, time.Millisecond))
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if calls != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}
}

func TestDo_ExhaustsRetries(t *testing.T) {
	calls := 0
	want := errors.New("permanent")
	err := Do(context.Background(), func(ctx context.Context) error {
		calls++
		return want
	}, Constant(3, time.Millisecond))
	if !errors.Is(err, want) {
		t.Errorf("err = %v, want %v", err, want)
	}
	if calls != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}
}

func TestDo_RespectsRetryablePredicate(t *testing.T) {
	calls := 0
	terminal := errors.New("don't retry me")
	p := Policy{
		MaxAttempts: 5,
		Backoff:     constantBackoff{interval: time.Millisecond},
		Retryable:   func(err error) bool { return !errors.Is(err, terminal) },
	}
	err := Do(context.Background(), func(ctx context.Context) error {
		calls++
		return terminal
	}, p)
	if !errors.Is(err, terminal) {
		t.Errorf("err = %v", err)
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1 (Retryable should stop further attempts)", calls)
	}
}

func TestDo_CtxCancelDuringWait(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()
	calls := 0
	err := Do(ctx, func(ctx context.Context) error {
		calls++
		return errors.New("retry me")
	}, Constant(10, 100*time.Millisecond))
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
	if calls > 2 {
		t.Errorf("calls = %d, expected to stop early", calls)
	}
}

func TestDo_CtxCancelledBeforeStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	err := Do(ctx, func(ctx context.Context) error {
		calls++
		return nil
	}, Constant(3, time.Millisecond))
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v", err)
	}
	if calls != 0 {
		t.Errorf("calls = %d, want 0", calls)
	}
}

func TestDo_InvalidPolicy(t *testing.T) {
	err := Do(context.Background(), func(ctx context.Context) error { return nil }, Policy{})
	if !errors.Is(err, ErrInvalidPolicy) {
		t.Errorf("err = %v, want ErrInvalidPolicy", err)
	}
}

func TestExponentialBackoff_Growth(t *testing.T) {
	b := exponentialBackoff{base: 100 * time.Millisecond, max: 30 * time.Second}
	for n := 1; n <= 5; n++ {
		got := b.Next(n)
		want := time.Duration(100<<(n-1)) * time.Millisecond
		if got != want {
			t.Errorf("Next(%d) = %v, want %v", n, got, want)
		}
	}
}

func TestExponentialBackoff_CapsAtMax(t *testing.T) {
	b := exponentialBackoff{base: time.Second, max: 5 * time.Second}
	if got := b.Next(20); got != 5*time.Second {
		t.Errorf("Next(20) = %v, want 5s (capped)", got)
	}
}

func TestExponentialBackoff_ZeroBase(t *testing.T) {
	b := exponentialBackoff{base: 0, max: 5 * time.Second}
	if got := b.Next(1); got != 0 {
		t.Errorf("Next(1) = %v, want 0", got)
	}
}

func TestExponentialBackoff_Jitter(t *testing.T) {
	b := exponentialBackoff{base: time.Second, max: 30 * time.Second, jitterPct: 0.2}
	// With 20% jitter on a 1s base, returned value must lie in [0.8s, 1.2s].
	for i := 0; i < 50; i++ {
		got := b.Next(1)
		if got < 800*time.Millisecond || got > 1200*time.Millisecond {
			t.Fatalf("jittered value out of range: %v", got)
		}
	}
}
