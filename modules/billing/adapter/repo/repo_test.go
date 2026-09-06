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
	if _, err := subscriptions.UpsertSnapshot(ctx, "u-billing", "stripe", billingdomain.SubscriptionSnapshot{
		ProviderCustomerID:     "cus_123",
		ProviderSubscriptionID: "sub_123",
		Plan:                   billingdomain.PlanStarter,
		Status:                 billingdomain.StatusActive,
	}, time.Unix(100, 0), "evt_customer_load"); err != nil {
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
	if got.SubscriptionStatus != billingdomain.StatusActive {
		t.Fatalf("subscription status = %q, want %q", got.SubscriptionStatus, billingdomain.StatusActive)
	}
}

func TestCustomerStoreCheckoutReservationSerializesPaidIntents(t *testing.T) {
	db := newBillingRepoTestDB(t)
	store := NewCustomerStore(db, nil)
	ctx := context.Background()
	now := time.Unix(100, 0).UTC()

	first, acquired, err := store.ReserveCheckout(ctx, "u-reserve", "stripe", "cus_reserve", "reservation-1", "intent-1", now, now.Add(time.Hour))
	if err != nil || !acquired {
		t.Fatalf("first reservation acquired=%t err=%v", acquired, err)
	}
	if first.ProviderCustomerID != "cus_reserve" {
		t.Fatalf("provider_customer_id=%q, want cus_reserve", first.ProviderCustomerID)
	}
	second, acquired, err := store.ReserveCheckout(ctx, "u-reserve", "stripe", "cus_reserve", "reservation-2", "intent-2", now.Add(time.Minute), now.Add(2*time.Hour))
	if err != nil {
		t.Fatalf("second reservation: %v", err)
	}
	if acquired || second.ReservationID != first.ReservationID || second.IntentKey != "intent-1" {
		t.Fatalf("competing intent acquired reservation: acquired=%t row=%+v", acquired, second)
	}

	checkout := billingdomain.CheckoutResult{SessionID: "cs_1", CheckoutURL: "https://checkout.test/cs_1"}
	if err := store.CompleteCheckoutReservation(ctx, "u-reserve", "stripe", "reservation-1", checkout); err != nil {
		t.Fatalf("complete reservation: %v", err)
	}
	existing, acquired, err := store.ReserveCheckout(ctx, "u-reserve", "stripe", "cus_reserve", "reservation-3", "intent-1", now.Add(2*time.Minute), now.Add(2*time.Hour))
	if err != nil || acquired {
		t.Fatalf("same active reservation acquired=%t err=%v", acquired, err)
	}
	if existing.SessionID != checkout.SessionID || existing.CheckoutURL != checkout.CheckoutURL {
		t.Fatalf("completed reservation = %+v", existing)
	}

	stillReserved, acquired, err := store.ReserveCheckout(ctx, "u-reserve", "stripe", "cus_reserve", "reservation-4", "intent-4", now.Add(2*time.Hour), now.Add(3*time.Hour))
	if err != nil || acquired {
		t.Fatalf("elapsed reservation acquired=%t err=%v", acquired, err)
	}
	if stillReserved.ReservationID != "reservation-1" {
		t.Fatalf("elapsed reservation was replaced without provider proof: %+v", stillReserved)
	}
	replaced, err := store.ReplaceCheckoutReservation(ctx, "u-reserve", "stripe", "cus_reserve", "reservation-1", "reservation-4", "intent-4", now.Add(3*time.Hour))
	if err != nil || !replaced {
		t.Fatalf("CAS replacement replaced=%t err=%v", replaced, err)
	}
	// A concurrent contender holding the same stale proof must lose after the
	// first CAS changes the reservation ID.
	replaced, err = store.ReplaceCheckoutReservation(ctx, "u-reserve", "stripe", "cus_reserve", "reservation-1", "reservation-5", "intent-5", now.Add(3*time.Hour))
	if err != nil || replaced {
		t.Fatalf("stale CAS replaced=%t err=%v", replaced, err)
	}
}

func TestCustomerStoreCheckoutReservationLinkAndConditionalRelease(t *testing.T) {
	db := newBillingRepoTestDB(t)
	store := NewCustomerStore(db, nil)
	ctx := context.Background()
	now := time.Now().UTC()
	if _, acquired, err := store.ReserveCheckout(ctx, "u-release", "stripe", "cus_release", "reservation-1", "intent-1", now, now.Add(time.Hour)); err != nil || !acquired {
		t.Fatalf("ReserveCheckout acquired=%t err=%v", acquired, err)
	}
	if err := store.CompleteCheckoutReservation(ctx, "u-release", "stripe", "reservation-1", billingdomain.CheckoutResult{SessionID: "cs_1", CheckoutURL: "https://checkout.test/cs_1"}); err != nil {
		t.Fatalf("CompleteCheckoutReservation: %v", err)
	}
	if err := store.LinkCheckoutSubscription(ctx, "stripe", "cs_1", "sub_1"); err != nil {
		t.Fatalf("LinkCheckoutSubscription: %v", err)
	}
	if released, err := store.ReleaseCheckoutReservation(ctx, "stripe", "cs_other"); err != nil || released {
		t.Fatalf("unrelated session released=%t err=%v", released, err)
	}
	if released, err := store.ReleaseCheckoutReservationBySubscription(ctx, "stripe", "sub_1"); err != nil || !released {
		t.Fatalf("subscription release released=%t err=%v", released, err)
	}
}

func TestCustomerStoreReleasesSessionlessReservationByReservationID(t *testing.T) {
	db := newBillingRepoTestDB(t)
	store := NewCustomerStore(db, nil)
	ctx := context.Background()
	now := time.Now().UTC()
	if _, acquired, err := store.ReserveCheckout(ctx, "u-sessionless", "stripe", "cus_sessionless", "reservation-sessionless", "intent-1", now, now.Add(time.Hour)); err != nil || !acquired {
		t.Fatalf("ReserveCheckout acquired=%t err=%v", acquired, err)
	}
	if released, err := store.ReleaseCheckoutReservationByReservationID(ctx, "stripe", "reservation-other"); err != nil || released {
		t.Fatalf("unrelated release=%t err=%v", released, err)
	}
	if released, err := store.ReleaseCheckoutReservationByReservationID(ctx, "stripe", "reservation-sessionless"); err != nil || !released {
		t.Fatalf("matching release=%t err=%v", released, err)
	}
}

func TestSubscriptionRepoUpsertSnapshot(t *testing.T) {
	db := newBillingRepoTestDB(t)
	ctx := context.Background()
	userID := "u-snapshot"

	repo := NewSubscriptionRepo(db)
	start := time.Now().UTC()
	end := start.Add(30 * 24 * time.Hour)
	if _, err := repo.UpsertSnapshot(ctx, userID, "stripe", billingdomain.SubscriptionSnapshot{
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
	}, time.Unix(100, 0), "evt_snapshot_create"); err != nil {
		t.Fatalf("UpsertSnapshot create: %v", err)
	}

	if _, err := repo.UpsertSnapshot(ctx, userID, "stripe", billingdomain.SubscriptionSnapshot{
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
	}, time.Unix(200, 0), "evt_snapshot_update"); err != nil {
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

func TestSubscriptionRepoRejectsOutOfOrderSnapshot(t *testing.T) {
	db := newBillingRepoTestDB(t)
	ctx := context.Background()
	repo := NewSubscriptionRepo(db)
	userID := "u-out-of-order"

	applied, err := repo.UpsertSnapshot(ctx, userID, "stripe", billingdomain.SubscriptionSnapshot{
		ProviderSubscriptionID: "sub_new",
		Plan:                   billingdomain.PlanPro,
		Status:                 billingdomain.StatusActive,
	}, time.Unix(200, 0), "evt_new")
	if err != nil || !applied {
		t.Fatalf("new snapshot applied=%t err=%v", applied, err)
	}
	applied, err = repo.UpsertSnapshot(ctx, userID, "stripe", billingdomain.SubscriptionSnapshot{
		ProviderSubscriptionID: "sub_old",
		Plan:                   billingdomain.PlanStarter,
		Status:                 billingdomain.StatusCanceled,
	}, time.Unix(100, 0), "evt_old")
	if err != nil {
		t.Fatalf("old snapshot: %v", err)
	}
	if applied {
		t.Fatal("older snapshot should not be applied")
	}

	got, err := repo.FindByUser(ctx, userID)
	if err != nil {
		t.Fatalf("FindByUser: %v", err)
	}
	if got.ProviderSubscriptionID != "sub_new" || got.Status != string(billingdomain.StatusActive) || got.SnapshotEventID != "evt_new" {
		t.Fatalf("snapshot regressed: %+v", got)
	}
}

func TestSubscriptionRepoPreservesLifetimeEntitlement(t *testing.T) {
	db := newBillingRepoTestDB(t)
	ctx := context.Background()
	repo := NewSubscriptionRepo(db)
	userID := "u-lifetime-monotonic"

	applied, err := repo.UpsertSnapshot(ctx, userID, "stripe", billingdomain.SubscriptionSnapshot{
		ProviderCustomerID: "cus_lifetime",
		ProductType:        billingdomain.ProductSubscription,
		Plan:               billingdomain.PlanPro,
		Status:             billingdomain.StatusActive,
	}, time.Unix(200, 0), "evt_recurring")
	if err != nil || !applied {
		t.Fatalf("recurring snapshot applied=%t err=%v", applied, err)
	}
	// A lifetime payment is an irreversible entitlement and wins even if its
	// delivery is older than the recurring snapshot already observed.
	applied, err = repo.UpsertSnapshot(ctx, userID, "stripe", billingdomain.SubscriptionSnapshot{
		ProviderCustomerID: "cus_lifetime",
		ProductType:        billingdomain.ProductLifetime,
		Plan:               billingdomain.PlanLifetime,
		Status:             billingdomain.StatusActive,
	}, time.Unix(100, 0), "evt_lifetime")
	if err != nil || !applied {
		t.Fatalf("lifetime snapshot applied=%t err=%v", applied, err)
	}
	// Even a newer recurring webhook cannot downgrade lifetime access.
	applied, err = repo.UpsertSnapshot(ctx, userID, "stripe", billingdomain.SubscriptionSnapshot{
		ProviderCustomerID:     "cus_lifetime",
		ProviderSubscriptionID: "sub_late",
		ProductType:            billingdomain.ProductSubscription,
		Plan:                   billingdomain.PlanPremium,
		Status:                 billingdomain.StatusActive,
	}, time.Unix(300, 0), "evt_late_recurring")
	if err != nil {
		t.Fatalf("late recurring snapshot: %v", err)
	}
	if applied {
		t.Fatal("recurring snapshot must not replace lifetime entitlement")
	}

	got, err := repo.FindByUser(ctx, userID)
	if err != nil {
		t.Fatalf("FindByUser: %v", err)
	}
	if got.Plan != string(billingdomain.PlanLifetime) || got.SnapshotEventID != "evt_lifetime" {
		t.Fatalf("lifetime entitlement was lost: %+v", got)
	}
}

func TestSubscriptionRepoEqualTimestampPrefersSafeState(t *testing.T) {
	db := newBillingRepoTestDB(t)
	ctx := context.Background()
	repo := NewSubscriptionRepo(db)
	at := time.Unix(200, 0)

	// A late cancellation for an old subscription must not erase a different
	// active subscription created during the same Stripe timestamp second.
	if applied, err := repo.UpsertSnapshot(ctx, "u-equal-new", "stripe", billingdomain.SubscriptionSnapshot{
		ProviderSubscriptionID: "sub_new",
		Plan:                   billingdomain.PlanPro,
		Status:                 billingdomain.StatusActive,
	}, at, "evt_new_active"); err != nil || !applied {
		t.Fatalf("new active applied=%t err=%v", applied, err)
	}
	if applied, err := repo.UpsertSnapshot(ctx, "u-equal-new", "stripe", billingdomain.SubscriptionSnapshot{
		ProviderSubscriptionID: "sub_old",
		Plan:                   billingdomain.PlanStarter,
		Status:                 billingdomain.StatusCanceled,
	}, at, "evt_old_canceled"); err != nil || applied {
		t.Fatalf("old cancellation applied=%t err=%v", applied, err)
	}

	// For the same subscription ID, cancellation is terminal and wins ties.
	if applied, err := repo.UpsertSnapshot(ctx, "u-equal-same", "stripe", billingdomain.SubscriptionSnapshot{
		ProviderSubscriptionID: "sub_same",
		Plan:                   billingdomain.PlanPro,
		Status:                 billingdomain.StatusActive,
	}, at, "evt_same_active"); err != nil || !applied {
		t.Fatalf("same active applied=%t err=%v", applied, err)
	}
	if applied, err := repo.UpsertSnapshot(ctx, "u-equal-same", "stripe", billingdomain.SubscriptionSnapshot{
		ProviderSubscriptionID: "sub_same",
		Plan:                   billingdomain.PlanPro,
		Status:                 billingdomain.StatusCanceled,
	}, at, "evt_same_canceled"); err != nil || !applied {
		t.Fatalf("same cancellation applied=%t err=%v", applied, err)
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
	if _, err := subs.UpsertSnapshot(ctx, "u-resolve", "stripe", billingdomain.SubscriptionSnapshot{
		ProviderCustomerID:     "cus_resolve",
		ProviderSubscriptionID: "sub_resolve",
		Plan:                   billingdomain.PlanStarter,
		Status:                 billingdomain.StatusActive,
	}, time.Unix(100, 0), "evt_resolve"); err != nil {
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
