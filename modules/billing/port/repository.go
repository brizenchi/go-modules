package port

import (
	"context"
	"time"

	"github.com/brizenchi/go-modules/modules/billing/domain"
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

// SubscriptionRepository persists the current provider-derived subscription
// snapshot used by query, change, cancellation and portal flows.
type SubscriptionRepository interface {
	// UpsertSnapshot applies snapshot only when version is not stale. The bool
	// reports whether the read model changed; stale events return (false, nil).
	UpsertSnapshot(ctx context.Context, userID, provider string, snapshot domain.SubscriptionSnapshot, occurredAt time.Time, providerEventID string) (bool, error)
}

// CheckoutReservationRepository serializes subscription/lifetime checkout
// creation across processes. A reservation must live at least as long as the
// provider-side checkout session it protects.
type CheckoutReservationRepository interface {
	ReserveCheckout(ctx context.Context, userID, provider, providerCustomerID, reservationID, intentKey string, now, expiresAt time.Time) (*domain.BillingCheckoutReservation, bool, error)
	ReplaceCheckoutReservation(ctx context.Context, userID, provider, providerCustomerID, expectedReservationID, reservationID, intentKey string, expiresAt time.Time) (bool, error)
	CompleteCheckoutReservation(ctx context.Context, userID, provider, reservationID string, result domain.CheckoutResult) error
	LinkCheckoutSubscription(ctx context.Context, provider, providerSessionID, providerSubscriptionID string) error
	ReleaseCheckoutReservation(ctx context.Context, provider, providerSessionID string) (bool, error)
	ReleaseCheckoutReservationByReservationID(ctx context.Context, provider, reservationID string) (bool, error)
	ReleaseCheckoutReservationBySubscription(ctx context.Context, provider, providerSubscriptionID string) (bool, error)
}

// UserResolver maps webhook payload hints to a userID known to the host
// application. The billing module is intentionally agnostic to the host's
// user model: implementers do whatever lookup they need.
type UserResolver interface {
	Resolve(ctx context.Context, hint UserHint) (userID string, err error)
}
