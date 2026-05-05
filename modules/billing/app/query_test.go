package app

import (
	"context"
	"testing"

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
