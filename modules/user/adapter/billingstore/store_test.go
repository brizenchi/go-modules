package billingstore

import (
	"context"
	"testing"
	"time"

	billingdomain "github.com/brizenchi/go-modules/modules/billing/domain"
	billingport "github.com/brizenchi/go-modules/modules/billing/port"
	"github.com/brizenchi/go-modules/modules/user/adapter/gormrepo"
	userdomain "github.com/brizenchi/go-modules/modules/user/domain"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newBillingTestRepo(t *testing.T) *gormrepo.Repo {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := gormrepo.AutoMigrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return gormrepo.New(db)
}

func TestUserResolverByCustomerID(t *testing.T) {
	users := newBillingTestRepo(t)
	ctx := context.Background()
	user := &userdomain.User{
		Email:            "billing@example.com",
		StripeCustomerID: "cus_123",
	}
	if err := users.Create(ctx, user); err != nil {
		t.Fatalf("Create: %v", err)
	}

	resolver := NewUserResolver(users)
	userID, err := resolver.Resolve(ctx, billingport.UserHint{ProviderCustomerID: "cus_123"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if userID != user.ID {
		t.Fatalf("userID = %q, want %q", userID, user.ID)
	}
}

func TestApplySubscriptionSnapshot(t *testing.T) {
	users := newBillingTestRepo(t)
	ctx := context.Background()
	user := &userdomain.User{Email: "snapshot@example.com"}
	if err := users.Create(ctx, user); err != nil {
		t.Fatalf("Create: %v", err)
	}

	start := time.Now().UTC()
	end := start.Add(30 * 24 * time.Hour)
	err := ApplySubscriptionSnapshot(ctx, users, user.ID, billingdomain.SubscriptionSnapshot{
		ProviderSubscriptionID: "sub_123",
		ProviderCustomerID:     "cus_123",
		ProviderPriceID:        "price_123",
		ProviderProductID:      "prod_123",
		Plan:                   billingdomain.PlanPro,
		Status:                 billingdomain.StatusActive,
		PeriodStart:            &start,
		PeriodEnd:              &end,
	})
	if err != nil {
		t.Fatalf("ApplySubscriptionSnapshot: %v", err)
	}

	got, err := users.FindByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.Plan != string(billingdomain.PlanPro) {
		t.Fatalf("plan = %q, want %q", got.Plan, billingdomain.PlanPro)
	}
	if got.StripeSubscriptionID != "sub_123" {
		t.Fatalf("stripe_subscription_id = %q, want sub_123", got.StripeSubscriptionID)
	}
}
