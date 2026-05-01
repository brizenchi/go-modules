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
