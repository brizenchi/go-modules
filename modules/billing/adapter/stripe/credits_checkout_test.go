package stripe

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestGetCreditsCheckoutVerifiesPaidStateAndActualLineItems(t *testing.T) {
	for _, tc := range []struct {
		name, status, payment, product, price string
		quantity                              int64
		wantError, wantPurchase               bool
	}{
		{"paid", "complete", "paid", "credits", "price_1CreditsAlpha123456789", 2, false, true},
		{"unpaid", "complete", "unpaid", "credits", "price_1CreditsAlpha123456789", 2, false, false},
		{"expired", "expired", "unpaid", "credits", "price_1CreditsAlpha123456789", 2, false, false},
		{"wrong-product", "complete", "paid", "lifetime", "price_1CreditsAlpha123456789", 2, true, false},
		{"wrong-price", "complete", "paid", "credits", "price_unknown", 2, true, false},
		{"zero-quantity", "complete", "paid", "credits", "price_1CreditsAlpha123456789", 0, true, false},
		{"too-many", "complete", "paid", "credits", "price_1CreditsAlpha123456789", 101, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := stripeMock(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" || r.URL.Path != "/v1/checkout/sessions/cs_example" || r.URL.Query().Get("expand[0]") != "line_items" {
					t.Errorf("request %s %s", r.Method, r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"id": "cs_example", "object": "checkout.session", "mode": "payment", "status": tc.status, "payment_status": tc.payment, "client_reference_id": "buyer", "metadata": map[string]string{"user_id": "buyer", "product_type": tc.product, "price_id": "price_1CreditsAlpha123456789", "quantity": "1"}, "line_items": map[string]any{"object": "list", "data": []any{map[string]any{"id": "li_example", "object": "item", "price": map[string]any{"id": tc.price, "object": "price"}, "quantity": tc.quantity}}}})
			})
			p.cfg.CreditsPriceIDs = []string{"price_1CreditsAlpha123456789"}
			p.cfg.CreditsPerUnit = 100
			result, err := p.GetCreditsCheckout(t.Context(), "cs_example")
			if (err != nil) != tc.wantError {
				t.Fatalf("result=%+v error=%v", result, err)
			}
			if err != nil {
				return
			}
			if (result.Purchase != nil) != tc.wantPurchase {
				t.Fatalf("purchase=%+v", result.Purchase)
			}
			if tc.wantPurchase && (result.Purchase.TotalCredits != 200 || result.Purchase.Quantity != 2 || result.Purchase.CheckoutSessionID != "cs_example") {
				t.Fatalf("must use settled quantity: %+v", result.Purchase)
			}
		})
	}
}

func TestGetCreditsCheckoutPropagatesCancellation(t *testing.T) {
	calls := 0
	p := stripeMock(t, func(http.ResponseWriter, *http.Request) { calls++ })
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := p.GetCreditsCheckout(ctx, "cs_example"); err == nil {
		t.Fatal("canceled read succeeded")
	}
	if calls != 0 {
		t.Fatalf("canceled context made %d requests", calls)
	}
}
