package port

import (
	"context"

	"github.com/brizenchi/go-modules/billing/domain"
)

// BillingEventRepository persists webhook events for audit and idempotency.
type BillingEventRepository interface {
	// CreateIfAbsent inserts the event. If a row with the same
	// (provider, provider_event_id) already exists, the existing row is
	// returned with no error; the bool indicates whether the caller
	// inserted (true) or found an existing row (false).
	CreateIfAbsent(ctx context.Context, e *domain.BillingEvent) (*domain.BillingEvent, bool, error)

	// MarkProcessed marks an event as fully handled.
	MarkProcessed(ctx context.Context, provider, providerEventID string) error
}

// UserResolver maps webhook payload hints to a userID known to the host
// application. The billing module is intentionally agnostic to the host's
// user model: implementers do whatever lookup they need.
type UserResolver interface {
	Resolve(ctx context.Context, hint UserHint) (userID string, err error)
}
