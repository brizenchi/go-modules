package bootstrap

import (
	"context"
	"errors"
	"testing"
	"time"

	authdomain "github.com/brizenchi/go-modules/modules/auth/domain"
	authevent "github.com/brizenchi/go-modules/modules/auth/event"
	billingdomain "github.com/brizenchi/go-modules/modules/billing/domain"
	billingevent "github.com/brizenchi/go-modules/modules/billing/event"
	referraldomain "github.com/brizenchi/go-modules/modules/referral/domain"
	referralevent "github.com/brizenchi/go-modules/modules/referral/event"
	"github.com/brizenchi/quickstart-template/internal/hostapi"
	"github.com/brizenchi/quickstart-template/internal/platform"
	"github.com/brizenchi/quickstart-template/internal/user"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func boolPtr(value bool) *bool { return &value }

func TestTemplateSubscriptionsApplyHostRules(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	moduleConfig := platform.Config{
		ServiceName: "subscription-test",
		Email:       platform.EmailConfig{Provider: "none"},
		Auth: platform.AuthConfig{
			UserJWTSecret:    "test-jwt-secret",
			FrontendRedirect: "http://localhost:3000",
			Email:            platform.AuthEmailConfig{Enabled: boolPtr(false)},
		},
		Billing: platform.BillingConfig{
			Enabled:  boolPtr(true),
			Provider: "stripe",
			Stripe: platform.StripeConfig{
				SecretKey:     "sk_test_placeholder",
				WebhookSecret: "whsec_placeholder",
			},
		},
		Referral: platform.ReferralConfig{Enabled: boolPtr(true), ActivationReward: 50},
	}
	if err := platform.Migrate(db, moduleConfig); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	modules, err := platform.New(db, moduleConfig)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	appConfig := AppConfig{
		Auth:     moduleConfig.Auth,
		Email:    moduleConfig.Email,
		Billing:  moduleConfig.Billing,
		Referral: moduleConfig.Referral,
	}
	appConfig.Host.SignupCredits = 20
	deps := hostapi.Deps{DB: db, Modules: modules, Users: modules.Users, Config: appConfig.Host}
	subscribeModuleEvents(deps, appConfig)

	identity, err := user.NewAuthStore(modules.Users).FindOrCreateByEmail(context.Background(), "event@example.com")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	now := time.Now().UTC()
	modules.Auth.Deps.Bus.Publish(context.Background(), authevent.Envelope{
		Kind:       authevent.KindUserSignedUp,
		UserID:     identity.UserID,
		OccurredAt: now,
		Payload:    authevent.UserSignedUp{Identity: authdomain.Identity{UserID: identity.UserID, Email: identity.Email}},
	})
	modules.Billing.Bus.Publish(context.Background(), billingevent.Envelope{
		Kind:            billingevent.KindCreditsPurchased,
		UserID:          identity.UserID,
		ProviderEventID: "evt_credits_1",
		OccurredAt:      now,
		Payload:         billingevent.CreditsPurchased{TotalCredits: 100},
	})
	// Provider retries must not double-grant paid credits.
	modules.Billing.Bus.Publish(context.Background(), billingevent.Envelope{
		Kind:            billingevent.KindCreditsPurchased,
		UserID:          identity.UserID,
		ProviderEventID: "evt_credits_1",
		OccurredAt:      now,
		Payload:         billingevent.CreditsPurchased{TotalCredits: 100},
	})
	referralActivated := referralevent.Envelope{
		Kind:       referralevent.KindReferralActivated,
		OccurredAt: now,
		Payload: referralevent.ReferralActivated{Referral: referraldomain.Referral{
			ReferrerID:    identity.UserID,
			RefereeID:     "referee",
			RewardCredits: 50,
		}},
	}
	if err := modules.Referral.Deps.Bus.Publish(context.Background(), referralActivated); err != nil {
		t.Fatalf("publish referral activation: %v", err)
	}
	// Retrying delivery after a transient listener failure must not double-grant.
	if err := modules.Referral.Deps.Bus.Publish(context.Background(), referralActivated); err != nil {
		t.Fatalf("retry referral activation: %v", err)
	}

	got, err := modules.Users.FindByID(context.Background(), identity.UserID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.Credits != 170 {
		t.Fatalf("credits=%d, want signup 20 + purchase 100 + referral 50", got.Credits)
	}
}

func TestReferralRewardWaitsForPaidRenewalAfterTrial(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	moduleConfig := platform.Config{
		ServiceName: "trial-referral-test",
		Email:       platform.EmailConfig{Provider: "none"},
		Auth: platform.AuthConfig{
			UserJWTSecret:    "test-jwt-secret",
			FrontendRedirect: "http://localhost:3000",
			Email:            platform.AuthEmailConfig{Enabled: boolPtr(false)},
		},
		Billing: platform.BillingConfig{
			Enabled:  boolPtr(true),
			Provider: "stripe",
			Stripe: platform.StripeConfig{
				SecretKey:     "sk_test_placeholder",
				WebhookSecret: "whsec_placeholder",
			},
		},
		Referral: platform.ReferralConfig{Enabled: boolPtr(true), ActivationReward: 50},
	}
	if err := platform.Migrate(db, moduleConfig); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	modules, err := platform.New(db, moduleConfig)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	appConfig := AppConfig{Auth: moduleConfig.Auth, Email: moduleConfig.Email, Billing: moduleConfig.Billing, Referral: moduleConfig.Referral}
	deps := hostapi.Deps{DB: db, Modules: modules, Users: modules.Users, Config: appConfig.Host}
	subscribeModuleEvents(deps, appConfig)

	authStore := user.NewAuthStore(modules.Users)
	referrer, err := authStore.FindOrCreateByEmail(context.Background(), "referrer@example.com")
	if err != nil {
		t.Fatalf("create referrer: %v", err)
	}
	referee, err := authStore.FindOrCreateByEmail(context.Background(), "referee@example.com")
	if err != nil {
		t.Fatalf("create referee: %v", err)
	}
	code, err := modules.Referral.Code.GetOrCreate(context.Background(), referrer.UserID)
	if err != nil {
		t.Fatalf("create referral code: %v", err)
	}
	if _, err := modules.Referral.Attribute.AttributeReferral(context.Background(), referee.UserID, code.Value); err != nil {
		t.Fatalf("attribute referral: %v", err)
	}

	now := time.Now().UTC()
	if err := modules.Billing.Bus.Publish(context.Background(), billingevent.Envelope{
		Kind:            billingevent.KindSubscriptionActivated,
		UserID:          referee.UserID,
		ProviderEventID: "evt_trial_started",
		OccurredAt:      now,
		Payload: billingevent.SubscriptionActivated{Snapshot: billingdomain.SubscriptionSnapshot{
			Status: billingdomain.StatusTrialing,
		}},
	}); err != nil {
		t.Fatalf("publish trial activation: %v", err)
	}
	pending, err := modules.Referral.Deps.Referrals.FindByReferee(context.Background(), referee.UserID)
	if err != nil {
		t.Fatalf("find pending referral: %v", err)
	}
	if pending.Status != referraldomain.StatusPending || pending.RewardCredits != 0 {
		t.Fatalf("trial referral=%+v, want pending without reward", pending)
	}
	if err := modules.Billing.Bus.Publish(context.Background(), billingevent.Envelope{
		Kind:            billingevent.KindSubscriptionRenewed,
		UserID:          referee.UserID,
		ProviderEventID: "evt_comped_renewal",
		OccurredAt:      now.Add(30 * time.Minute),
		Payload: billingevent.SubscriptionRenewed{
			AmountPaid: 0,
			Currency:   "usd",
			Snapshot:   billingdomain.SubscriptionSnapshot{Status: billingdomain.StatusActive},
		},
	}); err != nil {
		t.Fatalf("publish zero-amount renewal: %v", err)
	}
	pending, err = modules.Referral.Deps.Referrals.FindByReferee(context.Background(), referee.UserID)
	if err != nil {
		t.Fatalf("find referral after zero-amount renewal: %v", err)
	}
	if pending.Status != referraldomain.StatusPending || pending.RewardCredits != 0 {
		t.Fatalf("zero-amount renewal referral=%+v, want pending without reward", pending)
	}

	paidRenewal := billingevent.Envelope{
		Kind:            billingevent.KindSubscriptionRenewed,
		UserID:          referee.UserID,
		ProviderEventID: "evt_trial_paid",
		OccurredAt:      now.Add(time.Hour),
		Payload: billingevent.SubscriptionRenewed{
			AmountPaid: 1200,
			Currency:   "usd",
			Snapshot:   billingdomain.SubscriptionSnapshot{Status: billingdomain.StatusActive},
		},
	}
	if err := modules.Billing.Bus.Publish(context.Background(), paidRenewal); err != nil {
		t.Fatalf("publish first paid renewal: %v", err)
	}
	// Delivery is at-least-once; the reward credit ledger must absorb retries.
	if err := modules.Billing.Bus.Publish(context.Background(), paidRenewal); err != nil {
		t.Fatalf("retry first paid renewal: %v", err)
	}

	activated, err := modules.Referral.Deps.Referrals.FindByReferee(context.Background(), referee.UserID)
	if err != nil {
		t.Fatalf("find activated referral: %v", err)
	}
	if activated.Status != referraldomain.StatusActivated || activated.RewardCredits != 50 {
		t.Fatalf("converted referral=%+v, want activated reward 50", activated)
	}
	updatedReferrer, err := modules.Users.FindByID(context.Background(), referrer.UserID)
	if err != nil {
		t.Fatalf("find referrer: %v", err)
	}
	if updatedReferrer.Credits != 50 {
		t.Fatalf("referrer credits=%d, want exactly 50 after retry", updatedReferrer.Credits)
	}
}

func TestLoginRetriesReferralAttributionAfterRegisteredListenerFailure(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	moduleConfig := platform.Config{
		ServiceName: "login-referral-retry-test",
		Email:       platform.EmailConfig{Provider: "none"},
		Auth: platform.AuthConfig{
			UserJWTSecret: "test-jwt-secret",
			Email:         platform.AuthEmailConfig{Enabled: boolPtr(false)},
		},
		Billing:  platform.BillingConfig{Enabled: boolPtr(false)},
		Referral: platform.ReferralConfig{Enabled: boolPtr(true)},
	}
	if err := platform.Migrate(db, moduleConfig); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	modules, err := platform.New(db, moduleConfig)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	appConfig := AppConfig{Auth: moduleConfig.Auth, Email: moduleConfig.Email, Billing: moduleConfig.Billing, Referral: moduleConfig.Referral}
	deps := hostapi.Deps{DB: db, Modules: modules, Users: modules.Users, Config: appConfig.Host}
	subscribeModuleEvents(deps, appConfig)

	authStore := user.NewAuthStore(modules.Users)
	referrer, err := authStore.FindOrCreateByEmail(context.Background(), "retry-referrer@example.com")
	if err != nil {
		t.Fatalf("create referrer: %v", err)
	}
	referee, err := authStore.FindOrCreateByEmail(context.Background(), "retry-referee@example.com")
	if err != nil {
		t.Fatalf("create referee: %v", err)
	}
	code, err := modules.Referral.Code.GetOrCreate(context.Background(), referrer.UserID)
	if err != nil {
		t.Fatalf("create referral code: %v", err)
	}

	deliveryAttempts := 0
	modules.Referral.Subscribe(referralevent.KindReferralRegistered, func(context.Context, referralevent.Envelope) error {
		deliveryAttempts++
		if deliveryAttempts <= 3 {
			return errors.New("temporary registered-listener failure")
		}
		return nil
	})
	now := time.Now().UTC()
	ctx := platform.WithReferralCode(context.Background(), code.Value)
	modules.Auth.Deps.Bus.Publish(ctx, authevent.Envelope{
		Kind:       authevent.KindUserSignedUp,
		UserID:     referee.UserID,
		OccurredAt: now,
		Payload:    authevent.UserSignedUp{Identity: authdomain.Identity{UserID: referee.UserID, Email: referee.Email}},
	})
	modules.Auth.Deps.Bus.Publish(ctx, authevent.Envelope{
		Kind:       authevent.KindUserLoggedIn,
		UserID:     referee.UserID,
		OccurredAt: now,
		Payload: authevent.UserLoggedIn{
			Identity: authdomain.Identity{UserID: referee.UserID, Email: referee.Email, IsNew: true},
			Provider: authdomain.ProviderEmail,
		},
	})

	stored, err := modules.Referral.Deps.Referrals.FindByReferee(context.Background(), referee.UserID)
	if err != nil {
		t.Fatalf("find referral after login retry: %v", err)
	}
	if stored.ReferrerID != referrer.UserID || stored.Code != code.Value {
		t.Fatalf("stored referral=%+v, want original code owner", stored)
	}
	if deliveryAttempts != 4 {
		t.Fatalf("registered delivery attempts=%d, want three signup attempts plus login recovery", deliveryAttempts)
	}

	returningUser, err := authStore.FindOrCreateByEmail(context.Background(), "returning-user@example.com")
	if err != nil {
		t.Fatalf("create returning user: %v", err)
	}
	modules.Auth.Deps.Bus.Publish(ctx, authevent.Envelope{
		Kind:       authevent.KindUserLoggedIn,
		UserID:     returningUser.UserID,
		OccurredAt: now.Add(time.Minute),
		Payload: authevent.UserLoggedIn{
			Identity: authdomain.Identity{UserID: returningUser.UserID, Email: returningUser.Email, IsNew: false},
			Provider: authdomain.ProviderEmail,
		},
	})
	if _, err := modules.Referral.Deps.Referrals.FindByReferee(context.Background(), returningUser.UserID); !errors.Is(err, referraldomain.ErrNotFound) {
		t.Fatalf("returning user referral lookup error=%v, want ErrNotFound", err)
	}
	if deliveryAttempts != 4 {
		t.Fatalf("registered delivery attempts=%d after returning login, want unchanged", deliveryAttempts)
	}
}
