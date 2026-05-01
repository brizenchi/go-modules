package port

import (
	"context"

	"github.com/brizenchi/go-modules/modules/billing/event"
)

// Listener handles a single billing domain event.
//
// Returning an error from a listener does NOT prevent other listeners from
// running, but it is logged. Listeners should be idempotent because a
// webhook may be delivered more than once before the idempotency check
// catches it (race window).
type Listener func(ctx context.Context, env event.Envelope) error

// EventBus dispatches domain events to subscribers.
type EventBus interface {
	// Subscribe registers a listener for a single event kind.
	// The empty Kind ("") subscribes to all events.
	Subscribe(kind event.Kind, listener Listener)

	// Publish dispatches an event to all matching listeners.
	// Implementations may run synchronously or asynchronously; either way,
	// Publish must not block the caller longer than necessary.
	Publish(ctx context.Context, env event.Envelope)
}
