package port

import (
	"context"

	"github.com/brizenchi/go-modules/modules/referral/event"
)

// Listener handles a single referral domain event. Listeners must be
// idempotent because a failed delivery can be retried.
type Listener func(ctx context.Context, env event.Envelope) error

// EventBus dispatches referral domain events to subscribers.
type EventBus interface {
	Subscribe(kind event.Kind, fn Listener)
	// Publish runs every matching listener and returns their aggregate error.
	// Implementations must not stop dispatching sibling listeners after one
	// fails; callers use the returned error to arrange an at-least-once retry.
	Publish(ctx context.Context, env event.Envelope) error
}
