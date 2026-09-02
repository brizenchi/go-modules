package bootstrap

import (
	"context"
	"testing"
	"time"

	authdomain "github.com/brizenchi/go-modules/modules/auth/domain"
	authevent "github.com/brizenchi/go-modules/modules/auth/event"
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
			UserJWTSecret: "test-jwt-secret",
			Email:         platform.AuthEmailConfig{Enabled: boolPtr(false)},
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
	modules.Referral.Deps.Bus.Publish(context.Background(), referralevent.Envelope{
		Kind:       referralevent.KindReferralActivated,
		OccurredAt: now,
		Payload: referralevent.ReferralActivated{Referral: referraldomain.Referral{
			ReferrerID:    identity.UserID,
			RefereeID:     "referee",
			RewardCredits: 50,
		}},
	})

	got, err := modules.Users.FindByID(context.Background(), identity.UserID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.Credits != 170 {
		t.Fatalf("credits=%d, want signup 20 + purchase 100 + referral 50", got.Credits)
	}
}
