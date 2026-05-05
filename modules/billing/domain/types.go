package domain

import "time"

// CheckoutInput is the provider-agnostic input for opening a hosted checkout.
type CheckoutInput struct {
	UserID      string
	Email       string
	CustomerID  string // optional, providers may require this
	ProductType ProductType
	Plan        PlanType
	Interval    BillingInterval
	PriceID     string // required for credits, optional for subscription
	Quantity    int64
	SuccessURL  string
	CancelURL   string
	TrialDays   int64 // 0 = no trial
	Metadata    map[string]string
}

// CheckoutResult is the result of CreateCheckout.
type CheckoutResult struct {
	SessionID   string
	CheckoutURL string
}

// SubscriptionChangeInput describes a professional in-place plan change.
//
// Providers should keep the billing cycle unchanged by default and use
// prorated charging unless the host explicitly asks to reset the cycle.
type SubscriptionChangeInput struct {
	Plan     PlanType
	Interval BillingInterval
	Mode     SubscriptionChangeMode
}

// SubscriptionPreviewInput previews a subscription mutation before applying it.
type SubscriptionPreviewInput struct {
	Plan     PlanType
	Interval BillingInterval
	Mode     SubscriptionChangeMode
}

// SubscriptionPreview summarizes the expected commercial effect.
type SubscriptionPreview struct {
	Currency              string
	AmountDueNow          float64
	CurrentPeriodEnd      *time.Time
	NextBillingAt         *time.Time
	TargetPlan            PlanType
	TargetInterval        BillingInterval
	Mode                  SubscriptionChangeMode
	ImmediateCharge       bool
	EffectiveAtPeriodEnd  bool
	Message               string
}

// SubscriptionSnapshot is a provider-agnostic view of a subscription.
type SubscriptionSnapshot struct {
	ProviderSubscriptionID string
	ProviderCustomerID     string
	ProviderPriceID        string
	ProviderProductID      string
	ProductType            ProductType
	Plan                   PlanType
	Interval               BillingInterval
	Status                 SubscriptionStatus
	CancelAtPeriodEnd      bool
	PeriodStart            *time.Time
	PeriodEnd              *time.Time
	CancelEffectiveAt      *time.Time
}

// PaymentMethodCard summarizes a default card on file.
type PaymentMethodCard struct {
	Brand    string
	Last4    string
	ExpMonth int64
	ExpYear  int64
}

// PortalSessionResult is the result of opening a hosted billing portal.
type PortalSessionResult struct {
	URL string
}

// InvoiceItem is a provider-agnostic invoice line.
type InvoiceItem struct {
	ID        string
	AmountUSD float64
	Status    string
	Period    string // YYYY-MM
	PDFURL    string
	CreatedAt time.Time
}
