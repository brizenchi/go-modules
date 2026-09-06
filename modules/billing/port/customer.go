package port

import (
	"context"

	"github.com/brizenchi/go-modules/modules/billing/domain"
)

// AccountLookup is the only host-user projection required by billing.
// Implement it in the host project; billing never queries a fixed users table.
type AccountLookup interface {
	FindBillingAccount(ctx context.Context, userID string) (Account, error)
	FindUserIDByEmail(ctx context.Context, email string) (string, error)
}

// Account is deliberately smaller than any host User model.
type Account struct {
	UserID string
	Email  string
}

// CustomerStore persists the mapping between host-app users and
// provider-side customer/subscription identifiers.
//
// This is the only "user" knowledge the billing module requires. Hosts
// A standard GORM implementation stores those identifiers in billing-owned
// tables and receives AccountLookup separately for the user's email.
type CustomerStore interface {
	// LoadCustomer returns the user's email and any known provider
	// customer/subscription IDs.
	LoadCustomer(ctx context.Context, userID string) (Customer, error)

	// SaveCustomerID persists the provider customer ID for a user.
	SaveCustomerID(ctx context.Context, userID, provider, customerID string) error

	// HasUsedTrial reports whether the user has ever had a subscription
	// in trialing or active state (i.e. has consumed their free-trial
	// opportunity). Used to prevent granting duplicate trials.
	HasUsedTrial(ctx context.Context, userID string) (bool, error)
}

// Customer is a minimal projection of a user for billing purposes.
type Customer struct {
	UserID                 string
	Email                  string
	Plan                   string
	ProviderCustomerID     string
	ProviderSubscriptionID string
	SubscriptionStatus     domain.SubscriptionStatus
}
