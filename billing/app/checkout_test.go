package app

import (
	"context"
	"errors"
	"testing"

	"github.com/brizenchi/go-modules/billing/domain"
	"github.com/brizenchi/go-modules/billing/port"
)

func TestCheckout_RequiresUserID(t *testing.T) {
	prov := newMockProvider()
	store := newMockCustomerStore(port.Customer{})
	svc := NewCheckoutService(prov, store)
	_, err := svc.Create(context.Background(), CheckoutInput{
		ProductType: domain.ProductSubscription,
		SuccessURL:  "https://x", CancelURL: "https://x",
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for missing user_id, got %v", err)
	}
}

func TestCheckout_PullsEmailFromCustomerStore(t *testing.T) {
	prov := newMockProvider()
	store := newMockCustomerStore(port.Customer{
		UserID: "u1",
		Email:  "found@example.com",
	})
	svc := NewCheckoutService(prov, store)
	_, err := svc.Create(context.Background(), CheckoutInput{
		UserID:      "u1",
		ProductType: domain.ProductSubscription,
		Plan:        domain.PlanStarter,
		Interval:    domain.IntervalMonthly,
		SuccessURL:  "https://x", CancelURL: "https://y",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckout_RejectsMissingURLs(t *testing.T) {
	svc := NewCheckoutService(newMockProvider(), newMockCustomerStore(port.Customer{Email: "a@b"}))
	_, err := svc.Create(context.Background(), CheckoutInput{
		UserID:      "u1",
		ProductType: domain.ProductSubscription,
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCheckout_PersistsNewCustomerID(t *testing.T) {
	prov := newMockProvider()
	prov.ensureCustomerID = "cus_NEW"
	store := newMockCustomerStore(port.Customer{
		UserID:             "u1",
		Email:              "u@x",
		ProviderCustomerID: "cus_OLD",
	})
	svc := NewCheckoutService(prov, store)
	_, err := svc.Create(context.Background(), CheckoutInput{
		UserID:      "u1",
		ProductType: domain.ProductSubscription,
		Plan:        domain.PlanStarter,
		Interval:    domain.IntervalMonthly,
		SuccessURL:  "https://x", CancelURL: "https://y",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := store.saved["u1"]; got != "cus_NEW" {
		t.Errorf("saved customer id = %q, want cus_NEW", got)
	}
}

func TestCheckout_ReusesExistingCustomerID(t *testing.T) {
	prov := newMockProvider()
	store := newMockCustomerStore(port.Customer{
		UserID:             "u1",
		Email:              "u@x",
		ProviderCustomerID: "cus_KEEP",
	})
	svc := NewCheckoutService(prov, store)
	_, _ = svc.Create(context.Background(), CheckoutInput{
		UserID:      "u1",
		ProductType: domain.ProductSubscription,
		Plan:        domain.PlanStarter,
		Interval:    domain.IntervalMonthly,
		SuccessURL:  "https://x", CancelURL: "https://y",
	})
	if _, ok := store.saved["u1"]; ok {
		t.Error("expected no save when customer id unchanged")
	}
}
