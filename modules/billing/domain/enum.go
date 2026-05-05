package domain

// PlanType identifies a subscription plan.
type PlanType string

const (
	PlanFree     PlanType = "free"
	PlanStarter  PlanType = "starter"
	PlanPro      PlanType = "pro"
	PlanPremium  PlanType = "premium"
	PlanLifetime PlanType = "lifetime"
)

func (p PlanType) Valid() bool {
	switch p {
	case PlanFree, PlanStarter, PlanPro, PlanPremium, PlanLifetime:
		return true
	}
	return false
}

// BillingInterval is the recurrence interval for a subscription.
type BillingInterval string

const (
	IntervalMonthly BillingInterval = "monthly"
	IntervalYearly  BillingInterval = "yearly"
)

func (i BillingInterval) Valid() bool {
	switch i {
	case IntervalMonthly, IntervalYearly:
		return true
	}
	return false
}

// ProductType distinguishes subscriptions from one-time purchases.
type ProductType string

const (
	ProductSubscription ProductType = "subscription"
	ProductCredits      ProductType = "credits"
	ProductLifetime     ProductType = "lifetime"
)

// SubscriptionStatus is the lifecycle state of a subscription.
type SubscriptionStatus string

const (
	StatusTrialing      SubscriptionStatus = "trialing"
	StatusActive        SubscriptionStatus = "active"
	StatusPastDue       SubscriptionStatus = "past_due"
	StatusCanceling     SubscriptionStatus = "canceling"
	StatusCanceled      SubscriptionStatus = "canceled"
	StatusIncomplete    SubscriptionStatus = "incomplete"
	StatusPaymentFailed SubscriptionStatus = "payment_failed"
)

// CancelMode controls when a cancellation takes effect.
type CancelMode string

const (
	CancelAtPeriodEnd CancelMode = "end_of_period"
	CancelIn3Days     CancelMode = "3days"
)

func (m CancelMode) Valid() bool {
	switch m {
	case CancelAtPeriodEnd, CancelIn3Days:
		return true
	}
	return false
}

// SubscriptionChangeMode controls how a plan or interval switch applies.
type SubscriptionChangeMode string

const (
	ChangeModeImmediateProrated   SubscriptionChangeMode = "immediate_prorated"
	ChangeModeImmediateResetCycle SubscriptionChangeMode = "immediate_reset_cycle"
	ChangeModePeriodEnd           SubscriptionChangeMode = "period_end"
)

func (m SubscriptionChangeMode) Valid() bool {
	switch m {
	case ChangeModeImmediateProrated, ChangeModeImmediateResetCycle, ChangeModePeriodEnd:
		return true
	}
	return false
}
