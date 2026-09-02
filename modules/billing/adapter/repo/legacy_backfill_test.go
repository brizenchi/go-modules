package repo

import (
	"context"
	"testing"
	"time"

	billingdomain "github.com/brizenchi/go-modules/modules/billing/domain"
	billingport "github.com/brizenchi/go-modules/modules/billing/port"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type legacyUserFixture struct {
	ID                   string `gorm:"primaryKey"`
	Email                string
	Plan                 string
	StripeCustomerID     string
	StripeSubscriptionID string
	StripePriceID        string
	StripeProductID      string
	BillingStatus        string
	BillingPeriodStart   *time.Time
	BillingPeriodEnd     *time.Time
	CancelEffectiveAt    *time.Time
	CreatedAt            time.Time
}

func (legacyUserFixture) TableName() string { return "users" }

func newLegacySyncTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&legacyUserFixture{}); err != nil {
		t.Fatalf("migrate users: %v", err)
	}
	if err := AutoMigrate(db); err != nil {
		t.Fatalf("migrate billing: %v", err)
	}
	return db
}

func TestBackfillLegacyStripeStateCopiesLegacyUsersFields(t *testing.T) {
	db := newLegacySyncTestDB(t)
	ctx := context.Background()

	start := time.Now().UTC()
	end := start.Add(30 * 24 * time.Hour)
	user := &legacyUserFixture{
		ID:                   "u-backfill",
		Email:                "backfill@example.com",
		Plan:                 string(billingdomain.PlanPro),
		StripeCustomerID:     "cus_backfill",
		StripeSubscriptionID: "sub_backfill",
		StripePriceID:        "price_backfill",
		StripeProductID:      "prod_backfill",
		BillingStatus:        string(billingdomain.StatusActive),
		BillingPeriodStart:   &start,
		BillingPeriodEnd:     &end,
	}
	if err := db.WithContext(ctx).Create(user).Error; err != nil {
		t.Fatalf("Create: %v", err)
	}

	report, err := BackfillLegacyStripeState(ctx, db, LegacyBillingSyncOptions{})
	if err != nil {
		t.Fatalf("BackfillLegacyStripeState: %v", err)
	}
	if report.Scanned != 1 || report.CustomersSynced != 1 || report.SubscriptionsSynced != 1 {
		t.Fatalf("unexpected report: %+v", report)
	}

	lookup := newAccountLookupStub(billingport.Account{UserID: user.ID, Email: user.Email})
	customer, err := NewCustomerStore(db, lookup).LoadCustomer(ctx, "u-backfill")
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
	ctx := context.Background()

	user := &legacyUserFixture{
		ID:               "u-check",
		Email:            "check@example.com",
		Plan:             string(billingdomain.PlanLifetime),
		StripeCustomerID: "cus_check",
		StripePriceID:    "price_lifetime",
		BillingStatus:    string(billingdomain.StatusActive),
	}
	if err := db.WithContext(ctx).Create(user).Error; err != nil {
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
