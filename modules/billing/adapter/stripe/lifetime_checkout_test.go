package stripe

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/brizenchi/go-modules/modules/billing/domain"
)

func TestLifetimeCheckoutRejectsInvalidPriceBeforeCreatingSession(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
	}{
		{"recurring", `{"id":"price_lifetime","active":true,"type":"recurring","recurring":{"interval":"month"}}`, 200},
		{"inactive", `{"id":"price_lifetime","active":false,"type":"one_time"}`, 200},
		{"missing", `{"error":{"type":"invalid_request_error","code":"resource_missing","param":"price","message":"fixture price does not exist"}}`, 404},
	} {
		t.Run(tc.name, func(t *testing.T) {
			creates := 0
			p := stripeMock(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.Method == http.MethodGet && r.URL.Path == "/v1/prices/price_lifetime" {
					w.WriteHeader(tc.status)
					_, _ = w.Write([]byte(tc.body))
					return
				}
				creates++
				_, _ = w.Write([]byte(`{"id":"cs_not-a-real-session","url":"https://checkout.stripe.com/test"}`))
			})
			_, err := p.CreateCheckout(context.Background(), domain.CheckoutInput{ProductType: domain.ProductLifetime})
			if !errors.Is(err, domain.ErrPriceConfiguration) || creates != 0 {
				t.Fatalf("error=%v creates=%d; want price configuration error without checkout", err, creates)
			}
		})
	}
}

func TestLifetimeCheckoutDoesNotHidePriceReadFailuresOrCreateSession(t *testing.T) {
	creates := 0
	p := stripeMock(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			creates++
		}
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"type":"authentication_error","message":"fixture authentication failed"}}`))
	})
	_, err := p.CreateCheckout(context.Background(), domain.CheckoutInput{ProductType: domain.ProductLifetime})
	if err == nil || errors.Is(err, domain.ErrPriceConfiguration) || creates != 0 {
		t.Fatalf("error=%v creates=%d; want original read failure without checkout", err, creates)
	}
}
