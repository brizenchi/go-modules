package platform

import (
	"fmt"
	"strings"

	"github.com/brizenchi/go-modules/modules/billing"
	billingeventbus "github.com/brizenchi/go-modules/modules/billing/adapter/eventbus"
	billingrepo "github.com/brizenchi/go-modules/modules/billing/adapter/repo"
	stripeadapter "github.com/brizenchi/go-modules/modules/billing/adapter/stripe"
	billingapp "github.com/brizenchi/go-modules/modules/billing/app"
	billingdomain "github.com/brizenchi/go-modules/modules/billing/domain"
	"github.com/brizenchi/quickstart-template/internal/user"
	"gorm.io/gorm"
)

func buildBilling(db *gorm.DB, cfg Config, users *user.Repository) (*billing.Module, error) {
	if !strings.EqualFold(strings.TrimSpace(cfg.Billing.Provider), "stripe") {
		return nil, fmt.Errorf("platform: unsupported billing provider %q; implement buildBilling for this SaaS", cfg.Billing.Provider)
	}
	stripeCfg := cfg.Billing.Stripe
	if strings.TrimSpace(stripeCfg.SecretKey) == "" || strings.TrimSpace(stripeCfg.WebhookSecret) == "" {
		return nil, fmt.Errorf("platform: stripe secret_key and webhook_secret required when billing enabled")
	}
	lookup := user.NewBillingLookup(users)
	returnURLs, err := billingapp.NewOriginReturnURLValidator(cfg.Auth.FrontendRedirect)
	if err != nil {
		return nil, fmt.Errorf("platform: billing frontend origin: %w", err)
	}
	return billing.New(billing.Deps{
		Provider: stripeadapter.NewProvider(stripeadapter.Config{
			Enabled:        true,
			SecretKey:      stripeCfg.SecretKey,
			WebhookSecret:  stripeCfg.WebhookSecret,
			PublishableKey: stripeCfg.PublishableKey,
			SubscriptionPrices: map[billingdomain.PlanType]map[billingdomain.BillingInterval]string{
				billingdomain.PlanStarter: {
					billingdomain.IntervalMonthly: stripeCfg.Prices.StarterMonthly,
					billingdomain.IntervalYearly:  stripeCfg.Prices.StarterYearly,
				},
				billingdomain.PlanPro: {
					billingdomain.IntervalMonthly: stripeCfg.Prices.ProMonthly,
					billingdomain.IntervalYearly:  stripeCfg.Prices.ProYearly,
				},
				billingdomain.PlanPremium: {
					billingdomain.IntervalMonthly: stripeCfg.Prices.PremiumMonthly,
					billingdomain.IntervalYearly:  stripeCfg.Prices.PremiumYearly,
				},
			},
			LifetimePriceID: stripeCfg.Prices.Lifetime,
			CreditsPriceIDs: stripeCfg.Prices.Credits,
			CreditsPerUnit:  stripeCfg.Credits.PerPackage,
			TrialDays:       stripeCfg.TrialDays,
		}),
		Bus:                billingeventbus.NewInProc(),
		Customers:          billingrepo.NewCustomerStore(db, lookup),
		EventRepo:          billingrepo.NewBillingEventRepo(db),
		Subscriptions:      billingrepo.NewSubscriptionRepo(db),
		UserResolver:       billingrepo.NewUserResolver(db, lookup),
		GetUserID:          userIDFromGin,
		ReturnURLValidator: returnURLs,
	}), nil
}
