// Package eventbus provides in-process EventBus implementations for the auth module.
package eventbus

import (
	"context"
	"log/slog"
	"sync"

	"github.com/brizenchi/go-modules/modules/auth/event"
	"github.com/brizenchi/go-modules/modules/auth/port"
)

// InProc is a synchronous, in-process EventBus.
//
// Listeners run sequentially in subscription order. Panics are recovered
// and logged. Errors are logged but do not stop sibling listeners. Use a
// queue-backed bus for cross-process at-least-once delivery.
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

func (b *InProc) Publish(ctx context.Context, env event.Envelope) {
	b.mu.RLock()
	listeners := append([]port.Listener(nil), b.listeners[env.Kind]...)
	wildcards := append([]port.Listener(nil), b.wildcards...)
	b.mu.RUnlock()

	for _, fn := range append(listeners, wildcards...) {
		b.run(ctx, env, fn)
	}
}

func (b *InProc) run(ctx context.Context, env event.Envelope, fn port.Listener) {
	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "auth: listener panic", "kind", env.Kind, "user_id", env.UserID, "recover", r)
		}
	}()
	if err := fn(ctx, env); err != nil {
		slog.ErrorContext(ctx, "auth: listener returned error", "kind", env.Kind, "user_id", env.UserID, "error", err)
	}
}

var _ port.EventBus = (*InProc)(nil)
