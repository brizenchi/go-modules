package port

import "context"

// CustomerStore persists the mapping between host-app users and
// provider-side customer/subscription identifiers.
//
// This is the only "user" knowledge the billing module requires. Hosts
// can implement it against an existing users table (e.g. by writing to
// stripe_customer_id columns) or a dedicated billing_customers table.
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
}
