package app

import (
	"context"
	"errors"
	"testing"

	"github.com/brizenchi/go-modules/modules/billing/domain"
	"github.com/brizenchi/go-modules/modules/billing/port"
)

func TestOriginReturnURLValidator_ExactOriginOnly(t *testing.T) {
	validator, err := NewOriginReturnURLValidator("https://app.example.com/login/callback")
	if err != nil {
		t.Fatalf("NewOriginReturnURLValidator: %v", err)
	}
	for _, allowed := range []string{
		"https://app.example.com/billing/success?session_id=x",
		"https://APP.EXAMPLE.COM:443/account",
	} {
		if err := validator.ValidateReturnURL(allowed); err != nil {
			t.Errorf("allowed URL %q: %v", allowed, err)
		}
	}
	for _, rejected := range []string{
		"https://evil.example/account",
		"https://app.example.com.evil.example/account",
		"http://app.example.com/account",
		"https://app.example.com:444/account",
		"//app.example.com/account",
		"javascript:alert(1)",
		"https://user@app.example.com/account",
		" https://app.example.com/account",
	} {
		if err := validator.ValidateReturnURL(rejected); !errors.Is(err, domain.ErrInvalidReturnURL) {
			t.Errorf("rejected URL %q error = %v, want ErrInvalidReturnURL", rejected, err)
		}
	}
}

func TestCheckoutAndPortalRejectExternalReturnURLsBeforeProvider(t *testing.T) {
	validator, err := NewOriginReturnURLValidator("https://app.example.com")
	if err != nil {
		t.Fatalf("NewOriginReturnURLValidator: %v", err)
	}
	provider := newMockProvider()
	customers := newMockCustomerStore(port.Customer{
		UserID:             "u1",
		Email:              "user@example.com",
		ProviderCustomerID: "cus_1",
	})
	checkout := NewCheckoutService(provider, customers, validator)
	_, err = checkout.Create(context.Background(), CheckoutInput{
		UserID:      "u1",
		ProductType: domain.ProductCredits,
		Quantity:    1,
		SuccessURL:  "https://evil.example/collect",
		CancelURL:   "https://app.example.com/billing",
	})
	if !errors.Is(err, domain.ErrInvalidReturnURL) || provider.checkoutCalls != 0 {
		t.Fatalf("checkout error=%v provider calls=%d", err, provider.checkoutCalls)
	}

	subscriptions := NewSubscriptionService(provider, customers, newMockBus(), validator)
	_, err = subscriptions.OpenBillingPortal(context.Background(), "u1", "https://app.example.com.evil.example/account")
	if !errors.Is(err, domain.ErrInvalidReturnURL) {
		t.Fatalf("portal error=%v, want ErrInvalidReturnURL", err)
	}
}
