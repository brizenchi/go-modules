package eventbus

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/brizenchi/go-modules/modules/billing/event"
)

func TestInProc_TopicDispatch(t *testing.T) {
	bus := NewInProc()
	var activatedCount, renewedCount int32
	bus.Subscribe(event.KindSubscriptionActivated, func(ctx context.Context, e event.Envelope) error {
		atomic.AddInt32(&activatedCount, 1)
		return nil
	})
	bus.Subscribe(event.KindSubscriptionRenewed, func(ctx context.Context, e event.Envelope) error {
		atomic.AddInt32(&renewedCount, 1)
		return nil
	})

	bus.Publish(context.Background(), event.Envelope{Kind: event.KindSubscriptionActivated})
	bus.Publish(context.Background(), event.Envelope{Kind: event.KindSubscriptionActivated})
	bus.Publish(context.Background(), event.Envelope{Kind: event.KindSubscriptionRenewed})

	if got := atomic.LoadInt32(&activatedCount); got != 2 {
		t.Errorf("activated count = %d, want 2", got)
	}
	if got := atomic.LoadInt32(&renewedCount); got != 1 {
		t.Errorf("renewed count = %d, want 1", got)
	}
}

func TestInProc_Wildcard(t *testing.T) {
	bus := NewInProc()
	var seen []event.Kind
	bus.Subscribe("", func(ctx context.Context, e event.Envelope) error {
		seen = append(seen, e.Kind)
		return nil
	})
	bus.Publish(context.Background(), event.Envelope{Kind: event.KindSubscriptionActivated})
	bus.Publish(context.Background(), event.Envelope{Kind: event.KindCreditsPurchased})

	if len(seen) != 2 {
		t.Fatalf("wildcard saw %d events, want 2", len(seen))
	}
}

func TestInProc_PanicRecovery(t *testing.T) {
	bus := NewInProc()
	var ranAfter int32
	bus.Subscribe(event.KindSubscriptionActivated, func(ctx context.Context, e event.Envelope) error {
		panic("boom")
	})
	bus.Subscribe(event.KindSubscriptionActivated, func(ctx context.Context, e event.Envelope) error {
		atomic.AddInt32(&ranAfter, 1)
		return nil
	})
	// Must not panic out of Publish.
	bus.Publish(context.Background(), event.Envelope{Kind: event.KindSubscriptionActivated})
	if got := atomic.LoadInt32(&ranAfter); got != 1 {
		t.Errorf("listener after panic ran %d times, want 1", got)
	}
}

func TestInProc_ListenerErrorDoesNotStopOthers(t *testing.T) {
	bus := NewInProc()
	var ran int32
	bus.Subscribe(event.KindSubscriptionActivated, func(ctx context.Context, e event.Envelope) error {
		return errors.New("first failed")
	})
	bus.Subscribe(event.KindSubscriptionActivated, func(ctx context.Context, e event.Envelope) error {
		atomic.AddInt32(&ran, 1)
		return nil
	})
	bus.Publish(context.Background(), event.Envelope{Kind: event.KindSubscriptionActivated})
	if got := atomic.LoadInt32(&ran); got != 1 {
		t.Errorf("second listener ran %d times, want 1", got)
	}
}

func TestInProc_ConcurrentSubscribeAndPublish(t *testing.T) {
	// Race-detector smoke test: many goroutines subscribing and publishing.
	bus := NewInProc()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			bus.Subscribe(event.KindSubscriptionActivated, func(ctx context.Context, e event.Envelope) error { return nil })
		}()
		go func() {
			defer wg.Done()
			bus.Publish(context.Background(), event.Envelope{Kind: event.KindSubscriptionActivated})
		}()
	}
	wg.Wait()
}
