package eventbus

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/brizenchi/go-modules/modules/referral/event"
)

func TestInProc_Dispatch(t *testing.T) {
	bus := NewInProc()
	var reg, act int32
	bus.Subscribe(event.KindReferralRegistered, func(_ context.Context, _ event.Envelope) error {
		atomic.AddInt32(&reg, 1)
		return nil
	})
	bus.Subscribe(event.KindReferralActivated, func(_ context.Context, _ event.Envelope) error {
		atomic.AddInt32(&act, 1)
		return nil
	})
	if err := bus.Publish(context.Background(), event.Envelope{Kind: event.KindReferralRegistered}); err != nil {
		t.Fatal(err)
	}
	if err := bus.Publish(context.Background(), event.Envelope{Kind: event.KindReferralActivated}); err != nil {
		t.Fatal(err)
	}
	if err := bus.Publish(context.Background(), event.Envelope{Kind: event.KindReferralActivated}); err != nil {
		t.Fatal(err)
	}
	if reg != 1 || act != 2 {
		t.Errorf("counts (%d,%d) want (1,2)", reg, act)
	}
}

func TestInProc_PanicRecovery(t *testing.T) {
	bus := NewInProc()
	var ran int32
	bus.Subscribe(event.KindReferralActivated, func(_ context.Context, _ event.Envelope) error {
		panic("x")
	})
	bus.Subscribe(event.KindReferralActivated, func(_ context.Context, _ event.Envelope) error {
		atomic.AddInt32(&ran, 1)
		return nil
	})
	err := bus.Publish(context.Background(), event.Envelope{Kind: event.KindReferralActivated})
	if err == nil {
		t.Fatal("expected recovered listener panic to be returned")
	}
	if ran != 1 {
		t.Errorf("post-panic listener ran %d times", ran)
	}
}

func TestInProc_ReturnsListenerErrorsAfterRunningSiblings(t *testing.T) {
	bus := NewInProc()
	want := errors.New("reward unavailable")
	var ran int32
	bus.Subscribe(event.KindReferralActivated, func(_ context.Context, _ event.Envelope) error {
		return want
	})
	bus.Subscribe(event.KindReferralActivated, func(_ context.Context, _ event.Envelope) error {
		atomic.AddInt32(&ran, 1)
		return nil
	})

	err := bus.Publish(context.Background(), event.Envelope{Kind: event.KindReferralActivated})
	if !errors.Is(err, want) {
		t.Fatalf("error=%v, want wrapped listener error", err)
	}
	if ran != 1 {
		t.Fatalf("successful sibling ran %d times, want 1", ran)
	}
}
