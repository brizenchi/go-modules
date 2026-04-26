package port

import (
	"context"

	"github.com/brizenchi/go-modules/referral/event"
)

// Listener handles a single referral domain event. Errors are logged
// and do not block sibling listeners. Listeners must be idempotent.
type Listener func(ctx context.Context, env event.Envelope) error

// EventBus dispatches referral domain events to subscribers.
type EventBus interface {
	Subscribe(kind event.Kind, fn Listener)
	Publish(ctx context.Context, env event.Envelope)
}
