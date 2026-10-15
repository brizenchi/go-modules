package app

import (
	"context"
	"errors"
	"testing"

	"github.com/brizenchi/go-modules/modules/billing/domain"
	"github.com/brizenchi/go-modules/modules/billing/port"
)

func TestCheckoutObserverReceivesSuccessfulNormalizedIntentOnly(t *testing.T) {
	provider := newMockProvider()
	store := newMockCustomerStore(port.Customer{UserID: "observer-user", Email: "owner@example.test"})
	amount := int64(990)
	mode := false
	provider.checkoutResult = &domain.CheckoutResult{SessionID: "cs_observer", CheckoutURL: "https://checkout.example.test/session", AmountTotal: &amount, Currency: "usd", Livemode: &mode}
	type contextKey struct{}
	ctx := context.WithValue(t.Context(), contextKey{}, "correlated")
	calls := 0
	service := NewCheckoutService(provider, store).WithObserver(func(observed context.Context, in CheckoutInput, result CheckoutResult) {
		calls++
		if observed.Value(contextKey{}) != "correlated" || in.UserID != "observer-user" || in.Email != "owner@example.test" || in.Quantity != 2 || result.SessionID != "cs_observer" || result.AmountTotal == nil || *result.AmountTotal != 990 {
			t.Fatalf("unexpected observation: %+v %+v", in, result)
		}
	})
	input := CheckoutInput{UserID: " observer-user ", Email: "ignored@example.test", ProductType: domain.ProductCredits, Quantity: 2, SuccessURL: "https://app.example.test/success", CancelURL: "https://app.example.test/cancel"}
	result, err := service.Create(ctx, input)
	if err != nil || result.SessionID != "cs_observer" || calls != 1 {
		t.Fatalf("checkout result: %+v %v calls=%d", result, err, calls)
	}
	provider.checkoutErr = errors.New("provider unavailable")
	if _, err := service.Create(ctx, input); err == nil || calls != 1 {
		t.Fatal("failed creation emitted successful intent")
	}
	input.UserID = ""
	if _, err := service.Create(ctx, input); err == nil || calls != 1 {
		t.Fatal("invalid request emitted successful intent")
	}
}
