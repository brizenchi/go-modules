package eventbus

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/brizenchi/go-modules/referral/event"
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
	bus.Publish(context.Background(), event.Envelope{Kind: event.KindReferralRegistered})
	bus.Publish(context.Background(), event.Envelope{Kind: event.KindReferralActivated})
	bus.Publish(context.Background(), event.Envelope{Kind: event.KindReferralActivated})
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
	bus.Publish(context.Background(), event.Envelope{Kind: event.KindReferralActivated})
	if ran != 1 {
		t.Errorf("post-panic listener ran %d times", ran)
	}
}
