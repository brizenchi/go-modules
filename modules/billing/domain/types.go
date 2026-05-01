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

// SubscriptionSnapshot is a provider-agnostic view of a subscription.
type SubscriptionSnapshot struct {
	ProviderSubscriptionID string
	ProviderCustomerID     string
	ProviderPriceID        string
	ProviderProductID      string
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

// InvoiceItem is a provider-agnostic invoice line.
type InvoiceItem struct {
	ID        string
	AmountUSD float64
	Status    string
	Period    string // YYYY-MM
	PDFURL    string
	CreatedAt time.Time
}
