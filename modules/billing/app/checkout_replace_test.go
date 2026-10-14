package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/brizenchi/go-modules/modules/billing/domain"
	"github.com/brizenchi/go-modules/modules/billing/port"
)

type expiringCheckoutProvider struct {
	*mockProvider
	expireCalls  int
	expireErr    error
	expired      *domain.CheckoutSessionSnapshot
	beforeExpire func()
}

func (p *expiringCheckoutProvider) ExpireCheckoutSession(ctx context.Context, id string) (*domain.CheckoutSessionSnapshot, error) {
	p.expireCalls++
	if p.beforeExpire != nil {
		p.beforeExpire()
	}
	if p.expireErr != nil {
		return nil, p.expireErr
	}
	if p.expired != nil {
		return p.expired, nil
	}
	return &domain.CheckoutSessionSnapshot{SessionID: id, State: domain.CheckoutSessionExpired, PaymentStatus: "unpaid"}, nil
}

func TestCheckout_ChangedIntentExpiresUnpaidSessionBeforeReplacement(t *testing.T) {
	for _, recovered := range []bool{false, true} {
		t.Run(map[bool]string{false: "known session", true: "recovered lost response"}[recovered], func(t *testing.T) {
			p := &expiringCheckoutProvider{mockProvider: newMockProvider()}
			store := newMockCustomerStore(port.Customer{UserID: "u1", Email: "user@example.test"})
			svc := NewCheckoutService(p, store)
			first := CheckoutInput{UserID: "u1", ProductType: domain.ProductSubscription, Plan: domain.PlanStarter, Interval: domain.IntervalMonthly, SuccessURL: "https://app.test/success", CancelURL: "https://app.test/cancel"}
			if _, err := svc.Create(context.Background(), first); err != nil {
				t.Fatal(err)
			}
			oldID := store.reservation.ReservationID
			snapshot := &domain.CheckoutSessionSnapshot{SessionID: "cs_test", State: domain.CheckoutSessionOpen, PaymentStatus: "unpaid"}
			if recovered {
				store.reservation.SessionID = ""
				store.reservation.ExpiresAt = time.Now().Add(-time.Hour)
				p.findCheckoutSession = snapshot
			} else {
				p.checkoutSession = snapshot
			}
			p.beforeExpire = func() {
				if p.checkoutCalls != 1 || store.reservation.ReservationID != oldID {
					t.Fatal("reservation replaced before expiration")
				}
			}
			second := first
			second.Plan = domain.PlanPro
			if _, err := svc.Create(context.Background(), second); err != nil {
				t.Fatalf("replace unpaid checkout: %v", err)
			}
			if p.expireCalls != 1 || p.checkoutCalls != 2 || store.reservation.ReservationID == oldID {
				t.Fatalf("expire=%d create=%d reservation=%+v", p.expireCalls, p.checkoutCalls, store.reservation)
			}
		})
	}
}

func TestCheckout_ExpirationFailureOrConcurrentCompletionDoesNotReplace(t *testing.T) {
	for _, tt := range []struct {
		name     string
		err      error
		snapshot *domain.CheckoutSessionSnapshot
	}{
		{name: "provider error", err: errors.New("expiration unavailable")},
		{name: "payment won race", snapshot: &domain.CheckoutSessionSnapshot{SessionID: "cs_test", State: domain.CheckoutSessionComplete, PaymentStatus: "paid"}},
		{name: "still open", snapshot: &domain.CheckoutSessionSnapshot{SessionID: "cs_test", State: domain.CheckoutSessionOpen, PaymentStatus: "unpaid"}},
		{name: "wrong session", snapshot: &domain.CheckoutSessionSnapshot{SessionID: "cs_other", State: domain.CheckoutSessionExpired, PaymentStatus: "unpaid"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p := &expiringCheckoutProvider{mockProvider: newMockProvider(), expireErr: tt.err, expired: tt.snapshot}
			store := newMockCustomerStore(port.Customer{UserID: "u1", Email: "user@example.test"})
			svc := NewCheckoutService(p, store)
			in := CheckoutInput{UserID: "u1", ProductType: domain.ProductLifetime, SuccessURL: "https://app.test/success", CancelURL: "https://app.test/cancel"}
			if _, err := svc.Create(context.Background(), in); err != nil {
				t.Fatal(err)
			}
			oldID := store.reservation.ReservationID
			p.checkoutSession = &domain.CheckoutSessionSnapshot{SessionID: "cs_test", State: domain.CheckoutSessionOpen, PaymentStatus: "unpaid"}
			in.ProductType = domain.ProductSubscription
			in.Plan = domain.PlanPro
			in.Interval = domain.IntervalMonthly
			if _, err := svc.Create(context.Background(), in); err == nil {
				t.Fatal("unsafe replacement succeeded")
			}
			if p.expireCalls != 1 || p.checkoutCalls != 1 || store.reservation.ReservationID != oldID {
				t.Fatalf("expire=%d create=%d reservation=%+v", p.expireCalls, p.checkoutCalls, store.reservation)
			}
		})
	}
}

type checkoutReplacementLostStore struct{ *mockCustomerStore }

func (s *checkoutReplacementLostStore) ReplaceCheckoutReservation(ctx context.Context, userID, provider, providerCustomerID, expectedReservationID, reservationID, intentKey string, expiresAt time.Time) (bool, error) {
	return false, nil
}

func TestCheckout_ExpiredSessionReplacementLostRaceCannotCreateNewCheckout(t *testing.T) {
	p := &expiringCheckoutProvider{mockProvider: newMockProvider()}
	store := &checkoutReplacementLostStore{newMockCustomerStore(port.Customer{UserID: "u1", Email: "user@example.test"})}
	svc := NewCheckoutService(p, store)
	in := CheckoutInput{UserID: "u1", ProductType: domain.ProductLifetime, SuccessURL: "https://app.test/success", CancelURL: "https://app.test/cancel"}
	if _, err := svc.Create(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	p.checkoutSession = &domain.CheckoutSessionSnapshot{SessionID: "cs_test", State: domain.CheckoutSessionOpen, PaymentStatus: "unpaid"}
	in.ProductType = domain.ProductSubscription
	in.Plan = domain.PlanPro
	in.Interval = domain.IntervalMonthly
	if _, err := svc.Create(context.Background(), in); !errors.Is(err, domain.ErrSubscriptionCheckoutConflict) {
		t.Fatalf("error=%v, want concurrent replacement conflict", err)
	}
	if p.expireCalls != 1 || p.checkoutCalls != 1 {
		t.Fatalf("expire=%d create=%d", p.expireCalls, p.checkoutCalls)
	}
}

func TestCheckout_SameIntentWithExpirationCapabilityReusesOriginal(t *testing.T) {
	p := &expiringCheckoutProvider{mockProvider: newMockProvider()}
	store := newMockCustomerStore(port.Customer{UserID: "u1", Email: "user@example.test"})
	svc := NewCheckoutService(p, store)
	in := CheckoutInput{UserID: "u1", ProductType: domain.ProductLifetime, SuccessURL: "https://app.test/success", CancelURL: "https://app.test/cancel"}
	first, err := svc.Create(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	p.checkoutSession = &domain.CheckoutSessionSnapshot{SessionID: first.SessionID, CheckoutURL: first.CheckoutURL, State: domain.CheckoutSessionOpen, PaymentStatus: "unpaid"}
	second, err := svc.Create(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if second.SessionID != first.SessionID || p.expireCalls != 0 || p.checkoutCalls != 1 {
		t.Fatalf("result=%+v expire=%d create=%d", second, p.expireCalls, p.checkoutCalls)
	}
}
