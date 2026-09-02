package repo

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	billingdomain "github.com/brizenchi/go-modules/modules/billing/domain"
	billingport "github.com/brizenchi/go-modules/modules/billing/port"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type accountLookupStub struct {
	byID    map[string]billingport.Account
	byEmail map[string]string
}

func newAccountLookupStub(accounts ...billingport.Account) *accountLookupStub {
	lookup := &accountLookupStub{
		byID:    make(map[string]billingport.Account, len(accounts)),
		byEmail: make(map[string]string, len(accounts)),
	}
	for _, account := range accounts {
		lookup.byID[account.UserID] = account
		lookup.byEmail[strings.ToLower(account.Email)] = account.UserID
	}
	return lookup
}

func (s *accountLookupStub) FindBillingAccount(_ context.Context, userID string) (billingport.Account, error) {
	account, ok := s.byID[strings.TrimSpace(userID)]
	if !ok {
		return billingport.Account{}, fmt.Errorf("account not found")
	}
	return account, nil
}

func (s *accountLookupStub) FindUserIDByEmail(_ context.Context, email string) (string, error) {
	userID, ok := s.byEmail[strings.ToLower(strings.TrimSpace(email))]
	if !ok {
		return "", fmt.Errorf("account not found")
	}
	return userID, nil
}

func newBillingRepoTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := AutoMigrate(db); err != nil {
		t.Fatalf("migrate billing: %v", err)
	}
	return db
}

func TestCustomerStoreLoadCustomerWithoutBillingRows(t *testing.T) {
	db := newBillingRepoTestDB(t)
	ctx := context.Background()
	lookup := newAccountLookupStub(billingport.Account{UserID: "u-reader", Email: "reader@example.com"})

	store := NewCustomerStore(db, lookup)
	got, err := store.LoadCustomer(ctx, "u-reader")
	if err != nil {
		t.Fatalf("LoadCustomer: %v", err)
	}
	if got.Email != "reader@example.com" {
		t.Fatalf("email = %q, want reader@example.com", got.Email)
	}
	if got.Plan != string(billingdomain.PlanFree) {
		t.Fatalf("plan = %q, want %q", got.Plan, billingdomain.PlanFree)
	}
	if got.ProviderCustomerID != "" || got.ProviderSubscriptionID != "" {
		t.Fatalf("expected empty provider ids, got %+v", got)
	}
}

func TestCustomerStoreSaveAndLoadCustomer(t *testing.T) {
	db := newBillingRepoTestDB(t)
	ctx := context.Background()
	lookup := newAccountLookupStub(billingport.Account{UserID: "u-billing", Email: "billing@example.com"})

	store := NewCustomerStore(db, lookup)
	if err := store.SaveCustomerID(ctx, "u-billing", "stripe", "cus_123"); err != nil {
		t.Fatalf("SaveCustomerID: %v", err)
	}

	subscriptions := NewSubscriptionRepo(db)
	if err := subscriptions.UpsertSnapshot(ctx, "u-billing", "stripe", billingdomain.SubscriptionSnapshot{
		ProviderCustomerID:     "cus_123",
		ProviderSubscriptionID: "sub_123",
		Plan:                   billingdomain.PlanStarter,
		Status:                 billingdomain.StatusActive,
	}); err != nil {
		t.Fatalf("UpsertSnapshot: %v", err)
	}

	got, err := store.LoadCustomer(ctx, "u-billing")
	if err != nil {
		t.Fatalf("LoadCustomer: %v", err)
	}
	if got.ProviderCustomerID != "cus_123" {
		t.Fatalf("provider_customer_id = %q, want cus_123", got.ProviderCustomerID)
	}
	if got.ProviderSubscriptionID != "sub_123" {
		t.Fatalf("provider_subscription_id = %q, want sub_123", got.ProviderSubscriptionID)
	}
	if got.Plan != string(billingdomain.PlanStarter) {
		t.Fatalf("plan = %q, want %q", got.Plan, billingdomain.PlanStarter)
	}
}

func TestSubscriptionRepoUpsertSnapshot(t *testing.T) {
	db := newBillingRepoTestDB(t)
	ctx := context.Background()
	userID := "u-snapshot"

	repo := NewSubscriptionRepo(db)
	start := time.Now().UTC()
	end := start.Add(30 * 24 * time.Hour)
	if err := repo.UpsertSnapshot(ctx, userID, "stripe", billingdomain.SubscriptionSnapshot{
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

	if err := repo.UpsertSnapshot(ctx, userID, "stripe", billingdomain.SubscriptionSnapshot{
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

	got, err := repo.FindByUser(ctx, userID)
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
	if err := db.Model(&billingdomain.BillingSubscription{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 1 {
		t.Fatalf("subscription row count = %d, want 1", count)
	}
}

func TestUserResolverUsesBillingTablesThenEmailFallback(t *testing.T) {
	db := newBillingRepoTestDB(t)
	ctx := context.Background()
	lookup := newAccountLookupStub(billingport.Account{UserID: "u-resolve", Email: "resolve@example.com"})

	customers := NewCustomerStore(db, lookup)
	if err := customers.SaveCustomerID(ctx, "u-resolve", "stripe", "cus_resolve"); err != nil {
		t.Fatalf("SaveCustomerID: %v", err)
	}
	subs := NewSubscriptionRepo(db)
	if err := subs.UpsertSnapshot(ctx, "u-resolve", "stripe", billingdomain.SubscriptionSnapshot{
		ProviderCustomerID:     "cus_resolve",
		ProviderSubscriptionID: "sub_resolve",
		Plan:                   billingdomain.PlanStarter,
		Status:                 billingdomain.StatusActive,
	}); err != nil {
		t.Fatalf("UpsertSnapshot: %v", err)
	}

	resolver := NewUserResolver(db, lookup)

	userID, err := resolver.Resolve(ctx, billingport.UserHint{ProviderCustomerID: "cus_resolve"})
	if err != nil {
		t.Fatalf("Resolve by customer: %v", err)
	}
	if userID != "u-resolve" {
		t.Fatalf("customer user_id = %q, want u-resolve", userID)
	}

	userID, err = resolver.Resolve(ctx, billingport.UserHint{ProviderSubscriptionID: "sub_resolve"})
	if err != nil {
		t.Fatalf("Resolve by subscription: %v", err)
	}
	if userID != "u-resolve" {
		t.Fatalf("subscription user_id = %q, want u-resolve", userID)
	}

	userID, err = resolver.Resolve(ctx, billingport.UserHint{Email: "resolve@example.com"})
	if err != nil {
		t.Fatalf("Resolve by email: %v", err)
	}
	if userID != "u-resolve" {
		t.Fatalf("email user_id = %q, want u-resolve", userID)
	}
}
