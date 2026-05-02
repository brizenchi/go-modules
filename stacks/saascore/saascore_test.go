package saascore

import (
	"context"
	"strings"
	"testing"
	"time"

	authdomain "github.com/brizenchi/go-modules/modules/auth/domain"
	authevent "github.com/brizenchi/go-modules/modules/auth/event"
	billingdomain "github.com/brizenchi/go-modules/modules/billing/domain"
	"github.com/brizenchi/go-modules/modules/billing/event"
	referraldomain "github.com/brizenchi/go-modules/modules/referral/domain"
	referralevent "github.com/brizenchi/go-modules/modules/referral/event"
	userdomain "github.com/brizenchi/go-modules/modules/user/domain"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return db
}

func TestNewStack(t *testing.T) {
	db := newTestDB(t)
	stack, err := New(db, Config{
		ServiceName: "test-svc",
		Auth: AuthConfig{
			UserJWTSecret: "super-secret",
			EmailCode: EmailCodeConfig{
				Debug: true,
			},
		},
		Email: EmailConfig{Provider: "log"},
		Referral: ReferralConfig{
			BaseLink:         "http://localhost:3000/invite?ref=",
			ActivationReward: 50,
		},
	}, HostHooks{}, PolicyHooks{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if stack.Auth == nil || stack.Billing == nil || stack.Referral == nil || stack.Email == nil || stack.Users == nil {
		t.Fatalf("stack not fully initialized: %+v", stack)
	}
}

func TestNewStackRequiresJWTSecret(t *testing.T) {
	db := newTestDB(t)
	if _, err := New(db, Config{}, HostHooks{}, PolicyHooks{}); err == nil {
		t.Fatal("expected error for missing jwt secret")
	}
}

func TestNewStackRejectsHalfConfiguredGoogleOAuth(t *testing.T) {
	db := newTestDB(t)
	_, err := New(db, Config{
		Auth: AuthConfig{
			UserJWTSecret: "super-secret",
			Google: GoogleOAuthConfig{
				ClientID: "client-id-only",
			},
		},
		Email: EmailConfig{Provider: "log"},
	}, HostHooks{}, PolicyHooks{})
	if err == nil || !strings.Contains(err.Error(), "google oauth requires") {
		t.Fatalf("expected google oauth validation error, got %v", err)
	}
}

func TestNewStackAllowsDisabledGoogleOAuthWithOnlyTemplateDefaults(t *testing.T) {
	db := newTestDB(t)
	_, err := New(db, Config{
		Auth: AuthConfig{
			UserJWTSecret: "super-secret",
			Google: GoogleOAuthConfig{
				RedirectURL: "http://localhost:8080/api/v1/auth/google/callback",
				StateSecret: "dev-only-state-secret",
			},
		},
		Email: EmailConfig{Provider: "log"},
	}, HostHooks{}, PolicyHooks{})
	if err != nil {
		t.Fatalf("expected disabled google oauth defaults to be allowed, got %v", err)
	}
}

func TestNewStackRejectsStripeWithoutSecrets(t *testing.T) {
	db := newTestDB(t)
	_, err := New(db, Config{
		Auth: AuthConfig{
			UserJWTSecret: "super-secret",
		},
		Email: EmailConfig{Provider: "log"},
		Billing: BillingConfig{
			Stripe: StripeConfig{
				Enabled: true,
			},
		},
	}, HostHooks{}, PolicyHooks{})
	if err == nil || !strings.Contains(err.Error(), "billing.stripe.secret_key required") {
		t.Fatalf("expected stripe validation error, got %v", err)
	}
}

func TestNewStackRejectsBrevoWithoutSender(t *testing.T) {
	db := newTestDB(t)
	_, err := New(db, Config{
		Auth: AuthConfig{
			UserJWTSecret: "super-secret",
		},
		Email: EmailConfig{
			Provider: "brevo",
		},
	}, HostHooks{}, PolicyHooks{})
	if err == nil || !strings.Contains(err.Error(), "email brevo api key and sender email required") {
		t.Fatalf("expected brevo validation error, got %v", err)
	}
}

func TestNewStackRejectsResendWithoutSender(t *testing.T) {
	db := newTestDB(t)
	_, err := New(db, Config{
		Auth: AuthConfig{
			UserJWTSecret: "super-secret",
		},
		Email: EmailConfig{
			Provider: "resend",
		},
	}, HostHooks{}, PolicyHooks{})
	if err == nil || !strings.Contains(err.Error(), "email resend api key and sender email required") {
		t.Fatalf("expected resend validation error, got %v", err)
	}
}

func TestBillingReactivatedSyncsUserState(t *testing.T) {
	db := newTestDB(t)
	stack, err := New(db, Config{
		ServiceName: "test-svc",
		Auth: AuthConfig{
			UserJWTSecret: "super-secret",
			EmailCode:     EmailCodeConfig{Debug: true},
		},
		Email: EmailConfig{Provider: "log"},
	}, HostHooks{}, PolicyHooks{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	newUser := &userdomain.User{
		ID:    "u-1",
		Email: "u1@example.com",
		Plan:  userdomain.PlanFree,
		Role:  userdomain.RoleUser,
	}
	if err := stack.Users.Create(ctx, newUser); err != nil {
		t.Fatalf("create user: %v", err)
	}

	snapshotEnd := time.Now().UTC().Add(30 * 24 * time.Hour)
	if err := stack.onSubscriptionReactivated(ctx, event.Envelope{
		UserID: "u-1",
		Payload: event.SubscriptionReactivated{
			Snapshot: subscriptionSnapshot("cus_1", "sub_1", "price_pro", "pro", "active", snapshotEnd),
		},
	}); err != nil {
		t.Fatalf("onSubscriptionReactivated: %v", err)
	}

	got, err := stack.Users.FindByID(ctx, "u-1")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.BillingStatus != "active" || got.StripeSubscriptionID != "sub_1" {
		t.Fatalf("unexpected billing sync result: %+v", got)
	}
}

func TestReferralRewardPolicyHookOverridesDefault(t *testing.T) {
	db := newTestDB(t)
	stack, err := New(db, Config{
		ServiceName: "test-svc",
		Auth: AuthConfig{
			UserJWTSecret: "super-secret",
			EmailCode:     EmailCodeConfig{Debug: true},
		},
		Email: EmailConfig{Provider: "log"},
		Referral: ReferralConfig{
			ActivationReward: 50,
		},
	}, HostHooks{}, PolicyHooks{
		ResolveReferralReward: func(ctx context.Context, input ReferralRewardPolicyInput) (int, error) {
			return 120, nil
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	reward, err := stack.resolveReferralReward(context.Background(), ReferralRewardPolicyInput{
		ReferrerID: "u-referrer",
		RefereeID:  "u-referee",
	})
	if err != nil {
		t.Fatalf("resolveReferralReward: %v", err)
	}
	if reward != 120 {
		t.Fatalf("reward = %d, want 120", reward)
	}
}

func TestReferralRewardFallsBackToConfig(t *testing.T) {
	db := newTestDB(t)
	stack, err := New(db, Config{
		ServiceName: "test-svc",
		Auth: AuthConfig{
			UserJWTSecret: "super-secret",
			EmailCode:     EmailCodeConfig{Debug: true},
		},
		Email: EmailConfig{Provider: "log"},
		Referral: ReferralConfig{
			ActivationReward: 75,
		},
	}, HostHooks{}, PolicyHooks{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	reward, err := stack.resolveReferralReward(context.Background(), ReferralRewardPolicyInput{
		ReferrerID: "u-referrer",
		RefereeID:  "u-referee",
	})
	if err != nil {
		t.Fatalf("resolveReferralReward: %v", err)
	}
	if reward != 75 {
		t.Fatalf("reward = %d, want 75", reward)
	}
}

func TestReferralRewardRejectsNegativeValue(t *testing.T) {
	db := newTestDB(t)
	stack, err := New(db, Config{
		ServiceName: "test-svc",
		Auth: AuthConfig{
			UserJWTSecret: "super-secret",
			EmailCode:     EmailCodeConfig{Debug: true},
		},
		Email: EmailConfig{Provider: "log"},
	}, HostHooks{}, PolicyHooks{
		ResolveReferralReward: func(ctx context.Context, input ReferralRewardPolicyInput) (int, error) {
			return -1, nil
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = stack.resolveReferralReward(context.Background(), ReferralRewardPolicyInput{})
	if err == nil || !strings.Contains(err.Error(), "cannot be negative") {
		t.Fatalf("expected negative reward validation error, got %v", err)
	}
}

func TestUserSignedUpHostHookRuns(t *testing.T) {
	db := newTestDB(t)
	called := false
	stack, err := New(db, Config{
		ServiceName: "test-svc",
		Auth: AuthConfig{
			UserJWTSecret: "super-secret",
			EmailCode:     EmailCodeConfig{Debug: true},
		},
		Email: EmailConfig{Provider: "log"},
	}, HostHooks{
		OnUserSignedUp: func(ctx context.Context, event UserSignedUpEvent) error {
			called = event.UserID == "u-1"
			return nil
		},
	}, PolicyHooks{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	err = stack.onUserSignedUpHost(context.Background(), authevent.Envelope{
		UserID:     "u-1",
		OccurredAt: time.Now().UTC(),
		Payload: authevent.UserSignedUp{
			Identity: authdomain.Identity{UserID: "u-1", Email: "u1@example.com"},
		},
	})
	if err != nil {
		t.Fatalf("onUserSignedUpHost: %v", err)
	}
	if !called {
		t.Fatal("expected host signup hook to be called")
	}
}

func TestCreditsPurchasedHostHookRunsAndAddsCredits(t *testing.T) {
	db := newTestDB(t)
	called := false
	stack, err := New(db, Config{
		ServiceName: "test-svc",
		Auth: AuthConfig{
			UserJWTSecret: "super-secret",
			EmailCode:     EmailCodeConfig{Debug: true},
		},
		Email: EmailConfig{Provider: "log"},
	}, HostHooks{
		OnCreditsPurchased: func(ctx context.Context, event CreditsPurchasedEvent) error {
			called = event.UserID == "u-credits" && event.TotalCredits == 300
			return nil
		},
	}, PolicyHooks{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	if err := stack.Users.Create(ctx, &userdomain.User{
		ID:    "u-credits",
		Email: "credits@example.com",
		Role:  userdomain.RoleUser,
		Plan:  userdomain.PlanFree,
	}); err != nil {
		t.Fatalf("create user: %v", err)
	}

	err = stack.onCreditsPurchased(ctx, event.Envelope{
		UserID:          "u-credits",
		Provider:        "stripe",
		ProviderEventID: "evt_credit_1",
		OccurredAt:      time.Now().UTC(),
		Payload: event.CreditsPurchased{
			Quantity:       3,
			CreditsPerUnit: 100,
			TotalCredits:   300,
			PriceID:        "price_credits",
		},
	})
	if err != nil {
		t.Fatalf("onCreditsPurchased: %v", err)
	}
	if !called {
		t.Fatal("expected credits purchased hook to be called")
	}

	got, err := stack.Users.FindByID(ctx, "u-credits")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.Credits != 300 {
		t.Fatalf("credits = %d, want 300", got.Credits)
	}
}

func TestReferralRegisteredHostHookRuns(t *testing.T) {
	db := newTestDB(t)
	called := false
	stack, err := New(db, Config{
		ServiceName: "test-svc",
		Auth: AuthConfig{
			UserJWTSecret: "super-secret",
			EmailCode:     EmailCodeConfig{Debug: true},
		},
		Email: EmailConfig{Provider: "log"},
	}, HostHooks{
		OnReferralRegistered: func(ctx context.Context, event ReferralRegisteredEvent) error {
			called = event.Referral.ReferrerID == "u-referrer" && event.Referral.RefereeID == "u-referee"
			return nil
		},
	}, PolicyHooks{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	err = stack.onReferralRegisteredHost(context.Background(), referralevent.Envelope{
		OccurredAt: time.Now().UTC(),
		Payload: referralevent.ReferralRegistered{
			Referral: referraldomain.Referral{
				ReferrerID: "u-referrer",
				RefereeID:  "u-referee",
				Code:       "INV-123",
				Status:     referraldomain.StatusPending,
			},
		},
	})
	if err != nil {
		t.Fatalf("onReferralRegisteredHost: %v", err)
	}
	if !called {
		t.Fatal("expected referral registered hook to be called")
	}
}

func TestSubscriptionActivatedHostHookRuns(t *testing.T) {
	db := newTestDB(t)
	called := false
	stack, err := New(db, Config{
		ServiceName: "test-svc",
		Auth: AuthConfig{
			UserJWTSecret: "super-secret",
			EmailCode:     EmailCodeConfig{Debug: true},
		},
		Email: EmailConfig{Provider: "log"},
		Referral: ReferralConfig{
			ActivationReward: 50,
		},
	}, HostHooks{
		OnSubscriptionActivated: func(ctx context.Context, event SubscriptionEvent) error {
			called = event.UserID == "u-sub" && event.Snapshot.ProviderSubscriptionID == "sub_activated"
			return nil
		},
	}, PolicyHooks{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	if err := stack.Users.Create(ctx, &userdomain.User{
		ID:    "u-sub",
		Email: "sub@example.com",
		Role:  userdomain.RoleUser,
		Plan:  userdomain.PlanFree,
	}); err != nil {
		t.Fatalf("create user: %v", err)
	}

	end := time.Now().UTC().Add(30 * 24 * time.Hour)
	err = stack.onSubscriptionActivated(ctx, event.Envelope{
		UserID:          "u-sub",
		Provider:        "stripe",
		ProviderEventID: "evt_sub_1",
		OccurredAt:      time.Now().UTC(),
		Payload: event.SubscriptionActivated{
			Snapshot: billingdomain.SubscriptionSnapshot{
				ProviderCustomerID:     "cus_activated",
				ProviderSubscriptionID: "sub_activated",
				ProviderPriceID:        "price_pro",
				Plan:                   billingdomain.PlanPro,
				Status:                 billingdomain.StatusActive,
				PeriodEnd:              &end,
			},
		},
	})
	if err != nil {
		t.Fatalf("onSubscriptionActivated: %v", err)
	}
	if !called {
		t.Fatal("expected subscription activated hook to be called")
	}
}
