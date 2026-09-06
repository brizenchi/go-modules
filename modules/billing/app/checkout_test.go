package app

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/brizenchi/go-modules/modules/billing/domain"
	"github.com/brizenchi/go-modules/modules/billing/port"
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

func TestCheckout_AuthenticatedCustomerEmailOverridesRequestBody(t *testing.T) {
	prov := newMockProvider()
	store := newMockCustomerStore(port.Customer{UserID: "u1", Email: "owner@example.com"})
	svc := NewCheckoutService(prov, store)
	_, err := svc.Create(context.Background(), CheckoutInput{
		UserID:      "u1",
		Email:       "attacker@example.net",
		ProductType: domain.ProductCredits,
		Quantity:    1,
		SuccessURL:  "https://app.test/success",
		CancelURL:   "https://app.test/cancel",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got := prov.checkoutInputs[0].Email; got != "owner@example.com" {
		t.Fatalf("provider email = %q, want authenticated customer email", got)
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

func TestCheckout_RejectsNewPaidCheckoutWhileSubscriptionIsOngoing(t *testing.T) {
	for _, productType := range []domain.ProductType{domain.ProductSubscription, domain.ProductLifetime} {
		t.Run(string(productType), func(t *testing.T) {
			prov := newMockProvider()
			store := newMockCustomerStore(port.Customer{
				UserID:                 "u1",
				Email:                  "u@example.com",
				ProviderSubscriptionID: "sub_ongoing",
				SubscriptionStatus:     domain.StatusCanceling,
			})
			svc := NewCheckoutService(prov, store)
			_, err := svc.Create(context.Background(), CheckoutInput{
				UserID:      "u1",
				ProductType: productType,
				Plan:        domain.PlanStarter,
				Interval:    domain.IntervalMonthly,
				SuccessURL:  "https://app.test/success",
				CancelURL:   "https://app.test/cancel",
			})
			if !errors.Is(err, domain.ErrSubscriptionCheckoutConflict) {
				t.Fatalf("error = %v, want ErrSubscriptionCheckoutConflict", err)
			}
			if prov.checkoutCalls != 0 {
				t.Fatalf("provider checkout calls = %d, want 0", prov.checkoutCalls)
			}
		})
	}
}

func TestCheckout_RejectsNewPaidCheckoutForLifetimeOwner(t *testing.T) {
	for _, productType := range []domain.ProductType{domain.ProductSubscription, domain.ProductLifetime} {
		t.Run(string(productType), func(t *testing.T) {
			prov := newMockProvider()
			svc := NewCheckoutService(prov, newMockCustomerStore(port.Customer{
				UserID: "u1",
				Email:  "life@example.com",
				Plan:   string(domain.PlanLifetime),
			}))
			_, err := svc.Create(context.Background(), CheckoutInput{
				UserID:      "u1",
				ProductType: productType,
				Plan:        domain.PlanStarter,
				Interval:    domain.IntervalMonthly,
				SuccessURL:  "https://app.test/success",
				CancelURL:   "https://app.test/cancel",
			})
			if !errors.Is(err, domain.ErrSubscriptionCheckoutConflict) {
				t.Fatalf("error = %v, want ErrSubscriptionCheckoutConflict", err)
			}
		})
	}
}

func TestCheckout_AllowsNewPaidCheckoutAfterSubscriptionEnded(t *testing.T) {
	for _, status := range []domain.SubscriptionStatus{domain.StatusCanceled, domain.StatusIncompleteExpired} {
		t.Run(string(status), func(t *testing.T) {
			prov := newMockProvider()
			store := newMockCustomerStore(port.Customer{
				UserID:                 "u1",
				Email:                  "u@example.com",
				ProviderSubscriptionID: "sub_ended",
				SubscriptionStatus:     status,
			})
			svc := NewCheckoutService(prov, store)
			_, err := svc.Create(context.Background(), CheckoutInput{
				UserID:      "u1",
				ProductType: domain.ProductLifetime,
				SuccessURL:  "https://app.test/success",
				CancelURL:   "https://app.test/cancel",
			})
			if err != nil {
				t.Fatalf("Create: %v", err)
			}
			if prov.checkoutCalls != 1 {
				t.Fatalf("provider checkout calls = %d, want 1", prov.checkoutCalls)
			}
		})
	}
}

func TestCheckout_AllowsCreditsPurchaseWithExistingPaidEntitlement(t *testing.T) {
	for _, customer := range []port.Customer{
		{
			UserID:                 "u-recurring",
			Email:                  "recurring@example.com",
			ProviderSubscriptionID: "sub_active",
			SubscriptionStatus:     domain.StatusActive,
		},
		{
			UserID: "u-lifetime",
			Email:  "lifetime@example.com",
			Plan:   string(domain.PlanLifetime),
		},
	} {
		t.Run(customer.UserID, func(t *testing.T) {
			prov := newMockProvider()
			svc := NewCheckoutService(prov, newMockCustomerStore(customer))
			_, err := svc.Create(context.Background(), CheckoutInput{
				UserID:      customer.UserID,
				ProductType: domain.ProductCredits,
				Quantity:    1,
				SuccessURL:  "https://app.test/success",
				CancelURL:   "https://app.test/cancel",
			})
			if err != nil {
				t.Fatalf("Create credits checkout: %v", err)
			}
			if prov.checkoutCalls != 1 {
				t.Fatalf("provider checkout calls = %d, want 1", prov.checkoutCalls)
			}
		})
	}
}

func TestCheckout_ValidatesCreditsQuantity(t *testing.T) {
	for _, quantity := range []int64{-1, 0, 101} {
		t.Run(fmt.Sprintf("quantity_%d", quantity), func(t *testing.T) {
			prov := newMockProvider()
			svc := NewCheckoutService(prov, newMockCustomerStore(port.Customer{Email: "u@example.com"}))
			_, err := svc.Create(context.Background(), CheckoutInput{
				UserID:      "u1",
				ProductType: domain.ProductCredits,
				Quantity:    quantity,
				SuccessURL:  "https://app.test/success",
				CancelURL:   "https://app.test/cancel",
			})
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("error = %v, want ErrInvalidInput", err)
			}
			if prov.checkoutCalls != 0 {
				t.Fatalf("provider checkout calls = %d, want 0", prov.checkoutCalls)
			}
		})
	}
}

func TestCheckout_CreditsIgnoreClientSelectedPrice(t *testing.T) {
	prov := newMockProvider()
	svc := NewCheckoutService(prov, newMockCustomerStore(port.Customer{UserID: "u1", Email: "u@example.com"}))
	_, err := svc.Create(context.Background(), CheckoutInput{
		UserID:      "u1",
		ProductType: domain.ProductCredits,
		PriceID:     "price_client_selected",
		Quantity:    1,
		SuccessURL:  "https://app.test/success",
		CancelURL:   "https://app.test/cancel",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got := prov.checkoutInputs[0].PriceID; got != "" {
		t.Fatalf("provider price_id = %q, want server selection", got)
	}
}

func TestCheckout_ReservationBlocksDifferentPaidIntent(t *testing.T) {
	prov := newMockProvider()
	store := newMockCustomerStore(port.Customer{UserID: "u1", Email: "u@example.com"})
	svc := NewCheckoutService(prov, store)
	first, err := svc.Create(context.Background(), CheckoutInput{
		UserID:      "u1",
		ProductType: domain.ProductSubscription,
		Plan:        domain.PlanStarter,
		Interval:    domain.IntervalMonthly,
		SuccessURL:  "https://app.test/success",
		CancelURL:   "https://app.test/cancel",
	})
	if err != nil {
		t.Fatalf("first Create: %v", err)
	}
	_, err = svc.Create(context.Background(), CheckoutInput{
		UserID:      "u1",
		ProductType: domain.ProductLifetime,
		SuccessURL:  "https://different.test/success",
		CancelURL:   "https://different.test/cancel",
	})
	if !errors.Is(err, domain.ErrSubscriptionCheckoutConflict) {
		t.Fatalf("second error = %v, want checkout conflict", err)
	}
	if prov.checkoutCalls != 1 {
		t.Fatalf("provider checkout calls = %d, want 1", prov.checkoutCalls)
	}
	if first == nil || prov.checkoutInputs[0].CheckoutReservationID == "" || prov.checkoutInputs[0].CheckoutExpiresAt.IsZero() {
		t.Fatalf("provider did not receive reservation: %+v", prov.checkoutInputs)
	}
}

func TestCheckout_ReservationCompletedSessionConflictsEvenForSameIntent(t *testing.T) {
	prov := newMockProvider()
	prov.checkoutSession = &domain.CheckoutSessionSnapshot{SessionID: "cs_test", State: domain.CheckoutSessionComplete, PaymentStatus: "paid"}
	store := newMockCustomerStore(port.Customer{UserID: "u1", Email: "u@example.com"})
	svc := NewCheckoutService(prov, store)
	in := CheckoutInput{
		UserID:      "u1",
		ProductType: domain.ProductSubscription,
		Plan:        domain.PlanStarter,
		Interval:    domain.IntervalMonthly,
		SuccessURL:  "https://app.test/success",
		CancelURL:   "https://app.test/cancel",
	}
	_, err := svc.Create(context.Background(), in)
	if err != nil {
		t.Fatalf("first Create: %v", err)
	}
	_, err = svc.Create(context.Background(), in)
	if !errors.Is(err, domain.ErrSubscriptionCheckoutConflict) {
		t.Fatalf("second error = %v, want checkout conflict", err)
	}
	if prov.checkoutCalls != 1 {
		t.Fatalf("provider checkout calls = %d, want 1", prov.checkoutCalls)
	}
}

func TestCheckout_SameIntentReusesOpenProviderSession(t *testing.T) {
	prov := newMockProvider()
	store := newMockCustomerStore(port.Customer{UserID: "u1", Email: "u@example.com"})
	svc := NewCheckoutService(prov, store)
	in := CheckoutInput{
		UserID:      "u1",
		ProductType: domain.ProductSubscription,
		Plan:        domain.PlanStarter,
		Interval:    domain.IntervalMonthly,
		SuccessURL:  "https://app.test/success",
		CancelURL:   "https://app.test/cancel",
	}
	first, err := svc.Create(context.Background(), in)
	if err != nil {
		t.Fatalf("first Create: %v", err)
	}
	prov.checkoutSession = &domain.CheckoutSessionSnapshot{
		SessionID:   first.SessionID,
		CheckoutURL: "https://provider.test/recovered",
		State:       domain.CheckoutSessionOpen,
	}
	second, err := svc.Create(context.Background(), in)
	if err != nil {
		t.Fatalf("second Create: %v", err)
	}
	if second.SessionID != first.SessionID || second.CheckoutURL != "https://provider.test/recovered" {
		t.Fatalf("reused checkout = %+v, first = %+v", second, first)
	}
	if prov.checkoutCalls != 1 || prov.checkoutSessionCalls != 1 {
		t.Fatalf("checkout calls=%d status calls=%d, want 1/1", prov.checkoutCalls, prov.checkoutSessionCalls)
	}
}

func TestCheckout_ElapsedLocalExpiryDoesNotReplaceOpenProviderSession(t *testing.T) {
	prov := newMockProvider()
	prov.checkoutSession = &domain.CheckoutSessionSnapshot{SessionID: "cs_test", State: domain.CheckoutSessionOpen}
	store := newMockCustomerStore(port.Customer{UserID: "u1", Email: "u@example.com"})
	svc := NewCheckoutService(prov, store)
	in := CheckoutInput{UserID: "u1", ProductType: domain.ProductLifetime, SuccessURL: "https://app.test/success", CancelURL: "https://app.test/cancel"}
	if _, err := svc.Create(context.Background(), in); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	store.reservationMu.Lock()
	store.reservation.ExpiresAt = time.Now().Add(-time.Hour)
	store.reservationMu.Unlock()
	differentIntent := CheckoutInput{UserID: "u1", ProductType: domain.ProductSubscription, Plan: domain.PlanPro, Interval: domain.IntervalMonthly, SuccessURL: "https://app.test/success", CancelURL: "https://app.test/cancel"}
	if _, err := svc.Create(context.Background(), differentIntent); !errors.Is(err, domain.ErrSubscriptionCheckoutConflict) {
		t.Fatalf("second error = %v, want open-session conflict", err)
	}
	if prov.checkoutCalls != 1 || prov.checkoutSessionCalls != 1 {
		t.Fatalf("checkout calls=%d status calls=%d", prov.checkoutCalls, prov.checkoutSessionCalls)
	}
}

func TestCheckout_ProviderStatusFailureDoesNotReleaseReservation(t *testing.T) {
	prov := newMockProvider()
	store := newMockCustomerStore(port.Customer{UserID: "u1", Email: "u@example.com"})
	svc := NewCheckoutService(prov, store)
	in := CheckoutInput{UserID: "u1", ProductType: domain.ProductLifetime, SuccessURL: "https://app.test/success", CancelURL: "https://app.test/cancel"}
	if _, err := svc.Create(context.Background(), in); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	prov.checkoutSessionErr = errors.New("stripe unavailable")
	if _, err := svc.Create(context.Background(), in); err == nil || errors.Is(err, domain.ErrSubscriptionCheckoutConflict) {
		t.Fatalf("second error = %v, want provider verification failure", err)
	}
	if prov.checkoutCalls != 1 {
		t.Fatalf("provider checkout calls = %d, want 1", prov.checkoutCalls)
	}
}

func TestCheckout_ExpiredProviderSessionCanBeCASReplaced(t *testing.T) {
	prov := newMockProvider()
	store := newMockCustomerStore(port.Customer{UserID: "u1", Email: "u@example.com"})
	svc := NewCheckoutService(prov, store)
	firstInput := CheckoutInput{UserID: "u1", ProductType: domain.ProductLifetime, SuccessURL: "https://app.test/success", CancelURL: "https://app.test/cancel"}
	if _, err := svc.Create(context.Background(), firstInput); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	firstReservationID := prov.checkoutInputs[0].CheckoutReservationID
	prov.checkoutSession = &domain.CheckoutSessionSnapshot{SessionID: "cs_test", State: domain.CheckoutSessionExpired}
	secondInput := CheckoutInput{UserID: "u1", ProductType: domain.ProductSubscription, Plan: domain.PlanPro, Interval: domain.IntervalMonthly, SuccessURL: "https://app.test/success", CancelURL: "https://app.test/cancel"}
	if _, err := svc.Create(context.Background(), secondInput); err != nil {
		t.Fatalf("replacement Create: %v", err)
	}
	if prov.checkoutCalls != 2 || prov.checkoutInputs[1].CheckoutReservationID == firstReservationID {
		t.Fatalf("checkout inputs = %+v", prov.checkoutInputs)
	}
}

func TestCheckout_CompletedSessionCanBeReplacedOnlyByItsTerminalSubscription(t *testing.T) {
	prov := newMockProvider()
	prov.checkoutSession = &domain.CheckoutSessionSnapshot{SessionID: "cs_test", ProviderSubscriptionID: "sub_ended", State: domain.CheckoutSessionComplete, PaymentStatus: "paid"}
	store := newMockCustomerStore(port.Customer{UserID: "u1", Email: "u@example.com"})
	svc := NewCheckoutService(prov, store)
	if _, err := svc.Create(context.Background(), CheckoutInput{UserID: "u1", ProductType: domain.ProductSubscription, Plan: domain.PlanPro, Interval: domain.IntervalMonthly, SuccessURL: "https://app.test/success", CancelURL: "https://app.test/cancel"}); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	store.customer.ProviderSubscriptionID = "sub_ended"
	store.customer.SubscriptionStatus = domain.StatusCanceled
	if _, err := svc.Create(context.Background(), CheckoutInput{UserID: "u1", ProductType: domain.ProductLifetime, SuccessURL: "https://app.test/success", CancelURL: "https://app.test/cancel"}); err != nil {
		t.Fatalf("terminal replacement Create: %v", err)
	}
	if prov.checkoutCalls != 2 {
		t.Fatalf("provider checkout calls = %d, want 2", prov.checkoutCalls)
	}
}

func TestCheckout_ConcurrentDifferentIntentsCreateOneProviderSession(t *testing.T) {
	prov := newMockProvider()
	started := make(chan struct{})
	continueCheckout := make(chan struct{})
	prov.checkoutStarted = started
	prov.checkoutContinue = continueCheckout
	store := newMockCustomerStore(port.Customer{UserID: "u1", Email: "u@example.com"})
	svc := NewCheckoutService(prov, store)
	firstDone := make(chan error, 1)
	go func() {
		_, err := svc.Create(context.Background(), CheckoutInput{UserID: "u1", ProductType: domain.ProductLifetime, SuccessURL: "https://app.test/success", CancelURL: "https://app.test/cancel"})
		firstDone <- err
	}()
	<-started
	_, secondErr := svc.Create(context.Background(), CheckoutInput{UserID: "u1", ProductType: domain.ProductSubscription, Plan: domain.PlanPro, Interval: domain.IntervalMonthly, SuccessURL: "https://app.test/success", CancelURL: "https://app.test/cancel"})
	close(continueCheckout)
	if err := <-firstDone; err != nil {
		t.Fatalf("first Create: %v", err)
	}
	if !errors.Is(secondErr, domain.ErrSubscriptionCheckoutConflict) {
		t.Fatalf("second error = %v, want checkout conflict", secondErr)
	}
	if prov.checkoutCalls != 1 {
		t.Fatalf("provider checkout calls = %d, want 1", prov.checkoutCalls)
	}
}

func TestCheckout_SameIntentRetriesLostProviderResponseWithSameReservation(t *testing.T) {
	prov := newMockProvider()
	prov.checkoutErr = errors.New("response lost")
	store := newMockCustomerStore(port.Customer{UserID: "u1", Email: "u@example.com"})
	svc := NewCheckoutService(prov, store)
	in := CheckoutInput{UserID: "u1", ProductType: domain.ProductLifetime, SuccessURL: "https://app.test/success", CancelURL: "https://app.test/cancel"}
	if _, err := svc.Create(context.Background(), in); !errors.Is(err, prov.checkoutErr) {
		t.Fatalf("first error = %v", err)
	}
	firstReservationID := prov.checkoutInputs[0].CheckoutReservationID
	prov.checkoutErr = nil
	if _, err := svc.Create(context.Background(), in); err != nil {
		t.Fatalf("retry Create: %v", err)
	}
	if prov.checkoutCalls != 2 || prov.checkoutInputs[1].CheckoutReservationID != firstReservationID {
		t.Fatalf("checkout inputs = %+v, want same reservation id", prov.checkoutInputs)
	}
}

func TestCheckout_SameIntentRetriesLostCompletionWriteWithSameReservation(t *testing.T) {
	prov := newMockProvider()
	store := newMockCustomerStore(port.Customer{UserID: "u1", Email: "u@example.com"})
	store.completeReservationErrOnce = errors.New("db write failed")
	svc := NewCheckoutService(prov, store)
	in := CheckoutInput{UserID: "u1", ProductType: domain.ProductLifetime, SuccessURL: "https://app.test/success", CancelURL: "https://app.test/cancel"}
	if _, err := svc.Create(context.Background(), in); err == nil {
		t.Fatal("first Create should return completion write error")
	}
	firstReservationID := prov.checkoutInputs[0].CheckoutReservationID
	if _, err := svc.Create(context.Background(), in); err != nil {
		t.Fatalf("retry Create: %v", err)
	}
	if prov.checkoutCalls != 2 || prov.checkoutInputs[1].CheckoutReservationID != firstReservationID {
		t.Fatalf("checkout inputs = %+v, want same reservation id", prov.checkoutInputs)
	}
}

func TestCheckout_ExpiredSessionlessReservationRecoversOpenSession(t *testing.T) {
	prov := newMockProvider()
	prov.checkoutErr = errors.New("response lost")
	store := newMockCustomerStore(port.Customer{UserID: "u1", Email: "u@example.com"})
	svc := NewCheckoutService(prov, store)
	in := CheckoutInput{UserID: "u1", ProductType: domain.ProductLifetime, SuccessURL: "https://app.test/success", CancelURL: "https://app.test/cancel"}
	if _, err := svc.Create(context.Background(), in); !errors.Is(err, prov.checkoutErr) {
		t.Fatalf("first error = %v", err)
	}
	store.reservationMu.Lock()
	store.reservation.ExpiresAt = time.Now().Add(-time.Hour)
	reservationID := store.reservation.ReservationID
	store.reservationMu.Unlock()
	prov.checkoutErr = nil
	prov.findCheckoutSession = &domain.CheckoutSessionSnapshot{
		SessionID:   "cs_recovered",
		CheckoutURL: "https://provider.test/recovered",
		State:       domain.CheckoutSessionOpen,
	}
	result, err := svc.Create(context.Background(), in)
	if err != nil {
		t.Fatalf("recovery Create: %v", err)
	}
	if result.SessionID != "cs_recovered" || prov.checkoutCalls != 1 || prov.findCheckoutSessionCalls != 1 {
		t.Fatalf("result=%+v checkout calls=%d find calls=%d", result, prov.checkoutCalls, prov.findCheckoutSessionCalls)
	}
	if store.reservation == nil || store.reservation.ReservationID != reservationID || store.reservation.SessionID != "cs_recovered" {
		t.Fatalf("recovered reservation = %+v", store.reservation)
	}
}

func TestCheckout_ExpiredSessionlessSameIntentReplacesProviderExpiredSession(t *testing.T) {
	prov := newMockProvider()
	prov.checkoutErr = errors.New("response lost")
	store := newMockCustomerStore(port.Customer{UserID: "u1", Email: "u@example.com"})
	svc := NewCheckoutService(prov, store)
	in := CheckoutInput{UserID: "u1", ProductType: domain.ProductLifetime, SuccessURL: "https://app.test/success", CancelURL: "https://app.test/cancel"}
	if _, err := svc.Create(context.Background(), in); !errors.Is(err, prov.checkoutErr) {
		t.Fatalf("first error = %v", err)
	}
	oldReservationID := prov.checkoutInputs[0].CheckoutReservationID
	store.reservationMu.Lock()
	store.reservation.ExpiresAt = time.Now().Add(-time.Hour)
	store.reservationMu.Unlock()
	prov.checkoutErr = nil
	prov.findCheckoutSession = &domain.CheckoutSessionSnapshot{SessionID: "cs_expired", State: domain.CheckoutSessionExpired}
	if _, err := svc.Create(context.Background(), in); err != nil {
		t.Fatalf("replacement Create: %v", err)
	}
	if prov.checkoutCalls != 2 || prov.findCheckoutSessionCalls != 1 || prov.checkoutInputs[1].CheckoutReservationID == oldReservationID {
		t.Fatalf("checkout inputs=%+v find calls=%d", prov.checkoutInputs, prov.findCheckoutSessionCalls)
	}
	if !prov.checkoutInputs[1].CheckoutExpiresAt.After(time.Now()) {
		t.Fatalf("replacement used stale expiration: %v", prov.checkoutInputs[1].CheckoutExpiresAt)
	}
}

func TestCheckout_ExpiredSessionlessReservationCanCASReplaceAfterAuthoritativeMiss(t *testing.T) {
	prov := newMockProvider()
	prov.checkoutErr = errors.New("request never reached provider")
	store := newMockCustomerStore(port.Customer{UserID: "u1", Email: "u@example.com"})
	svc := NewCheckoutService(prov, store)
	first := CheckoutInput{UserID: "u1", ProductType: domain.ProductLifetime, SuccessURL: "https://app.test/success", CancelURL: "https://app.test/cancel"}
	if _, err := svc.Create(context.Background(), first); !errors.Is(err, prov.checkoutErr) {
		t.Fatalf("first error = %v", err)
	}
	oldReservationID := prov.checkoutInputs[0].CheckoutReservationID
	store.reservationMu.Lock()
	store.reservation.ExpiresAt = time.Now().Add(-time.Hour)
	store.reservationMu.Unlock()
	prov.checkoutErr = nil
	second := CheckoutInput{UserID: "u1", ProductType: domain.ProductSubscription, Plan: domain.PlanPro, Interval: domain.IntervalYearly, SuccessURL: "https://app.test/success", CancelURL: "https://app.test/cancel"}
	if _, err := svc.Create(context.Background(), second); err != nil {
		t.Fatalf("replacement Create: %v", err)
	}
	if prov.findCheckoutSessionCalls != 1 || prov.checkoutCalls != 2 || prov.checkoutInputs[1].CheckoutReservationID == oldReservationID {
		t.Fatalf("checkout inputs=%+v find calls=%d", prov.checkoutInputs, prov.findCheckoutSessionCalls)
	}
}

func TestCheckout_ExpiredSessionlessRecoveryErrorFailsClosed(t *testing.T) {
	prov := newMockProvider()
	prov.checkoutErr = errors.New("response lost")
	store := newMockCustomerStore(port.Customer{UserID: "u1", Email: "u@example.com"})
	svc := NewCheckoutService(prov, store)
	in := CheckoutInput{UserID: "u1", ProductType: domain.ProductLifetime, SuccessURL: "https://app.test/success", CancelURL: "https://app.test/cancel"}
	if _, err := svc.Create(context.Background(), in); !errors.Is(err, prov.checkoutErr) {
		t.Fatalf("first error = %v", err)
	}
	store.reservationMu.Lock()
	store.reservation.ExpiresAt = time.Now().Add(-time.Hour)
	store.reservationMu.Unlock()
	prov.checkoutErr = nil
	prov.findCheckoutSessionErr = errors.New("stripe unavailable")
	if _, err := svc.Create(context.Background(), in); err == nil || errors.Is(err, domain.ErrSubscriptionCheckoutConflict) {
		t.Fatalf("recovery error = %v, want provider error", err)
	}
	if prov.checkoutCalls != 1 || store.reservation == nil {
		t.Fatalf("checkout calls=%d reservation=%+v", prov.checkoutCalls, store.reservation)
	}
}

func TestCheckout_ExpiredSessionlessCompletedPaymentStillConflicts(t *testing.T) {
	prov := newMockProvider()
	prov.checkoutErr = errors.New("completion write lost")
	store := newMockCustomerStore(port.Customer{UserID: "u1", Email: "u@example.com"})
	svc := NewCheckoutService(prov, store)
	in := CheckoutInput{UserID: "u1", ProductType: domain.ProductLifetime, SuccessURL: "https://app.test/success", CancelURL: "https://app.test/cancel"}
	if _, err := svc.Create(context.Background(), in); !errors.Is(err, prov.checkoutErr) {
		t.Fatalf("first error = %v", err)
	}
	store.reservationMu.Lock()
	store.reservation.ExpiresAt = time.Now().Add(-time.Hour)
	store.reservationMu.Unlock()
	prov.checkoutErr = nil
	prov.findCheckoutSession = &domain.CheckoutSessionSnapshot{SessionID: "cs_paid", State: domain.CheckoutSessionComplete, PaymentStatus: "paid"}
	if _, err := svc.Create(context.Background(), in); !errors.Is(err, domain.ErrSubscriptionCheckoutConflict) {
		t.Fatalf("recovery error = %v, want checkout conflict", err)
	}
	if prov.checkoutCalls != 1 {
		t.Fatalf("provider checkout calls=%d, want 1", prov.checkoutCalls)
	}
}
