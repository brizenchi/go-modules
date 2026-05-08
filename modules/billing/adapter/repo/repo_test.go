package repo

import (
	"context"
	"testing"
	"time"

	billingdomain "github.com/brizenchi/go-modules/modules/billing/domain"
	billingport "github.com/brizenchi/go-modules/modules/billing/port"
	usergormrepo "github.com/brizenchi/go-modules/modules/user/adapter/gormrepo"
	userdomain "github.com/brizenchi/go-modules/modules/user/domain"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newBillingRepoTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := usergormrepo.AutoMigrate(db); err != nil {
		t.Fatalf("migrate users: %v", err)
	}
	if err := AutoMigrate(db); err != nil {
		t.Fatalf("migrate billing: %v", err)
	}
	return db
}

func seedBillingRepoUser(t *testing.T, db *gorm.DB, user *userdomain.User) *usergormrepo.Repo {
	t.Helper()
	repo := usergormrepo.New(db)
	if err := repo.Create(context.Background(), user); err != nil {
		t.Fatalf("Create: %v", err)
	}
	return repo
}

func TestCustomerStoreLoadCustomerWithoutBillingRows(t *testing.T) {
	db := newBillingRepoTestDB(t)
	ctx := context.Background()
	users := seedBillingRepoUser(t, db, &userdomain.User{
		Email: "reader@example.com",
		Plan:  userdomain.PlanPro,
	})

	u, err := users.FindByEmail(ctx, "reader@example.com")
	if err != nil {
		t.Fatalf("FindByEmail: %v", err)
	}

	store := NewCustomerStore(db)
	got, err := store.LoadCustomer(ctx, u.ID)
	if err != nil {
		t.Fatalf("LoadCustomer: %v", err)
	}
	if got.Email != "reader@example.com" {
		t.Fatalf("email = %q, want reader@example.com", got.Email)
	}
	if got.Plan != userdomain.PlanPro {
		t.Fatalf("plan = %q, want %q", got.Plan, userdomain.PlanPro)
	}
	if got.ProviderCustomerID != "" || got.ProviderSubscriptionID != "" {
		t.Fatalf("expected empty provider ids, got %+v", got)
	}
}

func TestCustomerStoreSaveAndLoadCustomer(t *testing.T) {
	db := newBillingRepoTestDB(t)
	ctx := context.Background()
	users := seedBillingRepoUser(t, db, &userdomain.User{
		Email: "billing@example.com",
		Plan:  userdomain.PlanStarter,
	})
	u, err := users.FindByEmail(ctx, "billing@example.com")
	if err != nil {
		t.Fatalf("FindByEmail: %v", err)
	}

	store := NewCustomerStore(db)
	if err := store.SaveCustomerID(ctx, u.ID, "stripe", "cus_123"); err != nil {
		t.Fatalf("SaveCustomerID: %v", err)
	}

	subscriptions := NewSubscriptionRepo(db)
	if err := subscriptions.UpsertSnapshot(ctx, u.ID, "stripe", billingdomain.SubscriptionSnapshot{
		ProviderCustomerID:     "cus_123",
		ProviderSubscriptionID: "sub_123",
		Plan:                   billingdomain.PlanStarter,
		Status:                 billingdomain.StatusActive,
	}); err != nil {
		t.Fatalf("UpsertSnapshot: %v", err)
	}

	got, err := store.LoadCustomer(ctx, u.ID)
	if err != nil {
		t.Fatalf("LoadCustomer: %v", err)
	}
	if got.ProviderCustomerID != "cus_123" {
		t.Fatalf("provider_customer_id = %q, want cus_123", got.ProviderCustomerID)
	}
	if got.ProviderSubscriptionID != "sub_123" {
		t.Fatalf("provider_subscription_id = %q, want sub_123", got.ProviderSubscriptionID)
	}
}

func TestSubscriptionRepoUpsertSnapshot(t *testing.T) {
	db := newBillingRepoTestDB(t)
	ctx := context.Background()
	users := seedBillingRepoUser(t, db, &userdomain.User{Email: "snapshot@example.com"})
	u, err := users.FindByEmail(ctx, "snapshot@example.com")
	if err != nil {
		t.Fatalf("FindByEmail: %v", err)
	}

	repo := NewSubscriptionRepo(db)
	start := time.Now().UTC()
	end := start.Add(30 * 24 * time.Hour)
	if err := repo.UpsertSnapshot(ctx, u.ID, "stripe", billingdomain.SubscriptionSnapshot{
		ProviderCustomerID:     "cus_1",
		ProviderSubscriptionID: "sub_1",
		ProviderPriceID:        "price_1",
		ProviderProductID:      "prod_1",
		ProductType:            billingdomain.ProductSubscription,
		Plan:                   billingdomain.PlanPro,
		Interval:               billingdomain.IntervalMonthly,
		Status:                 billingdomain.StatusActive,
		PeriodStart:            &start,
		PeriodEnd:              &end,
	}); err != nil {
		t.Fatalf("UpsertSnapshot create: %v", err)
	}

	if err := repo.UpsertSnapshot(ctx, u.ID, "stripe", billingdomain.SubscriptionSnapshot{
		ProviderCustomerID:     "cus_1",
		ProviderSubscriptionID: "sub_1",
		ProviderPriceID:        "price_2",
		ProviderProductID:      "prod_2",
		ProductType:            billingdomain.ProductSubscription,
		Plan:                   billingdomain.PlanPremium,
		Interval:               billingdomain.IntervalYearly,
		Status:                 billingdomain.StatusCanceling,
		PeriodStart:            &start,
		PeriodEnd:              &end,
		CancelAtPeriodEnd:      true,
	}); err != nil {
		t.Fatalf("UpsertSnapshot update: %v", err)
	}

	got, err := repo.FindByUser(ctx, u.ID)
	if err != nil {
		t.Fatalf("FindByUser: %v", err)
	}
	if got.ProviderPriceID != "price_2" {
		t.Fatalf("provider_price_id = %q, want price_2", got.ProviderPriceID)
	}
	if got.Plan != string(billingdomain.PlanPremium) {
		t.Fatalf("plan = %q, want %q", got.Plan, billingdomain.PlanPremium)
	}
	if got.BillingInterval != string(billingdomain.IntervalYearly) {
		t.Fatalf("billing_interval = %q, want %q", got.BillingInterval, billingdomain.IntervalYearly)
	}

	var count int64
	if err := db.Model(&billingdomain.BillingSubscription{}).Where("user_id = ?", u.ID).Count(&count).Error; err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 1 {
		t.Fatalf("subscription row count = %d, want 1", count)
	}
}

func TestUserResolverUsesBillingTablesThenEmailFallback(t *testing.T) {
	db := newBillingRepoTestDB(t)
	ctx := context.Background()
	users := seedBillingRepoUser(t, db, &userdomain.User{Email: "resolve@example.com"})
	u, err := users.FindByEmail(ctx, "resolve@example.com")
	if err != nil {
		t.Fatalf("FindByEmail: %v", err)
	}

	customers := NewCustomerStore(db)
	if err := customers.SaveCustomerID(ctx, u.ID, "stripe", "cus_resolve"); err != nil {
		t.Fatalf("SaveCustomerID: %v", err)
	}
	subs := NewSubscriptionRepo(db)
	if err := subs.UpsertSnapshot(ctx, u.ID, "stripe", billingdomain.SubscriptionSnapshot{
		ProviderCustomerID:     "cus_resolve",
		ProviderSubscriptionID: "sub_resolve",
		Plan:                   billingdomain.PlanStarter,
		Status:                 billingdomain.StatusActive,
	}); err != nil {
		t.Fatalf("UpsertSnapshot: %v", err)
	}

	resolver := NewUserResolver(db)

	userID, err := resolver.Resolve(ctx, billingport.UserHint{ProviderCustomerID: "cus_resolve"})
	if err != nil {
		t.Fatalf("Resolve by customer: %v", err)
	}
	if userID != u.ID {
		t.Fatalf("customer user_id = %q, want %q", userID, u.ID)
	}

	userID, err = resolver.Resolve(ctx, billingport.UserHint{ProviderSubscriptionID: "sub_resolve"})
	if err != nil {
		t.Fatalf("Resolve by subscription: %v", err)
	}
	if userID != u.ID {
		t.Fatalf("subscription user_id = %q, want %q", userID, u.ID)
	}

	userID, err = resolver.Resolve(ctx, billingport.UserHint{Email: "resolve@example.com"})
	if err != nil {
		t.Fatalf("Resolve by email: %v", err)
	}
	if userID != u.ID {
		t.Fatalf("email user_id = %q, want %q", userID, u.ID)
	}
}
