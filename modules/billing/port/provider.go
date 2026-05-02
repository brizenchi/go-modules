// Package port defines the interfaces the billing module depends on.
// Adapters live under modules/billing/adapter/.
package port

import (
	"context"

	"github.com/brizenchi/go-modules/modules/billing/domain"
	"github.com/brizenchi/go-modules/modules/billing/event"
)

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
}

// WebhookParseResult is the verified outcome of a webhook delivery.
//
// RawPayload is the original bytes (for idempotency persistence).
// ProviderEventID + Type identify the event. Envelopes are the domain
// events to publish. Envelopes may be empty for events we do not act on.
type WebhookParseResult struct {
	ProviderEventID string
	Type            string
	UserHint        UserHint
	RawPayload      []byte
	Envelopes       []event.Envelope
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
