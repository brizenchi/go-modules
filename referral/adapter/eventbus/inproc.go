// Package eventbus provides in-process EventBus implementations for the referral module.
package eventbus

import (
	"context"
	"log/slog"
	"sync"

	"github.com/brizenchi/go-modules/referral/event"
	"github.com/brizenchi/go-modules/referral/port"
)

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
			slog.Error("referral: listener panic", "kind", env.Kind, "recover", r)
		}
	}()
	if err := fn(ctx, env); err != nil {
		slog.Error("referral: listener returned error", "kind", env.Kind, "error", err)
	}
}

var _ port.EventBus = (*InProc)(nil)
