package serviceconfig

import (
	"strconv"
	"strings"

	"github.com/brizenchi/quickstart-template/internal/platform"
)

var providerNames = []string{"resend", "stripe"}
var priceFields = []string{"starter_monthly", "starter_yearly", "pro_monthly", "pro_yearly", "premium_monthly", "premium_yearly", "lifetime"}

func environmentConfiguration(cfg platform.Config, provider string) configuration {
	if provider == "resend" {
		return configuration{
			Enabled: strings.EqualFold(strings.TrimSpace(cfg.Email.Provider), "resend"),
			Fields: map[string]string{
				"sender_email": cfg.Email.Resend.SenderEmail, "sender_name": cfg.Email.Resend.SenderName,
				"email_auth_enabled": strconv.FormatBool(cfg.EmailAuthEnabled()),
			},
			Secrets: map[string]string{"api_key": cfg.Email.Resend.APIKey},
		}
	}
	s := cfg.Billing.Stripe
	mode := "test"
	if strings.HasPrefix(s.SecretKey, "sk_live_") || strings.HasPrefix(s.SecretKey, "rk_live_") || strings.HasPrefix(s.PublishableKey, "pk_live_") {
		mode = "live"
	}
	credits := s.Credits.PerPackage
	if credits <= 0 {
		credits = 100
	}
	return configuration{
		Enabled: cfg.BillingEnabled(),
		Fields: map[string]string{
			"mode": mode, "publishable_key": s.PublishableKey,
			"starter_monthly": s.Prices.StarterMonthly, "starter_yearly": s.Prices.StarterYearly,
			"pro_monthly": s.Prices.ProMonthly, "pro_yearly": s.Prices.ProYearly,
			"premium_monthly": s.Prices.PremiumMonthly, "premium_yearly": s.Prices.PremiumYearly,
			"lifetime": s.Prices.Lifetime, "credit_price_ids": strings.Join(s.Prices.Credits, ","),
			"trial_days": strconv.FormatInt(s.TrialDays, 10), "credits_per_package": strconv.FormatInt(credits, 10),
		},
		Secrets: map[string]string{"secret_key": s.SecretKey, "webhook_secret": s.WebhookSecret},
	}
}

func applyConfiguration(cfg *platform.Config, provider string, value configuration) {
	if provider == "resend" {
		cfg.Email.Provider = "none"
		emailAuth := false
		if value.Enabled {
			cfg.Email.Provider = "resend"
			emailAuth = value.Fields["email_auth_enabled"] == "true"
		}
		cfg.Auth.Email.Enabled = &emailAuth
		// Real delivery must never expose authentication codes in API responses.
		cfg.Auth.Email.Debug = false
		cfg.Email.Resend = platform.ResendConfig{APIKey: value.Secrets["api_key"], SenderEmail: value.Fields["sender_email"], SenderName: value.Fields["sender_name"]}
		return
	}
	enabled := value.Enabled
	cfg.Billing.Enabled = &enabled
	cfg.Billing.Provider = "stripe"
	s := &cfg.Billing.Stripe
	s.SecretKey, s.WebhookSecret = value.Secrets["secret_key"], value.Secrets["webhook_secret"]
	s.PublishableKey = value.Fields["publishable_key"]
	s.TrialDays, _ = strconv.ParseInt(value.Fields["trial_days"], 10, 64)
	s.Credits.PerPackage, _ = strconv.ParseInt(value.Fields["credits_per_package"], 10, 64)
	s.Prices = platform.StripePricesConfig{
		StarterMonthly: value.Fields["starter_monthly"], StarterYearly: value.Fields["starter_yearly"],
		ProMonthly: value.Fields["pro_monthly"], ProYearly: value.Fields["pro_yearly"],
		PremiumMonthly: value.Fields["premium_monthly"], PremiumYearly: value.Fields["premium_yearly"],
		Lifetime: value.Fields["lifetime"],
	}
	if value.Fields["credit_price_ids"] != "" {
		s.Prices.Credits = strings.Split(value.Fields["credit_price_ids"], ",")
	}
}

func cloneConfiguration(v configuration) configuration {
	copy := configuration{Enabled: v.Enabled, Fields: make(map[string]string, len(v.Fields)), Secrets: make(map[string]string, len(v.Secrets))}
	for key, value := range v.Fields {
		copy.Fields[key] = value
	}
	for key, value := range v.Secrets {
		copy.Secrets[key] = value
	}
	return copy
}
