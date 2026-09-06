// Package eventbus provides in-process EventBus implementations for the referral module.
package eventbus

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/brizenchi/go-modules/modules/referral/event"
	"github.com/brizenchi/go-modules/modules/referral/port"
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
			err = fmt.Errorf("referral listener panic: %v", r)
			slog.ErrorContext(ctx, "referral: listener panic", "kind", env.Kind, "recover", r)
		}
	}()
	if err = fn(ctx, env); err != nil {
		slog.ErrorContext(ctx, "referral: listener returned error", "kind", env.Kind, "error", err)
	}
	return err
}

var _ port.EventBus = (*InProc)(nil)
