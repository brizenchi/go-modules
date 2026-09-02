package port

import (
	"context"

	"github.com/brizenchi/go-modules/modules/billing/event"
)

// Listener handles a single billing domain event.
//
// Returning an error from a listener does NOT prevent other listeners from
// running. Publish aggregates those errors so a webhook caller can leave the
// provider event unprocessed and let the provider retry it. Listeners must be
// idempotent because earlier listeners may run again on that retry.
type Listener func(ctx context.Context, env event.Envelope) error

// EventBus dispatches domain events to subscribers.
type EventBus interface {
	// Subscribe registers a listener for a single event kind.
	// The empty Kind ("") subscribes to all events.
	Subscribe(kind event.Kind, listener Listener)

	// Publish dispatches an event to all matching listeners.
	// Implementations may run synchronously or asynchronously. An error means
	// one or more listeners did not complete successfully.
	Publish(ctx context.Context, env event.Envelope) error
}
