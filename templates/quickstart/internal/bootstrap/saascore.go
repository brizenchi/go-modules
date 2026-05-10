package bootstrap

import (
	"strings"
	"time"

	"github.com/brizenchi/go-modules/stacks/saascore"
)

func (c AppConfig) SaaSCoreConfig() saascore.Config {
	stripeEnabled := strings.TrimSpace(c.Billing.Stripe.SecretKey) != "" &&
		strings.TrimSpace(c.Billing.Stripe.WebhookSecret) != ""

	return saascore.Config{
		ServiceName: c.Server.Name,
		Auth: saascore.AuthConfig{
			FrontendRedirect: c.Auth.FrontendRedirect,
			UserJWTSecret:    c.Auth.UserJWTSecret,
			UserJWTExpire:    time.Duration(intWithDefault(c.Auth.UserJWTExpireHours, 168)) * time.Hour,
			WSTicketTTL:      time.Duration(intWithDefault(c.Auth.WSTicketTTLSeconds, 300)) * time.Second,
			AdminEmails:      c.Auth.AdminEmails,
			EmailCode: saascore.EmailCodeConfig{
				Debug:        c.Auth.Email.Debug,
				TTL:          time.Duration(intWithDefault(c.Auth.Email.Code.TTLMinutes, 10)) * time.Minute,
				MinResendGap: time.Duration(intWithDefault(c.Auth.Email.Code.MinResendGapSeconds, 60)) * time.Second,
				DailyCap:     intWithDefault(c.Auth.Email.Code.DailyCap, 10),
				MaxAttempts:  intWithDefault(c.Auth.Email.Code.MaxAttempts, 5),
			},
			Google: saascore.GoogleOAuthConfig{
				ClientID:     c.Auth.Google.ClientID,
				ClientSecret: c.Auth.Google.ClientSecret,
				RedirectURL:  c.Auth.Google.RedirectURL,
				StateSecret:  c.Auth.Google.StateSecret,
				StateTTL:     time.Duration(intWithDefault(c.Auth.Google.StateTTLMin, 20)) * time.Minute,
				Scope:        c.Auth.Google.Scope,
			},
		},
		Email: saascore.EmailConfig{
			Provider: c.Email.Provider,
			Brevo: saascore.BrevoConfig{
				APIKey:      c.Email.Brevo.APIKey,
				SenderEmail: c.Email.Brevo.SenderEmail,
				SenderName:  c.Email.Brevo.SenderName,
			},
			Resend: saascore.ResendConfig{
				APIKey:      c.Email.Resend.APIKey,
				SenderEmail: c.Email.Resend.SenderEmail,
				SenderName:  c.Email.Resend.SenderName,
			},
		},
		Billing: saascore.BillingConfig{
			Stripe: saascore.StripeConfig{
				Enabled:               stripeEnabled,
				SecretKey:             c.Billing.Stripe.SecretKey,
				PublishableKey:        c.Billing.Stripe.PublishableKey,
				WebhookSecret:         c.Billing.Stripe.WebhookSecret,
				TrialDays:             c.Billing.Stripe.TrialDays,
				StarterMonthlyPriceID: c.Billing.Stripe.Prices.StarterMonthly,
				StarterYearlyPriceID:  c.Billing.Stripe.Prices.StarterYearly,
				ProMonthlyPriceID:     c.Billing.Stripe.Prices.ProMonthly,
				ProYearlyPriceID:      c.Billing.Stripe.Prices.ProYearly,
				PremiumMonthlyPriceID: c.Billing.Stripe.Prices.PremiumMonthly,
				PremiumYearlyPriceID:  c.Billing.Stripe.Prices.PremiumYearly,
				LifetimePriceID:       c.Billing.Stripe.Prices.Lifetime,
				CreditsPriceIDs:       c.Billing.Stripe.Prices.Credits,
				CreditsPerPackage:     c.Billing.Stripe.Credits.PerPackage,
			},
		},
		Referral: saascore.ReferralConfig{
			Prefix:           c.Referral.Prefix,
			BaseLink:         c.Referral.BaseLink,
			ActivationReward: c.Referral.ActivationReward,
		},
	}
}

func intWithDefault(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}
