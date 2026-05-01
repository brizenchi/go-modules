// Package event defines billing domain events. Listeners subscribe to these
// to drive project-specific side effects (e.g. quota provisioning, notifications).
package event

import (
	"time"

	"github.com/brizenchi/go-modules/modules/billing/domain"
)

// Kind identifies a domain event type.
type Kind string

const (
	KindSubscriptionActivated   Kind = "subscription.activated"   // first payment / start
	KindSubscriptionRenewed     Kind = "subscription.renewed"     // recurring payment succeeded
	KindSubscriptionUpdated     Kind = "subscription.updated"     // generic update (plan change, etc)
	KindSubscriptionCanceling   Kind = "subscription.canceling"   // user requested cancel, still active until effective
	KindSubscriptionCanceled    Kind = "subscription.canceled"    // fully canceled, access revoked
	KindSubscriptionReactivated Kind = "subscription.reactivated" // canceling -> active again
	KindPaymentFailed           Kind = "payment.failed"
	KindCreditsPurchased        Kind = "credits.purchased"
)

// Envelope wraps every event with provenance.
type Envelope struct {
	Kind            Kind
	UserID          string
	Provider        string
	ProviderEventID string
	OccurredAt      time.Time
	Payload         any
}

// SubscriptionActivated is emitted on first successful subscription checkout.
type SubscriptionActivated struct {
	Snapshot domain.SubscriptionSnapshot
}

// SubscriptionRenewed is emitted on recurring invoice payment success.
type SubscriptionRenewed struct {
	Snapshot domain.SubscriptionSnapshot
}

// SubscriptionUpdated is emitted for non-billing subscription mutations.
type SubscriptionUpdated struct {
	Snapshot domain.SubscriptionSnapshot
}

// SubscriptionCanceling means user opted out; access remains until EffectiveAt.
type SubscriptionCanceling struct {
	Snapshot    domain.SubscriptionSnapshot
	Mode        domain.CancelMode
	EffectiveAt *time.Time
}

// SubscriptionCanceled means access has been revoked.
type SubscriptionCanceled struct {
	ProviderSubscriptionID string
	ProviderCustomerID     string
}

// SubscriptionReactivated means a canceling subscription was reactivated.
type SubscriptionReactivated struct {
	Snapshot domain.SubscriptionSnapshot
}

// PaymentFailed is emitted on invoice.payment_failed.
type PaymentFailed struct {
	ProviderSubscriptionID string
	ProviderCustomerID     string
}

// CreditsPurchased is emitted on a one-time credits checkout success.
type CreditsPurchased struct {
	Quantity       int64
	CreditsPerUnit int64
	TotalCredits   int64
	PriceID        string
}
