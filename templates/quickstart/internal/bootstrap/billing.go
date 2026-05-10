package bootstrap

import (
	stripeintegration "github.com/brizenchi/quickstart-template/internal/integration/stripe"
)

func (c AppConfig) StripeTopUpRuntimeConfig() stripeintegration.TopUpRuntimeConfig {
	return stripeintegration.TopUpRuntimeConfig{
		WebhookSecret: c.Billing.Stripe.WebhookSecret,
		MinAmountUSD:  stripeintegration.PositiveFloatOr(c.Billing.Stripe.TopUp.MinAmountUSD, stripeintegration.DefaultTopUpMinAmountUSD),
		MaxAmountUSD:  stripeintegration.PositiveFloatOr(c.Billing.Stripe.TopUp.MaxAmountUSD, stripeintegration.DefaultTopUpMaxAmountUSD),
		CreditsPerUSD: stripeintegration.PositiveInt64Or(c.Billing.Stripe.TopUp.CreditsPerUSD, stripeintegration.DefaultTopUpCreditsPerUSD),
	}
}
