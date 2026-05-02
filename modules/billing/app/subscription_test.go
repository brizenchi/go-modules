package app

import (
	"context"
	"errors"
	"testing"

	"github.com/brizenchi/go-modules/modules/billing/domain"
	"github.com/brizenchi/go-modules/modules/billing/event"
	"github.com/brizenchi/go-modules/modules/billing/port"
)

func TestCancel_PublishesCancelingEvent(t *testing.T) {
	prov := newMockProvider()
	store := newMockCustomerStore(port.Customer{
		UserID:                 "u1",
		ProviderCustomerID:     "cus_x",
		ProviderSubscriptionID: "sub_x",
	})
	bus := newMockBus()
	svc := NewSubscriptionService(prov, store, bus)

	res, err := svc.Cancel(context.Background(), "u1", domain.CancelAtPeriodEnd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prov.cancelCalls != 1 {
		t.Errorf("provider cancel calls = %d, want 1", prov.cancelCalls)
	}
	if res.Mode != domain.CancelAtPeriodEnd {
		t.Errorf("mode = %s, want end_of_period", res.Mode)
	}
	pub := bus.Published()
	if len(pub) != 1 || pub[0].Kind != event.KindSubscriptionCanceling {
		t.Errorf("expected one Canceling event, got %v", pub)
	}
}

func TestCancel_RejectsInvalidMode(t *testing.T) {
	svc := NewSubscriptionService(newMockProvider(), newMockCustomerStore(port.Customer{}), newMockBus())
	_, err := svc.Cancel(context.Background(), "u1", domain.CancelMode("immediate"))
	if !errors.Is(err, domain.ErrInvalidCancelMode) {
		t.Errorf("expected ErrInvalidCancelMode, got %v", err)
	}
}

func TestCancel_NoActiveSubscription(t *testing.T) {
	svc := NewSubscriptionService(newMockProvider(), newMockCustomerStore(port.Customer{UserID: "u1"}), newMockBus())
	_, err := svc.Cancel(context.Background(), "u1", domain.CancelAtPeriodEnd)
	if !errors.Is(err, domain.ErrNoActiveSubscription) {
		t.Errorf("expected ErrNoActiveSubscription, got %v", err)
	}
}

func TestReactivate_PublishesReactivatedEvent(t *testing.T) {
	prov := newMockProvider()
	store := newMockCustomerStore(port.Customer{
		UserID:                 "u1",
		ProviderSubscriptionID: "sub_x",
	})
	bus := newMockBus()
	svc := NewSubscriptionService(prov, store, bus)

	id, err := svc.Reactivate(context.Background(), "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "sub_x" {
		t.Errorf("returned subscription id = %q, want sub_x", id)
	}
	if prov.reactivateCalls != 1 {
		t.Errorf("provider reactivate calls = %d, want 1", prov.reactivateCalls)
	}
	pub := bus.Published()
	if len(pub) != 1 || pub[0].Kind != event.KindSubscriptionReactivated {
		t.Errorf("expected one Reactivated event, got %v", pub)
	}
}

func TestReactivate_NoSubscription(t *testing.T) {
	svc := NewSubscriptionService(newMockProvider(), newMockCustomerStore(port.Customer{UserID: "u1"}), newMockBus())
	_, err := svc.Reactivate(context.Background(), "u1")
	if !errors.Is(err, domain.ErrNoSubscriptionToReactive) {
		t.Errorf("expected ErrNoSubscriptionToReactive, got %v", err)
	}
}

func TestChange_PublishesUpdatedEvent(t *testing.T) {
	prov := newMockProvider()
	store := newMockCustomerStore(port.Customer{
		UserID:                 "u1",
		ProviderSubscriptionID: "sub_x",
	})
	bus := newMockBus()
	svc := NewSubscriptionService(prov, store, bus)

	res, err := svc.Change(context.Background(), "u1", domain.SubscriptionChangeInput{
		Plan:     domain.PlanPro,
		Interval: domain.IntervalMonthly,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prov.changeCalls != 1 {
		t.Errorf("provider change calls = %d, want 1", prov.changeCalls)
	}
	if res.Snapshot.Plan != domain.PlanPro {
		t.Errorf("plan = %s, want pro", res.Snapshot.Plan)
	}
	pub := bus.Published()
	if len(pub) != 1 || pub[0].Kind != event.KindSubscriptionUpdated {
		t.Errorf("expected one Updated event, got %v", pub)
	}
}

func TestChange_NoActiveSubscription(t *testing.T) {
	svc := NewSubscriptionService(newMockProvider(), newMockCustomerStore(port.Customer{UserID: "u1"}), newMockBus())
	_, err := svc.Change(context.Background(), "u1", domain.SubscriptionChangeInput{
		Plan:     domain.PlanPro,
		Interval: domain.IntervalMonthly,
	})
	if !errors.Is(err, domain.ErrNoActiveSubscription) {
		t.Errorf("expected ErrNoActiveSubscription, got %v", err)
	}
}

func TestOpenBillingPortal_ReturnsURL(t *testing.T) {
	prov := newMockProvider()
	store := newMockCustomerStore(port.Customer{
		UserID:             "u1",
		ProviderCustomerID: "cus_x",
	})
	svc := NewSubscriptionService(prov, store, newMockBus())

	res, err := svc.OpenBillingPortal(context.Background(), "u1", "https://app.test/billing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.URL == "" {
		t.Fatal("expected portal URL")
	}
}

func TestOpenBillingPortal_NoCustomer(t *testing.T) {
	svc := NewSubscriptionService(newMockProvider(), newMockCustomerStore(port.Customer{UserID: "u1"}), newMockBus())
	_, err := svc.OpenBillingPortal(context.Background(), "u1", "https://app.test/billing")
	if !errors.Is(err, domain.ErrNoBillingCustomer) {
		t.Errorf("expected ErrNoBillingCustomer, got %v", err)
	}
}

func TestChange_ResolvesMonthlyToYearlyAsImmediateResetCycle(t *testing.T) {
	prov := newMockProvider()
	prov.subSnapshot = &domain.SubscriptionSnapshot{
		ProviderSubscriptionID: "sub_x",
		Plan:                   domain.PlanPro,
		Interval:               domain.IntervalMonthly,
		Status:                 domain.StatusActive,
	}
	store := newMockCustomerStore(port.Customer{
		UserID:                 "u1",
		ProviderSubscriptionID: "sub_x",
	})
	res, err := NewSubscriptionService(prov, store, newMockBus()).Change(context.Background(), "u1", domain.SubscriptionChangeInput{
		Plan:     domain.PlanPro,
		Interval: domain.IntervalYearly,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Mode != domain.ChangeModeImmediateResetCycle {
		t.Errorf("mode = %s", res.Mode)
	}
}

func TestChange_ResolvesYearlyToMonthlyAsPeriodEnd(t *testing.T) {
	prov := newMockProvider()
	prov.subSnapshot = &domain.SubscriptionSnapshot{
		ProviderSubscriptionID: "sub_x",
		Plan:                   domain.PlanPro,
		Interval:               domain.IntervalYearly,
		Status:                 domain.StatusActive,
	}
	store := newMockCustomerStore(port.Customer{
		UserID:                 "u1",
		ProviderSubscriptionID: "sub_x",
	})
	res, err := NewSubscriptionService(prov, store, newMockBus()).Change(context.Background(), "u1", domain.SubscriptionChangeInput{
		Plan:     domain.PlanPro,
		Interval: domain.IntervalMonthly,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Mode != domain.ChangeModePeriodEnd {
		t.Errorf("mode = %s", res.Mode)
	}
	if prov.scheduleCalls != 1 {
		t.Errorf("schedule calls = %d, want 1", prov.scheduleCalls)
	}
}

func TestPreviewChange_UsesProviderPreview(t *testing.T) {
	prov := newMockProvider()
	store := newMockCustomerStore(port.Customer{
		UserID:                 "u1",
		ProviderCustomerID:     "cus_x",
		ProviderSubscriptionID: "sub_x",
	})
	res, err := NewSubscriptionService(prov, store, newMockBus()).PreviewChange(context.Background(), "u1", domain.SubscriptionPreviewInput{
		Plan:     domain.PlanPremium,
		Interval: domain.IntervalMonthly,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.TargetPlan != domain.PlanPremium {
		t.Errorf("target plan = %s", res.TargetPlan)
	}
}
