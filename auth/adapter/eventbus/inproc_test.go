package eventbus

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/brizenchi/go-modules/auth/event"
)

func TestInProc_DispatchByKind(t *testing.T) {
	bus := NewInProc()
	var su, in int32
	bus.Subscribe(event.KindUserSignedUp, func(ctx context.Context, e event.Envelope) error {
		atomic.AddInt32(&su, 1)
		return nil
	})
	bus.Subscribe(event.KindUserLoggedIn, func(ctx context.Context, e event.Envelope) error {
		atomic.AddInt32(&in, 1)
		return nil
	})
	ctx := context.Background()
	bus.Publish(ctx, event.Envelope{Kind: event.KindUserSignedUp})
	bus.Publish(ctx, event.Envelope{Kind: event.KindUserLoggedIn})
	bus.Publish(ctx, event.Envelope{Kind: event.KindUserLoggedIn})
	if su != 1 || in != 2 {
		t.Errorf("counts = (%d,%d), want (1,2)", su, in)
	}
}

func TestInProc_PanicRecovery(t *testing.T) {
	bus := NewInProc()
	var ran int32
	bus.Subscribe(event.KindUserSignedUp, func(ctx context.Context, e event.Envelope) error {
		panic("boom")
	})
	bus.Subscribe(event.KindUserSignedUp, func(ctx context.Context, e event.Envelope) error {
		atomic.AddInt32(&ran, 1)
		return nil
	})
	bus.Publish(context.Background(), event.Envelope{Kind: event.KindUserSignedUp})
	if ran != 1 {
		t.Errorf("post-panic listener ran %d times, want 1", ran)
	}
}
