package repo

import (
	"context"
	"testing"
	"time"

	billingdomain "github.com/brizenchi/go-modules/modules/billing/domain"
	usergormrepo "github.com/brizenchi/go-modules/modules/user/adapter/gormrepo"
	userdomain "github.com/brizenchi/go-modules/modules/user/domain"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newLegacySyncTestDB(t *testing.T) *gorm.DB {
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

func TestBackfillLegacyStripeStateCopiesLegacyUsersFields(t *testing.T) {
	db := newLegacySyncTestDB(t)
	users := usergormrepo.New(db)
	ctx := context.Background()

	start := time.Now().UTC()
	end := start.Add(30 * 24 * time.Hour)
	user := &userdomain.User{
		ID:                   "u-backfill",
		Email:                "backfill@example.com",
		Plan:                 userdomain.PlanPro,
		StripeCustomerID:     "cus_backfill",
		StripeSubscriptionID: "sub_backfill",
		StripePriceID:        "price_backfill",
		StripeProductID:      "prod_backfill",
		BillingStatus:        string(billingdomain.StatusActive),
		BillingPeriodStart:   &start,
		BillingPeriodEnd:     &end,
	}
	if err := users.Create(ctx, user); err != nil {
		t.Fatalf("Create: %v", err)
	}

	report, err := BackfillLegacyStripeState(ctx, db, LegacyBillingSyncOptions{})
	if err != nil {
		t.Fatalf("BackfillLegacyStripeState: %v", err)
	}
	if report.Scanned != 1 || report.CustomersSynced != 1 || report.SubscriptionsSynced != 1 {
		t.Fatalf("unexpected report: %+v", report)
	}

	customer, err := NewCustomerStore(db).LoadCustomer(ctx, "u-backfill")
	if err != nil {
		t.Fatalf("LoadCustomer: %v", err)
	}
	if customer.ProviderCustomerID != "cus_backfill" || customer.ProviderSubscriptionID != "sub_backfill" {
		t.Fatalf("unexpected customer row: %+v", customer)
	}

	sub, err := NewSubscriptionRepo(db).FindByUser(ctx, "u-backfill")
	if err != nil {
		t.Fatalf("FindByUser: %v", err)
	}
	if sub.ProviderPriceID != "price_backfill" || sub.Status != string(billingdomain.StatusActive) {
		t.Fatalf("unexpected subscription row: %+v", sub)
	}
}

func TestCheckLegacyStripeStateReportsMissingThenPassesAfterBackfill(t *testing.T) {
	db := newLegacySyncTestDB(t)
	users := usergormrepo.New(db)
	ctx := context.Background()

	user := &userdomain.User{
		ID:               "u-check",
		Email:            "check@example.com",
		Plan:             userdomain.PlanLifetime,
		StripeCustomerID: "cus_check",
		StripePriceID:    "price_lifetime",
		BillingStatus:    string(billingdomain.StatusActive),
	}
	if err := users.Create(ctx, user); err != nil {
		t.Fatalf("Create: %v", err)
	}

	check, err := CheckLegacyStripeState(ctx, db, LegacyBillingSyncOptions{})
	if err != nil {
		t.Fatalf("CheckLegacyStripeState: %v", err)
	}
	if check.OK() {
		t.Fatalf("expected missing-row issues, got %+v", check)
	}

	if _, err := BackfillLegacyStripeState(ctx, db, LegacyBillingSyncOptions{}); err != nil {
		t.Fatalf("BackfillLegacyStripeState: %v", err)
	}

	check, err = CheckLegacyStripeState(ctx, db, LegacyBillingSyncOptions{})
	if err != nil {
		t.Fatalf("CheckLegacyStripeState after backfill: %v", err)
	}
	if !check.OK() {
		t.Fatalf("expected no issues after backfill, got %+v", check)
	}
}
