// Package port defines the interfaces the billing module depends on.
// Adapters live under modules/billing/adapter/.
package port

import (
	"context"

	"github.com/brizenchi/go-modules/modules/billing/domain"
	"github.com/brizenchi/go-modules/modules/billing/event"
)

// ReturnURLValidator enforces host-owned redirect boundaries for checkout and
// billing portal URLs supplied by clients.
type ReturnURLValidator interface {
	ValidateReturnURL(rawURL string) error
}

// Provider abstracts a payment service (Stripe, Paddle, Alipay, ...).
//
// Implementations must be safe for concurrent use.
type Provider interface {
	// Name returns the provider identifier ("stripe", "paddle", ...).
	Name() string

	// Enabled reports whether the provider is configured and usable.
	Enabled() bool

	// EnsureCustomer returns an existing or new provider-side customer ID
	// for the given user.
	EnsureCustomer(ctx context.Context, userID, email, existingCustomerID string) (string, error)

	// CreateCheckout opens a hosted checkout session.
	CreateCheckout(ctx context.Context, in domain.CheckoutInput) (*domain.CheckoutResult, error)

	// GetCheckoutSession fetches authoritative provider-side checkout state.
	// Unknown states and provider errors must be returned as errors so callers
	// cannot replace a reservation without terminal proof.
	GetCheckoutSession(ctx context.Context, providerSessionID string) (*domain.CheckoutSessionSnapshot, error)

	// FindCheckoutSession locates a Session using the server-generated
	// reservation metadata. A nil result after an exhaustive provider query is
	// authoritative evidence that no Session exists for that reservation.
	FindCheckoutSession(ctx context.Context, providerCustomerID, reservationID string) (*domain.CheckoutSessionSnapshot, error)

	// CancelSubscription cancels a subscription according to the given mode.
	CancelSubscription(ctx context.Context, providerSubscriptionID string, mode domain.CancelMode) error

	// ChangeSubscription mutates the active subscription in-place.
	ChangeSubscription(ctx context.Context, providerSubscriptionID string, in domain.SubscriptionChangeInput) (*domain.SubscriptionSnapshot, error)

	// ScheduleSubscriptionChange applies a period-end switch.
	ScheduleSubscriptionChange(ctx context.Context, providerSubscriptionID string, in domain.SubscriptionChangeInput) (*domain.SubscriptionSnapshot, error)

	// PreviewSubscriptionChange estimates how the provider will bill the switch.
	PreviewSubscriptionChange(ctx context.Context, providerCustomerID, providerSubscriptionID string, in domain.SubscriptionPreviewInput) (*domain.SubscriptionPreview, error)

	// ReactivateSubscription clears any pending cancellation.
	ReactivateSubscription(ctx context.Context, providerSubscriptionID string) error

	// GetSubscription fetches a fresh snapshot of a subscription.
	GetSubscription(ctx context.Context, providerSubscriptionID string) (*domain.SubscriptionSnapshot, error)

	// GetDefaultPaymentMethod returns the customer's default card on file, if any.
	GetDefaultPaymentMethod(ctx context.Context, providerCustomerID string) (*domain.PaymentMethodCard, error)

	// ListInvoices returns paginated invoices for the customer.
	ListInvoices(ctx context.Context, providerCustomerID string, page, limit int) ([]domain.InvoiceItem, int, error)

	// CreateBillingPortalSession opens a hosted customer billing portal.
	CreateBillingPortalSession(ctx context.Context, providerCustomerID, returnURL string) (*domain.PortalSessionResult, error)

	// VerifyAndParseWebhook verifies the signature, parses the payload,
	// and returns: (a) the raw event id/type for idempotency, (b) the
	// derived domain events to dispatch.
	VerifyAndParseWebhook(payload []byte, signature string) (*WebhookParseResult, error)

	// MapPriceToPlan resolves a provider price ID into (plan, interval).
	// Returns PlanFree on unknown.
	MapPriceToPlan(priceID string) (domain.PlanType, domain.BillingInterval)

	// CreditsPerUnit returns credits granted per credits-product unit.
	CreditsPerUnit() int64

	// IsCreditsPriceID returns true when the price id is configured as a credits SKU.
	IsCreditsPriceID(priceID string) bool

	// LifetimePriceID returns the configured one-time lifetime price ID, if any.
	LifetimePriceID() string
}

// WebhookParseResult is the verified outcome of a webhook delivery.
//
// RawPayload is the original bytes (for idempotency persistence).
// ProviderEventID + Type identify the event. Envelopes are the domain
// events to publish. Envelopes may be empty for events we do not act on.
type WebhookParseResult struct {
	ProviderEventID            string
	Type                       string
	UserHint                   UserHint
	RawPayload                 []byte
	Envelopes                  []event.Envelope
	CheckoutSessionID          string
	CheckoutReservationID      string
	CheckoutSubscriptionID     string
	ReleaseCheckoutReservation bool
}

// UserHint carries identifiers extracted from a webhook payload that may
// help resolve which user the event belongs to. The application layer
// uses these against UserResolver.
type UserHint struct {
	UserID                 string
	Email                  string
	ProviderCustomerID     string
	ProviderSubscriptionID string
}

// ScheduledChangeCanceler is an optional provider capability.
type ScheduledChangeCanceler interface {
	CancelScheduledChange(ctx context.Context, subscriptionID string) error
}

// LifetimeSubscriptionRetirer replaces a recurring subscription only after a
// verified Lifetime payment. It must not refund, prorate or invoice the old plan.
type LifetimeSubscriptionRetirer interface {
	RetireSubscriptionForLifetime(ctx context.Context, subscriptionID, customerID string) error
}

// EventRetriever permits authenticated recovery without bypassing webhook verification.
type EventRetriever interface {
	RetrieveEvent(ctx context.Context, id string) (*WebhookParseResult, error)
}

// ContextWebhookParser preserves cancellation during provider enrichment.
type ContextWebhookParser interface {
	VerifyAndParseWebhookContext(context.Context, []byte, string) (*WebhookParseResult, error)
}

// CheckoutSessionExpirer is an optional provider capability for replacing an
// abandoned checkout. It returns authoritative state after attempting expiration;
// only an expired, unpaid session is proof that the old URL cannot accept payment.
// A session completed concurrently must never be treated as expired.
type CheckoutSessionExpirer interface {
	ExpireCheckoutSession(ctx context.Context, providerSessionID string) (*domain.CheckoutSessionSnapshot, error)
}

// CreditsCheckoutReader verifies a credit checkout through the authenticated provider API.
// It does not grant credits; hosts must check ownership and fulfill idempotently.
type CreditsCheckoutReader interface {
	GetCreditsCheckout(context.Context, string) (*CreditsCheckout, error)
}

// CreditsCheckout carries provider-confirmed state and a purchase only after payment.
type CreditsCheckout struct {
	SessionID, UserID, State, PaymentStatus string
	Purchase                                *event.CreditsPurchased
}
