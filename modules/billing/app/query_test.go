package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/brizenchi/go-modules/modules/billing/domain"
	"github.com/brizenchi/go-modules/modules/billing/port"
)

func TestGetSubscription_ReturnsLifetimeWithoutProviderSubscription(t *testing.T) {
	prov := newMockProvider()
	store := newMockCustomerStore(port.Customer{
		UserID:             "u1",
		Email:              "life@example.com",
		Plan:               "lifetime",
		ProviderCustomerID: "cus_life",
	})

	view, err := NewQueryService(prov, store).GetSubscription(context.Background(), "u1")
	if err != nil {
		t.Fatalf("GetSubscription: %v", err)
	}
	if view.Plan != domain.PlanLifetime {
		t.Fatalf("plan = %s, want lifetime", view.Plan)
	}
	if view.Status != domain.StatusActive {
		t.Fatalf("status = %s, want active", view.Status)
	}
	if view.Interval != "" {
		t.Fatalf("billing cycle = %s, want empty", view.Interval)
	}
}

func TestGetSubscription_ExposesExplicitCancelAtAsCanceling(t *testing.T) {
	deadline := time.Now().UTC().Add(72 * time.Hour)
	prov := newMockProvider()
	prov.subSnapshot = &domain.SubscriptionSnapshot{
		ProviderSubscriptionID: "sub_cancel_at",
		Status:                 domain.StatusCanceling,
		CancelAtPeriodEnd:      false,
		CancelEffectiveAt:      &deadline,
	}
	store := newMockCustomerStore(port.Customer{
		UserID:                 "u1",
		ProviderCustomerID:     "cus_1",
		ProviderSubscriptionID: "sub_cancel_at",
		SubscriptionStatus:     domain.StatusCanceling,
	})
	view, err := NewQueryService(prov, store).GetSubscription(context.Background(), "u1")
	if err != nil {
		t.Fatalf("GetSubscription: %v", err)
	}
	if view.Status != domain.StatusCanceling || view.CancelAtPeriodEnd || view.CancelEffectiveAt != deadline.UTC().Format("2006-01-02T15:04:05Z07:00") {
		t.Fatalf("view = %+v, want explicit pending cancellation", view)
	}
}

func TestGetSubscription_FailsClosedWhenProviderRefreshFails(t *testing.T) {
	providerErr := errors.New("provider unavailable")
	prov := newMockProvider()
	prov.subErr = providerErr
	store := newMockCustomerStore(port.Customer{
		UserID:                 "u1",
		Email:                  "paid@example.com",
		ProviderCustomerID:     "cus_paid",
		ProviderSubscriptionID: "sub_paid",
		SubscriptionStatus:     domain.StatusActive,
	})

	view, err := NewQueryService(prov, store).GetSubscription(context.Background(), "u1")
	if view != nil {
		t.Fatalf("view = %+v, want nil", view)
	}
	if !errors.Is(err, providerErr) {
		t.Fatalf("error = %v, want wrapped provider error", err)
	}
}

func TestGetSubscription_UsesCanonicalTerminalSnapshotWithoutProviderLookup(t *testing.T) {
	providerErr := errors.New("subscription deleted at provider")
	prov := newMockProvider()
	prov.subErr = providerErr
	store := newMockCustomerStore(port.Customer{
		UserID:                 "u1",
		Email:                  "ended@example.com",
		Plan:                   string(domain.PlanPro),
		ProviderCustomerID:     "cus_ended",
		ProviderSubscriptionID: "sub_deleted",
		SubscriptionStatus:     domain.StatusCanceled,
	})

	view, err := NewQueryService(prov, store).GetSubscription(context.Background(), "u1")
	if err != nil {
		t.Fatalf("GetSubscription: %v", err)
	}
	if view.Plan != domain.PlanPro || view.Status != domain.StatusCanceled {
		t.Fatalf("view=%+v, want canonical canceled pro subscription", view)
	}
}
