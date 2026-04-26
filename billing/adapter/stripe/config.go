// Package stripe is the Stripe adapter for pkg/billing/port.Provider.
//
// It is a self-contained implementation: it does not import host-app
// packages (no viper, no models, no DB). Callers construct a Config and
// pass it to NewProvider.
package stripe

import (
	"github.com/brizenchi/go-modules/billing/domain"
)

// Config holds the static configuration for the Stripe provider.
//
// Populate this from your config layer (viper, env, etc.) — the adapter
// itself does not read any configuration source.
type Config struct {
	Enabled        bool
	SecretKey      string
	WebhookSecret  string
	PublishableKey string

	// SubscriptionPrices is the matrix of (plan, interval) -> Stripe price ID.
	// Missing entries are treated as "not offered".
	SubscriptionPrices map[domain.PlanType]map[domain.BillingInterval]string

	// CreditsPriceIDs lists Stripe price IDs that represent credits SKUs.
	CreditsPriceIDs []string

	// CreditsPerUnit is the number of credits granted per credits-product unit.
	CreditsPerUnit int64

	// TrialDays is the number of free-trial days for new subscriptions.
	// 0 disables the trial.
	TrialDays int64
}

// PriceFor returns the Stripe price ID for a (plan, interval), or "" if missing.
func (c Config) PriceFor(plan domain.PlanType, interval domain.BillingInterval) string {
	if c.SubscriptionPrices == nil {
		return ""
	}
	if m := c.SubscriptionPrices[plan]; m != nil {
		return m[interval]
	}
	return ""
}

// PlanForPrice resolves a Stripe price ID into (plan, interval).
// Returns (PlanFree, "") if unknown.
func (c Config) PlanForPrice(priceID string) (domain.PlanType, domain.BillingInterval) {
	if priceID == "" {
		return domain.PlanFree, ""
	}
	for plan, intervals := range c.SubscriptionPrices {
		for interval, id := range intervals {
			if id == priceID {
				return plan, interval
			}
		}
	}
	return domain.PlanFree, ""
}

// IsCreditsPriceID reports whether the price ID is a credits SKU.
func (c Config) IsCreditsPriceID(priceID string) bool {
	for _, id := range c.CreditsPriceIDs {
		if id == priceID {
			return true
		}
	}
	return false
}
