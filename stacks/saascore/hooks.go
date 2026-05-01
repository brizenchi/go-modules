package saascore

import (
	"context"
	"time"

	authdomain "github.com/brizenchi/go-modules/modules/auth/domain"
	billingdomain "github.com/brizenchi/go-modules/modules/billing/domain"
	referraldomain "github.com/brizenchi/go-modules/modules/referral/domain"
)

type HostHooks struct {
	OnUserSignedUp            func(ctx context.Context, event UserSignedUpEvent) error
	OnSubscriptionActivated   func(ctx context.Context, event SubscriptionEvent) error
	OnSubscriptionRenewed     func(ctx context.Context, event SubscriptionEvent) error
	OnSubscriptionUpdated     func(ctx context.Context, event SubscriptionEvent) error
	OnSubscriptionReactivated func(ctx context.Context, event SubscriptionEvent) error
	OnSubscriptionCanceling   func(ctx context.Context, event SubscriptionCancelingEvent) error
	OnSubscriptionCanceled    func(ctx context.Context, event SubscriptionCanceledEvent) error
	OnPaymentFailed           func(ctx context.Context, event PaymentFailedEvent) error
	OnCreditsPurchased        func(ctx context.Context, event CreditsPurchasedEvent) error
	OnReferralRegistered      func(ctx context.Context, event ReferralRegisteredEvent) error
	OnReferralActivated       func(ctx context.Context, event ReferralActivatedEvent) error
}

type PolicyHooks struct {
	ResolveReferralReward func(ctx context.Context, input ReferralRewardPolicyInput) (int, error)
}

type UserSignedUpEvent struct {
	UserID     string
	OccurredAt time.Time
	Identity   authdomain.Identity
}

type SubscriptionEvent struct {
	UserID          string
	OccurredAt      time.Time
	Provider        string
	ProviderEventID string
	Snapshot        billingdomain.SubscriptionSnapshot
}

type SubscriptionCancelingEvent struct {
	UserID          string
	OccurredAt      time.Time
	Provider        string
	ProviderEventID string
	Snapshot        billingdomain.SubscriptionSnapshot
	Mode            billingdomain.CancelMode
	EffectiveAt     *time.Time
}

type SubscriptionCanceledEvent struct {
	UserID                 string
	OccurredAt             time.Time
	Provider               string
	ProviderEventID        string
	ProviderSubscriptionID string
	ProviderCustomerID     string
}

type PaymentFailedEvent struct {
	UserID                 string
	OccurredAt             time.Time
	Provider               string
	ProviderEventID        string
	ProviderSubscriptionID string
	ProviderCustomerID     string
}

type CreditsPurchasedEvent struct {
	UserID          string
	OccurredAt      time.Time
	Provider        string
	ProviderEventID string
	Quantity        int64
	CreditsPerUnit  int64
	TotalCredits    int64
	PriceID         string
}

type ReferralRegisteredEvent struct {
	OccurredAt time.Time
	Referral   referraldomain.Referral
}

type ReferralActivatedEvent struct {
	OccurredAt time.Time
	Referral   referraldomain.Referral
}

type ReferralRewardPolicyInput struct {
	ReferrerID string
	RefereeID  string
}
