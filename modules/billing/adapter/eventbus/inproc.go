// Package eventbus provides in-process EventBus implementations.
package eventbus

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/brizenchi/go-modules/modules/billing/event"
	"github.com/brizenchi/go-modules/modules/billing/port"
)

// InProc is a synchronous, in-process EventBus.
//
// Listeners run sequentially in subscription order on the caller's
// goroutine. A panicking listener is recovered and logged. A returning
// error is logged but does not stop sibling listeners. Publish returns the
// aggregate failure after every sibling has had a chance to run.
//
// For at-least-once delivery across processes or async dispatch, supply
// a different EventBus implementation (e.g. backed by a queue).
type InProc struct {
	mu        sync.RWMutex
	listeners map[event.Kind][]port.Listener
	wildcards []port.Listener
}

func NewInProc() *InProc {
	return &InProc{listeners: make(map[event.Kind][]port.Listener)}
}

func (b *InProc) Subscribe(kind event.Kind, fn port.Listener) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if kind == "" {
		b.wildcards = append(b.wildcards, fn)
		return
	}
	b.listeners[kind] = append(b.listeners[kind], fn)
}

func (b *InProc) Publish(ctx context.Context, env event.Envelope) error {
	b.mu.RLock()
	listeners := append([]port.Listener(nil), b.listeners[env.Kind]...)
	wildcards := append([]port.Listener(nil), b.wildcards...)
	b.mu.RUnlock()

	var failures []error
	for _, fn := range append(listeners, wildcards...) {
		if err := b.run(ctx, env, fn); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

func (b *InProc) run(ctx context.Context, env event.Envelope, fn port.Listener) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("billing listener panic: %v", r)
			slog.ErrorContext(ctx, "billing: listener panic",
				"kind", env.Kind,
				"event_id", env.ProviderEventID,
				"user_id", env.UserID,
				"recover", r,
			)
		}
	}()
	if err = fn(ctx, env); err != nil {
		slog.ErrorContext(ctx, "billing: listener returned error",
			"kind", env.Kind,
			"event_id", env.ProviderEventID,
			"user_id", env.UserID,
			"error", err,
		)
	}
	return err
}

var _ port.EventBus = (*InProc)(nil)
