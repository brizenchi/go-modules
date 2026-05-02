package stripe

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/brizenchi/go-modules/modules/billing/domain"
	stripesdk "github.com/stripe/stripe-go/v76"
)

// stripeMock spins up an httptest.Server, redirects the global stripe API
// backend at it for the test's lifetime, and returns a *Provider.
//
// Tests using stripeMock must NOT run in parallel — the stripe SDK uses
// global backend state that we reset on cleanup.
func stripeMock(t *testing.T, handler http.HandlerFunc) *Provider {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	prevKey := stripesdk.Key
	stripesdk.Key = "sk_test_mock"

	backend := stripesdk.GetBackendWithConfig(stripesdk.APIBackend, &stripesdk.BackendConfig{
		URL:        stripesdk.String(srv.URL),
		HTTPClient: srv.Client(),
	})
	stripesdk.SetBackend(stripesdk.APIBackend, backend)

	t.Cleanup(func() {
		stripesdk.Key = prevKey
		// Restore the default backend so the next test starts clean.
		stripesdk.SetBackend(stripesdk.APIBackend, stripesdk.GetBackendWithConfig(
			stripesdk.APIBackend, &stripesdk.BackendConfig{},
		))
	})

	return NewProvider(newTestConfig())
}

func readForm(t *testing.T, r *http.Request) url.Values {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	v, err := url.ParseQuery(string(body))
	if err != nil {
		t.Fatalf("parse form: %v", err)
	}
	return v
}

func TestEnsureCustomer_DisabledProvider(t *testing.T) {
	p := NewProvider(Config{Enabled: false})
	_, err := p.EnsureCustomer(context.Background(), "u", "e@x", "")
	if !errors.Is(err, domain.ErrProviderDisabled) {
		t.Fatalf("want ErrProviderDisabled, got %v", err)
	}
}

func TestEnsureCustomer_CreatesNew(t *testing.T) {
	p := stripeMock(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v1/customers" {
			t.Errorf("unexpected req: %s %s", r.Method, r.URL.Path)
		}
		form := readForm(t, r)
		if form.Get("email") != "alice@x.test" {
			t.Errorf("email = %q", form.Get("email"))
		}
		if form.Get("metadata[user_id]") != "user-1" {
			t.Errorf("user_id metadata = %q", form.Get("metadata[user_id]"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"cus_new123","object":"customer"}`))
	})

	id, err := p.EnsureCustomer(context.Background(), "user-1", "alice@x.test", "")
	if err != nil {
		t.Fatalf("EnsureCustomer: %v", err)
	}
	if id != "cus_new123" {
		t.Errorf("customer id = %q", id)
	}
}

func TestEnsureCustomer_ReusesExisting(t *testing.T) {
	p := stripeMock(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || !strings.HasPrefix(r.URL.Path, "/v1/customers/cus_existing") {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"cus_existing","object":"customer"}`))
	})

	id, err := p.EnsureCustomer(context.Background(), "u", "e@x", "cus_existing")
	if err != nil {
		t.Fatalf("EnsureCustomer: %v", err)
	}
	if id != "cus_existing" {
		t.Errorf("id = %q", id)
	}
}

func TestEnsureCustomer_RecreatesWhenExistingMissing(t *testing.T) {
	calls := 0
	p := stripeMock(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		switch {
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/v1/customers/cus_gone"):
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":{"message":"not found","type":"invalid_request_error"}}`))
		case r.Method == "POST" && r.URL.Path == "/v1/customers":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"cus_fresh","object":"customer"}`))
		default:
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
	})

	id, err := p.EnsureCustomer(context.Background(), "u", "e@x", "cus_gone")
	if err != nil {
		t.Fatalf("EnsureCustomer: %v", err)
	}
	if id != "cus_fresh" {
		t.Errorf("id = %q", id)
	}
	if calls != 2 {
		t.Errorf("expected 2 calls (GET + POST), got %d", calls)
	}
}

func TestCreateCheckout_Subscription(t *testing.T) {
	var captured url.Values
	p := stripeMock(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v1/checkout/sessions" {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		captured = readForm(t, r)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"cs_test_1","url":"https://stripe.test/cs_test_1","object":"checkout.session"}`))
	})

	res, err := p.CreateCheckout(context.Background(), domain.CheckoutInput{
		UserID:      "user-1",
		Email:       "real@x.test",
		ProductType: domain.ProductSubscription,
		Plan:        domain.PlanStarter,
		Interval:    domain.IntervalMonthly,
		SuccessURL:  "https://app.test/ok",
		CancelURL:   "https://app.test/cancel",
		Metadata: map[string]string{
			"referral": "rwf_xyz",
			"user_id":  "spoof", // attempt to override
		},
	})
	if err != nil {
		t.Fatalf("CreateCheckout: %v", err)
	}
	if res.SessionID != "cs_test_1" || res.CheckoutURL != "https://stripe.test/cs_test_1" {
		t.Errorf("unexpected result: %+v", res)
	}

	// Subscription mode + correct price.
	if got := captured.Get("mode"); got != "subscription" {
		t.Errorf("mode = %q", got)
	}
	if got := captured.Get("line_items[0][price]"); got != "price_starter_m" {
		t.Errorf("price = %q", got)
	}
	// client_reference_id always set to user_id.
	if got := captured.Get("client_reference_id"); got != "user-1" {
		t.Errorf("client_reference_id = %q", got)
	}
	// Metadata: referral preserved, system fields win.
	if got := captured.Get("metadata[referral]"); got != "rwf_xyz" {
		t.Errorf("referral metadata = %q", got)
	}
	if got := captured.Get("metadata[user_id]"); got != "user-1" {
		t.Errorf("user_id metadata should be system value, got %q", got)
	}
	if got := captured.Get("metadata[plan]"); got != "starter" {
		t.Errorf("plan metadata = %q", got)
	}
}

func TestCreateCheckout_Credits(t *testing.T) {
	var captured url.Values
	p := stripeMock(t, func(w http.ResponseWriter, r *http.Request) {
		captured = readForm(t, r)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"cs_credits","url":"u","object":"checkout.session"}`))
	})

	_, err := p.CreateCheckout(context.Background(), domain.CheckoutInput{
		UserID:      "u",
		Email:       "e@x",
		ProductType: domain.ProductCredits,
		PriceID:     "price_credits_a",
		Quantity:    3,
		SuccessURL:  "https://app.test/ok",
		CancelURL:   "https://app.test/cancel",
	})
	if err != nil {
		t.Fatalf("CreateCheckout: %v", err)
	}
	if got := captured.Get("mode"); got != "payment" {
		t.Errorf("mode = %q", got)
	}
	if got := captured.Get("line_items[0][price]"); got != "price_credits_a" {
		t.Errorf("price = %q", got)
	}
	if got := captured.Get("line_items[0][quantity]"); got != "3" {
		t.Errorf("quantity = %q", got)
	}
	if got := captured.Get("line_items[0][adjustable_quantity][enabled]"); got != "true" {
		t.Errorf("adjustable_quantity not set: %v", captured)
	}
}

func TestCreateCheckout_DisabledProvider(t *testing.T) {
	p := NewProvider(Config{Enabled: false})
	_, err := p.CreateCheckout(context.Background(), domain.CheckoutInput{})
	if !errors.Is(err, domain.ErrProviderDisabled) {
		t.Fatalf("want ErrProviderDisabled, got %v", err)
	}
}

func TestCreateCheckout_UnknownPriceForPlan(t *testing.T) {
	// PlanPro has no yearly configured in newTestConfig — should fail
	// without hitting the network.
	p := stripeMock(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	})
	_, err := p.CreateCheckout(context.Background(), domain.CheckoutInput{
		UserID:      "u",
		Email:       "e@x",
		ProductType: domain.ProductSubscription,
		Plan:        domain.PlanPro,
		Interval:    domain.IntervalYearly,
		SuccessURL:  "s",
		CancelURL:   "c",
	})
	if !errors.Is(err, domain.ErrPriceNotFound) {
		t.Fatalf("want ErrPriceNotFound, got %v", err)
	}
}

func TestCreateCheckout_UnknownProductType(t *testing.T) {
	p := stripeMock(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request")
	})
	_, err := p.CreateCheckout(context.Background(), domain.CheckoutInput{
		UserID:      "u",
		ProductType: domain.ProductType("garbage"),
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
}

func TestCreateCheckout_StripeError(t *testing.T) {
	p := stripeMock(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{
				"message": "bad",
				"type":    "invalid_request_error",
			},
		})
	})
	_, err := p.CreateCheckout(context.Background(), domain.CheckoutInput{
		UserID:      "u",
		Email:       "e@x",
		ProductType: domain.ProductSubscription,
		Plan:        domain.PlanStarter,
		Interval:    domain.IntervalMonthly,
		SuccessURL:  "s",
		CancelURL:   "c",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCancelSubscription_AtPeriodEnd(t *testing.T) {
	var captured url.Values
	var path string
	p := stripeMock(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		captured = readForm(t, r)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"sub_1","object":"subscription"}`))
	})

	if err := p.CancelSubscription(context.Background(), "sub_1", domain.CancelAtPeriodEnd); err != nil {
		t.Fatalf("CancelSubscription: %v", err)
	}
	if !strings.HasPrefix(path, "/v1/subscriptions/sub_1") {
		t.Errorf("path = %q", path)
	}
	if got := captured.Get("cancel_at_period_end"); got != "true" {
		t.Errorf("cancel_at_period_end = %q", got)
	}
}

func TestCancelSubscription_DisabledAndInvalid(t *testing.T) {
	disabled := NewProvider(Config{Enabled: false})
	if err := disabled.CancelSubscription(context.Background(), "sub", domain.CancelAtPeriodEnd); !errors.Is(err, domain.ErrProviderDisabled) {
		t.Errorf("disabled: %v", err)
	}
	enabled := NewProvider(newTestConfig())
	if err := enabled.CancelSubscription(context.Background(), "", domain.CancelAtPeriodEnd); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("empty id: %v", err)
	}
	if err := enabled.CancelSubscription(context.Background(), "sub", domain.CancelMode("garbage")); !errors.Is(err, domain.ErrInvalidCancelMode) {
		t.Errorf("bad mode: %v", err)
	}
}

func TestGetSubscription(t *testing.T) {
	p := stripeMock(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || !strings.HasPrefix(r.URL.Path, "/v1/subscriptions/sub_1") {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "sub_1",
			"object": "subscription",
			"customer": "cus_1",
			"status": "active",
			"cancel_at_period_end": false,
			"items": {
				"object": "list",
				"data": [{
					"id": "si_1",
					"price": {
						"id": "price_starter_m",
						"product": "prod_starter",
						"recurring": {"interval":"month"}
					}
				}]
			}
		}`))
	})

	snap, err := p.GetSubscription(context.Background(), "sub_1")
	if err != nil {
		t.Fatalf("GetSubscription: %v", err)
	}
	if snap.ProviderSubscriptionID != "sub_1" {
		t.Errorf("ID = %q", snap.ProviderSubscriptionID)
	}
	if snap.Plan != domain.PlanStarter || snap.Interval != domain.IntervalMonthly {
		t.Errorf("plan/interval = %s/%s", snap.Plan, snap.Interval)
	}
}

func TestGetDefaultPaymentMethod(t *testing.T) {
	p := stripeMock(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "cus_1",
			"object": "customer",
			"invoice_settings": {
				"default_payment_method": {
					"id": "pm_1",
					"card": {"brand":"visa","last4":"4242","exp_month":12,"exp_year":2030}
				}
			}
		}`))
	})

	card, err := p.GetDefaultPaymentMethod(context.Background(), "cus_1")
	if err != nil {
		t.Fatalf("GetDefaultPaymentMethod: %v", err)
	}
	if card == nil || card.Brand != "visa" || card.Last4 != "4242" || card.ExpMonth != 12 || card.ExpYear != 2030 {
		t.Errorf("card = %+v", card)
	}
}

func TestGetDefaultPaymentMethod_NoCard(t *testing.T) {
	p := stripeMock(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"cus_1","object":"customer","invoice_settings":{"default_payment_method":null}}`))
	})

	card, err := p.GetDefaultPaymentMethod(context.Background(), "cus_1")
	if err != nil {
		t.Fatal(err)
	}
	if card != nil {
		t.Errorf("expected nil, got %+v", card)
	}
}

func TestGetDefaultPaymentMethod_EmptyCustomer(t *testing.T) {
	p := NewProvider(newTestConfig())
	card, err := p.GetDefaultPaymentMethod(context.Background(), "")
	if err != nil || card != nil {
		t.Errorf("expected (nil,nil), got (%v,%v)", card, err)
	}
}

func TestListInvoices(t *testing.T) {
	p := stripeMock(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/v1/invoices") {
			t.Errorf("unexpected: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"object": "list",
			"has_more": false,
			"data": [
				{"id":"in_1","object":"invoice","amount_paid":1999,"status":"paid","created":1700000000,"invoice_pdf":"https://x/1.pdf","period_end":1700000000},
				{"id":"in_2","object":"invoice","amount_paid":2999,"status":"paid","created":1702000000,"invoice_pdf":"https://x/2.pdf","period_end":1702000000}
			]
		}`))
	})

	items, _, err := p.ListInvoices(context.Background(), "cus_1", 1, 10)
	if err != nil {
		t.Fatalf("ListInvoices: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len = %d, want 2", len(items))
	}
	if items[0].ID != "in_1" || items[0].AmountUSD != 19.99 {
		t.Errorf("item[0] = %+v", items[0])
	}
}

func TestReactivateSubscription(t *testing.T) {
	var captured url.Values
	p := stripeMock(t, func(w http.ResponseWriter, r *http.Request) {
		captured = readForm(t, r)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"sub_1","object":"subscription"}`))
	})

	if err := p.ReactivateSubscription(context.Background(), "sub_1"); err != nil {
		t.Fatalf("Reactivate: %v", err)
	}
	if got := captured.Get("cancel_at_period_end"); got != "false" {
		t.Errorf("cancel_at_period_end = %q", got)
	}
}

func TestChangeSubscription(t *testing.T) {
	requests := 0
	var updateForm url.Values
	p := stripeMock(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		switch requests {
		case 1:
			if r.Method != "GET" || !strings.HasPrefix(r.URL.Path, "/v1/subscriptions/sub_1") {
				t.Errorf("unexpected first request: %s %s", r.Method, r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"id":"sub_1",
				"object":"subscription",
				"customer":"cus_1",
				"items":{"object":"list","data":[{"id":"si_1","price":{"id":"price_starter_m","product":"prod_starter","recurring":{"interval":"month"}}}]}
			}`))
		case 2:
			if r.Method != "POST" || !strings.HasPrefix(r.URL.Path, "/v1/subscriptions/sub_1") {
				t.Errorf("unexpected second request: %s %s", r.Method, r.URL.Path)
			}
			updateForm = readForm(t, r)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"id":"sub_1",
				"object":"subscription",
				"customer":"cus_1",
				"status":"active",
				"cancel_at_period_end":false,
				"items":{"object":"list","data":[{"id":"si_1","price":{"id":"price_pro_m","product":"prod_pro","recurring":{"interval":"month"}}}]}
			}`))
		default:
			t.Fatalf("unexpected extra request %d", requests)
		}
	})

	snap, err := p.ChangeSubscription(context.Background(), "sub_1", domain.SubscriptionChangeInput{
		Plan:     domain.PlanPro,
		Interval: domain.IntervalMonthly,
	})
	if err != nil {
		t.Fatalf("ChangeSubscription: %v", err)
	}
	if got := updateForm.Get("items[0][id]"); got != "si_1" {
		t.Errorf("item id = %q", got)
	}
	if got := updateForm.Get("items[0][price]"); got != "price_pro_m" {
		t.Errorf("price = %q", got)
	}
	if got := updateForm.Get("proration_behavior"); got != "always_invoice" {
		t.Errorf("proration_behavior = %q", got)
	}
	if got := updateForm.Get("payment_behavior"); got != "pending_if_incomplete" {
		t.Errorf("payment_behavior = %q", got)
	}
	if got := updateForm.Get("billing_cycle_anchor"); got != "unchanged" {
		t.Errorf("billing_cycle_anchor = %q", got)
	}
	if snap.Plan != domain.PlanPro || snap.Interval != domain.IntervalMonthly {
		t.Errorf("unexpected snapshot: %+v", snap)
	}
}

func TestCreateBillingPortalSession(t *testing.T) {
	var captured url.Values
	p := stripeMock(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v1/billing_portal/sessions" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		captured = readForm(t, r)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"billing_portal.session","url":"https://billing.stripe.test/session_123"}`))
	})

	res, err := p.CreateBillingPortalSession(context.Background(), "cus_1", "https://app.test/billing")
	if err != nil {
		t.Fatalf("CreateBillingPortalSession: %v", err)
	}
	if got := captured.Get("customer"); got != "cus_1" {
		t.Errorf("customer = %q", got)
	}
	if got := captured.Get("return_url"); got != "https://app.test/billing" {
		t.Errorf("return_url = %q", got)
	}
	if res.URL != "https://billing.stripe.test/session_123" {
		t.Errorf("url = %q", res.URL)
	}
}
