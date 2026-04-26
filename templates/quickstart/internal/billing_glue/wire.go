// Package billing_glue wires the billing module to your User table.
package billing_glue

import (
	"context"
	"log/slog"

	authevent "github.com/brizenchi/go-modules/auth/event"
	"github.com/brizenchi/go-modules/billing"
	"github.com/brizenchi/go-modules/billing/adapter/eventbus"
	"github.com/brizenchi/go-modules/billing/adapter/repo"
	stripeadapter "github.com/brizenchi/go-modules/billing/adapter/stripe"
	"github.com/brizenchi/go-modules/billing/domain"
	"github.com/brizenchi/go-modules/billing/event"
	"github.com/brizenchi/go-modules/billing/port"

	"github.com/spf13/viper"
	"gorm.io/gorm"
)

// Init wires the billing module from viper config + your DB.
func Init(db *gorm.DB) *billing.Module {
	provider := stripeadapter.NewProvider(stripeadapter.Config{
		Enabled:        viper.GetBool("billing.stripe.enabled"),
		SecretKey:      viper.GetString("billing.stripe.secret_key"),
		WebhookSecret:  viper.GetString("billing.stripe.webhook_secret"),
		PublishableKey: viper.GetString("billing.stripe.publishable_key"),
		SubscriptionPrices: map[domain.PlanType]map[domain.BillingInterval]string{
			domain.PlanStarter: {
				domain.IntervalMonthly: viper.GetString("billing.stripe.prices.starter_monthly"),
				domain.IntervalYearly:  viper.GetString("billing.stripe.prices.starter_yearly"),
			},
			domain.PlanPro: {
				domain.IntervalMonthly: viper.GetString("billing.stripe.prices.pro_monthly"),
				domain.IntervalYearly:  viper.GetString("billing.stripe.prices.pro_yearly"),
			},
		},
		CreditsPriceIDs: viper.GetStringSlice("billing.stripe.prices.credits"),
		CreditsPerUnit:  int64(viper.GetInt("billing.stripe.credits.per_package")),
		TrialDays:       viper.GetInt64("billing.stripe.trial_days"),
	})

	mod := billing.New(billing.Deps{
		Provider:     provider,
		Bus:          eventbus.NewInProc(),
		Customers:    &customerStore{db: db},
		EventRepo:    repo.NewBillingEventRepo(db),
		UserResolver: &userResolver{db: db},
		GetUserID:    nil, // set by host's middleware; quickstart's main.go wires it
	})

	mod.Subscribe(event.KindSubscriptionActivated, OnSubscriptionActivated)
	mod.Subscribe(event.KindSubscriptionRenewed, OnSubscriptionRenewed)
	mod.Subscribe(event.KindCreditsPurchased, OnCreditsPurchased)
	mod.Subscribe(event.KindSubscriptionCanceling, OnSubscriptionCanceling)
	mod.Subscribe(event.KindSubscriptionCanceled, OnSubscriptionCanceled)
	mod.Subscribe(event.KindPaymentFailed, OnPaymentFailed)

	return mod
}

// --- listeners (project-specific business logic) ---------------------

// OnUserSignedUp fires from the auth module when a user signs up.
// Use this to provision billing-side entities (e.g. wallet rows).
//
// Wired in cmd/main.go:
//
//	authMod.Subscribe(authevent.KindUserSignedUp, billing_glue.OnUserSignedUp)
func OnUserSignedUp(ctx context.Context, env authevent.Envelope) error {
	// TODO: provision wallet, send welcome email, etc.
	_ = env.UserID
	return nil
}

func OnSubscriptionActivated(ctx context.Context, env event.Envelope) error {
	p, _ := env.Payload.(event.SubscriptionActivated)
	slog.Info("billing: subscription activated", "user_id", env.UserID, "plan", p.Snapshot.Plan)
	// TODO: grant plan benefits (credits, quotas, feature flags).
	return nil
}

func OnSubscriptionRenewed(ctx context.Context, env event.Envelope) error {
	// TODO: top up monthly credits.
	return nil
}

func OnCreditsPurchased(ctx context.Context, env event.Envelope) error {
	p, _ := env.Payload.(event.CreditsPurchased)
	slog.Info("billing: credits purchased",
		"user_id", env.UserID,
		"quantity", p.Quantity,
		"total_credits", p.TotalCredits,
	)
	// TODO: add to wallet.
	return nil
}

func OnSubscriptionCanceling(ctx context.Context, env event.Envelope) error {
	// TODO: schedule downgrade at EffectiveAt.
	return nil
}

func OnSubscriptionCanceled(ctx context.Context, env event.Envelope) error {
	// TODO: revoke entitlements.
	return nil
}

func OnPaymentFailed(ctx context.Context, env event.Envelope) error {
	// TODO: notify the user, freeze account after N retries.
	return nil
}

// --- ports against your User table -----------------------------------

type customerStore struct{ db *gorm.DB }

func (s *customerStore) LoadCustomer(ctx context.Context, userID string) (port.Customer, error) {
	// TODO: read your User row → return Customer{UserID, Email, ProviderCustomerID, ProviderSubscriptionID}.
	return port.Customer{UserID: userID}, nil
}

func (s *customerStore) SaveCustomerID(ctx context.Context, userID, provider, customerID string) error {
	// TODO: persist the provider customer id on your User row.
	return nil
}

type userResolver struct{ db *gorm.DB }

func (r *userResolver) Resolve(ctx context.Context, h port.UserHint) (string, error) {
	// TODO: implement (lookup by user_id → customer_id → subscription_id → email).
	return h.UserID, nil
}

var (
	_ port.CustomerStore = (*customerStore)(nil)
	_ port.UserResolver  = (*userResolver)(nil)
)
