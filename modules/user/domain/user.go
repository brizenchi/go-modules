package domain

import "time"

const (
	RoleUser  = "user"
	RoleAdmin = "admin"

	PlanFree    = "free"
	PlanStarter = "starter"
	PlanPro     = "pro"
	PlanPremium = "premium"
	PlanLifetime = "lifetime"
)

// User is the standard shared user shape reused across SaaS projects.
type User struct {
	ID                   string
	Email                string
	EmailVerified        bool
	EmailVerifiedAt      *time.Time
	Username             string
	AvatarURL            string
	Provider             string
	ProviderSubject      string
	Role                 string
	Plan                 string
	Credits              int
	StripeCustomerID     string
	StripeSubscriptionID string
	StripePriceID        string
	StripeProductID      string
	BillingStatus        string
	BillingPeriodStart   *time.Time
	BillingPeriodEnd     *time.Time
	CancelEffectiveAt    *time.Time
	LastLoginAt          *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}
