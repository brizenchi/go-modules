package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/brizenchi/go-modules/modules/billing/domain"
	"github.com/brizenchi/go-modules/modules/billing/event"
	"github.com/brizenchi/go-modules/modules/billing/port"
)

func recurringLifetimeCustomer() port.Customer {
	return port.Customer{UserID: "u1", Email: "user@example.test", Plan: "pro", ProviderCustomerID: "cus_existing", ProviderSubscriptionID: "sub_existing", SubscriptionStatus: domain.StatusActive}
}

func TestCheckout_LifetimeReplacesASettledRecurringReservation(t *testing.T) {
	for _, payment := range []string{"paid", "no_payment_required", "unpaid"} {
		t.Run(payment, func(t *testing.T) {
			provider := &lifetimeProvider{mockProvider: newMockProvider()}
			store := newMockCustomerStore(recurringLifetimeCustomer())
			store.reservation = &domain.BillingCheckoutReservation{UserID: "u1", Provider: "mock", ReservationID: "old", IntentKey: "old-subscription", SessionID: "cs_old", ProviderSubscriptionID: "sub_existing", ExpiresAt: time.Now().Add(time.Hour)}
			provider.checkoutSession = &domain.CheckoutSessionSnapshot{SessionID: "cs_old", State: domain.CheckoutSessionComplete, PaymentStatus: payment, ProviderSubscriptionID: "sub_existing"}
			_, err := NewCheckoutService(provider, store).Create(t.Context(), CheckoutInput{UserID: "u1", ProductType: domain.ProductLifetime, SuccessURL: "https://app.test/success", CancelURL: "https://app.test/cancel"})
			if payment == "unpaid" {
				if !errors.Is(err, domain.ErrSubscriptionCheckoutConflict) {
					t.Fatalf("want unpaid conflict, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if provider.checkoutCalls != 1 || provider.checkoutInputs[0].ProductType != domain.ProductLifetime || provider.cancelCalls != 0 {
				t.Fatalf("checkout changed subscription before payment: %+v", provider)
			}
		})
	}
}

type lifetimeProvider struct {
	*mockProvider
	retired   int
	retireErr error
}

func (p *lifetimeProvider) RetireSubscriptionForLifetime(_ context.Context, id, customerID string) error {
	if id != "sub_existing" || customerID != "cus_existing" {
		return domain.ErrInvalidInput
	}
	p.retired++
	return p.retireErr
}

func lifetimeWebhookFixture() (*lifetimeProvider, *mockCustomerStore, *mockRepo, *mockSubscriptionRepo, *mockBus) {
	provider := &lifetimeProvider{mockProvider: newMockProvider()}
	provider.parseResult = &port.WebhookParseResult{ProviderEventID: "evt_lifetime_paid", Type: "checkout.session.completed", UserHint: port.UserHint{UserID: "u1"}, Envelopes: []event.Envelope{{Kind: event.KindSubscriptionActivated, OccurredAt: time.Now().UTC(), Payload: event.SubscriptionActivated{Snapshot: domain.SubscriptionSnapshot{ProductType: domain.ProductLifetime, Plan: domain.PlanLifetime, Status: domain.StatusActive, ProviderCustomerID: "cus_existing"}}}}}
	return provider, newMockCustomerStore(recurringLifetimeCustomer()), newMockRepo(), &mockSubscriptionRepo{}, newMockBus()
}

func TestPaidLifetimeRetiresRecurringBeforeGrantAndDeduplicates(t *testing.T) {
	p, c, r, s, b := lifetimeWebhookFixture()
	service := NewWebhookService(p, r, s, nil, b).WithCustomers(c)
	if _, err := service.Process(t.Context(), nil, "fixture-signature"); err != nil {
		t.Fatal(err)
	}
	if p.retired != 1 || len(s.writes) != 1 || s.writes[0].snapshot.Plan != domain.PlanLifetime || !r.rows["evt_lifetime_paid"].Processed {
		t.Fatal("lifetime was not applied after retiring recurring")
	}
	if _, err := service.Process(t.Context(), nil, "fixture-signature"); err != nil {
		t.Fatal(err)
	}
	if p.retired != 1 {
		t.Fatal("duplicate payment retired subscription again")
	}
}

func TestPaidLifetimeCleanupFailureRetriesWithoutLosingPriorSubscription(t *testing.T) {
	p, c, r, s, b := lifetimeWebhookFixture()
	p.retireErr = errors.New("fixture provider offline")
	service := NewWebhookService(p, r, s, nil, b).WithCustomers(c)
	if _, err := service.Process(t.Context(), nil, "fixture-signature"); err == nil {
		t.Fatal("want retirement failure")
	}
	if len(s.writes) != 0 || r.rows["evt_lifetime_paid"].Processed {
		t.Fatal("failed retirement lost the recurring reference or marked processed")
	}
	p.retireErr = nil
	if _, err := service.Process(t.Context(), nil, "fixture-signature"); err != nil {
		t.Fatal(err)
	}
	if len(s.writes) != 1 || !r.rows["evt_lifetime_paid"].Processed {
		t.Fatal("retry did not activate lifetime")
	}
}

func TestUnpaidLifetimeCheckoutDoesNotRetireSubscription(t *testing.T) {
	p, c, r, s, b := lifetimeWebhookFixture()
	p.parseResult.Envelopes = nil // Unpaid Stripe checkout translation emits no entitlement event.
	if _, err := NewWebhookService(p, r, s, nil, b).WithCustomers(c).Process(t.Context(), nil, "fixture-signature"); err != nil {
		t.Fatal(err)
	}
	if p.retired != 0 || len(s.writes) != 0 {
		t.Fatal("unpaid checkout changed access")
	}
}

func TestLifetimeCustomerMismatchCannotRetireAnotherSubscription(t *testing.T) {
	p, c, r, s, b := lifetimeWebhookFixture()
	c.customer.ProviderCustomerID = "cus_other"
	if _, err := NewWebhookService(p, r, s, nil, b).WithCustomers(c).Process(t.Context(), nil, "fixture-signature"); err == nil {
		t.Fatal("want customer ownership failure")
	}
	if p.retired != 0 || len(s.writes) != 0 {
		t.Fatal("customer mismatch changed subscription")
	}
}
