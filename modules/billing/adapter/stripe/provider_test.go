package stripe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

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
	var idempotencyKey string
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
		idempotencyKey = r.Header.Get("Idempotency-Key")
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
	if idempotencyKey != stableIdempotencyKey("customer", "user-1") {
		t.Fatalf("customer idempotency key = %q", idempotencyKey)
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
			_, _ = w.Write([]byte(`{"error":{"code":"resource_missing","message":"not found","type":"invalid_request_error"}}`))
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

func TestEnsureCustomer_DoesNotReplaceExistingOnTransientError(t *testing.T) {
	calls := 0
	posts := 0
	p := stripeMock(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method == "POST" {
			posts++
		}
		if r.Method != "GET" || !strings.HasPrefix(r.URL.Path, "/v1/customers/cus_existing") {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":{"message":"temporarily unavailable","type":"api_error"}}`))
	})

	_, err := p.EnsureCustomer(context.Background(), "u", "e@x", "cus_existing")
	if err == nil || !strings.Contains(err.Error(), "verify existing customer") {
		t.Fatalf("error = %v, want transient lookup failure", err)
	}
	if calls == 0 || posts != 0 {
		t.Fatalf("calls = %d posts = %d, want lookup retries but no replacement POST", calls, posts)
	}
}

func TestCreateCheckout_Subscription(t *testing.T) {
	var captured url.Values
	var idempotencyKey string
	p := stripeMock(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v1/checkout/sessions" {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		captured = readForm(t, r)
		idempotencyKey = r.Header.Get("Idempotency-Key")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"cs_test_1","url":"https://stripe.test/cs_test_1","object":"checkout.session"}`))
	})

	res, err := p.CreateCheckout(context.Background(), domain.CheckoutInput{
		UserID:                "user-1",
		Email:                 "real@x.test",
		ProductType:           domain.ProductSubscription,
		Plan:                  domain.PlanStarter,
		Interval:              domain.IntervalMonthly,
		SuccessURL:            "https://app.test/ok",
		CancelURL:             "https://app.test/cancel",
		CheckoutReservationID: "reservation-123",
		CheckoutExpiresAt:     time.Unix(2_000_000_000, 0).UTC(),
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
	if want := stableIdempotencyKey("checkout-reservation", "reservation-123"); idempotencyKey != want {
		t.Fatalf("idempotency key = %q, want %q", idempotencyKey, want)
	}
	if got := captured.Get("expires_at"); got != "2000000000" {
		t.Fatalf("expires_at = %q, want reservation expiry", got)
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
	if got := captured.Get("metadata[checkout_reservation_id]"); got != "reservation-123" {
		t.Errorf("checkout reservation metadata = %q", got)
	}
}

func TestCreateCheckout_SubscriptionMetadataContainsResolvedTrial(t *testing.T) {
	var captured url.Values
	var idempotencyKey string
	p := stripeMock(t, func(w http.ResponseWriter, r *http.Request) {
		captured = readForm(t, r)
		idempotencyKey = r.Header.Get("Idempotency-Key")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"cs_trial","url":"https://stripe.test/cs_trial","object":"checkout.session"}`))
	})
	p.cfg.TrialDays = 14

	_, err := p.CreateCheckout(context.Background(), domain.CheckoutInput{
		UserID:      "user-trial",
		Email:       "trial@example.com",
		ProductType: domain.ProductSubscription,
		Plan:        domain.PlanStarter,
		Interval:    domain.IntervalMonthly,
		SuccessURL:  "https://app.test/ok",
		CancelURL:   "https://app.test/cancel",
	})
	if err != nil {
		t.Fatalf("CreateCheckout: %v", err)
	}
	if got := captured.Get("subscription_data[trial_period_days]"); got != "14" {
		t.Fatalf("trial period = %q, want 14", got)
	}
	if got := captured.Get("metadata[trial_days]"); got != "14" {
		t.Fatalf("trial metadata = %q, want 14", got)
	}
	wantKey := stableIdempotencyKey(
		"checkout",
		"user-trial",
		"",
		"subscription",
		"starter",
		"monthly",
		"price_starter_m",
		"1",
		"14",
		"https://app.test/ok",
		"https://app.test/cancel",
	)
	if idempotencyKey != wantKey {
		t.Fatalf("idempotency key = %q, want resolved-trial key %q", idempotencyKey, wantKey)
	}
}

func TestStableIdempotencyKey(t *testing.T) {
	first := stableIdempotencyKey("checkout", "u1", "subscription", "pro", "monthly")
	second := stableIdempotencyKey("checkout", "u1", "subscription", "pro", "monthly")
	different := stableIdempotencyKey("checkout", "u1", "subscription", "pro", "yearly")
	if first == "" || first != second {
		t.Fatalf("same checkout input must produce the same key: %q vs %q", first, second)
	}
	if first == different {
		t.Fatalf("different checkout target must produce a different key: %q", first)
	}
}

func TestGetCheckoutSession_MapsAuthoritativeStripeState(t *testing.T) {
	p := stripeMock(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !strings.HasPrefix(r.URL.Path, "/v1/checkout/sessions/") {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		id := strings.TrimPrefix(r.URL.Path, "/v1/checkout/sessions/")
		status := strings.TrimPrefix(id, "cs_")
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"id":%q,"object":"checkout.session","status":%q,"payment_status":"unpaid"}`, id, status)
	})
	for _, tt := range []struct {
		stripe string
		want   domain.CheckoutSessionState
	}{
		{stripe: "open", want: domain.CheckoutSessionOpen},
		{stripe: "complete", want: domain.CheckoutSessionComplete},
		{stripe: "expired", want: domain.CheckoutSessionExpired},
	} {
		got, err := p.GetCheckoutSession(context.Background(), "cs_"+tt.stripe)
		if err != nil {
			t.Fatalf("GetCheckoutSession(%s): %v", tt.stripe, err)
		}
		if got.State != tt.want || got.SessionID != "cs_"+tt.stripe {
			t.Fatalf("snapshot = %+v, want state %q", got, tt.want)
		}
	}
}

func TestFindCheckoutSessionUsesCustomerAndReservationMetadata(t *testing.T) {
	p := stripeMock(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/checkout/sessions" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("customer"); got != "cus_recover" {
			t.Fatalf("customer query = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"object":"list",
			"has_more":false,
			"data":[
				{"id":"cs_other","object":"checkout.session","status":"expired","metadata":{"checkout_reservation_id":"other"}},
				{"id":"cs_recovered","object":"checkout.session","url":"https://checkout.stripe.test/recovered","status":"open","payment_status":"unpaid","metadata":{"checkout_reservation_id":"reservation-1"}}
			]
		}`))
	})

	got, err := p.FindCheckoutSession(context.Background(), "cus_recover", "reservation-1")
	if err != nil {
		t.Fatalf("FindCheckoutSession: %v", err)
	}
	if got == nil || got.SessionID != "cs_recovered" || got.State != domain.CheckoutSessionOpen || got.CheckoutURL == "" {
		t.Fatalf("snapshot = %+v", got)
	}
}

func TestFindCheckoutSessionAuthoritativeMiss(t *testing.T) {
	p := stripeMock(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","has_more":false,"data":[]}`))
	})
	got, err := p.FindCheckoutSession(context.Background(), "cus_recover", "reservation-missing")
	if err != nil || got != nil {
		t.Fatalf("snapshot=%+v error=%v, want authoritative nil", got, err)
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
		PriceID:     "price_1CreditsBravo123456789",
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
	if got := captured.Get("line_items[0][price]"); got != "price_1CreditsAlpha123456789" {
		t.Errorf("price = %q, want first server-configured price", got)
	}
	if got := captured.Get("line_items[0][quantity]"); got != "3" {
		t.Errorf("quantity = %q", got)
	}
	if got := captured.Get("line_items[0][adjustable_quantity][enabled]"); got != "true" {
		t.Errorf("adjustable_quantity not set: %v", captured)
	}
}

func TestCreateCheckout_CreditsUsesFirstConfiguredPriceWhenOmitted(t *testing.T) {
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
		Quantity:    1,
		SuccessURL:  "https://app.test/ok",
		CancelURL:   "https://app.test/cancel",
	})
	if err != nil {
		t.Fatalf("CreateCheckout: %v", err)
	}
	if got := captured.Get("line_items[0][price]"); got != "price_1CreditsAlpha123456789" {
		t.Fatalf("price = %q, want first configured credits price", got)
	}
}

func TestCreateCheckout_CreditsSkipsInvalidConfiguredPrice(t *testing.T) {
	var captured url.Values
	p := stripeMock(t, func(w http.ResponseWriter, r *http.Request) {
		captured = readForm(t, r)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"cs_credits","url":"u","object":"checkout.session"}`))
	})
	p.cfg.CreditsPriceIDs = []string{"price_placeholder", "price_1CreditsUsable123456789"}

	_, err := p.CreateCheckout(context.Background(), domain.CheckoutInput{
		UserID:      "u",
		Email:       "e@x",
		ProductType: domain.ProductCredits,
		Quantity:    1,
		SuccessURL:  "https://app.test/ok",
		CancelURL:   "https://app.test/cancel",
	})
	if err != nil {
		t.Fatalf("CreateCheckout: %v", err)
	}
	if got := captured.Get("line_items[0][price]"); got != "price_1CreditsUsable123456789" {
		t.Fatalf("price = %q, want first valid configured price", got)
	}
}

func TestCreateCheckout_CreditsRejectsQuantityOutsideStripeBounds(t *testing.T) {
	for _, quantity := range []int64{-1, 0, 101} {
		t.Run(strconv.FormatInt(quantity, 10), func(t *testing.T) {
			p := NewProvider(newTestConfig())
			_, err := p.CreateCheckout(context.Background(), domain.CheckoutInput{
				ProductType: domain.ProductCredits,
				Quantity:    quantity,
			})
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("error = %v, want ErrInvalidInput", err)
			}
		})
	}
}

func TestCreateCheckout_Lifetime(t *testing.T) {
	var captured url.Values
	p := stripeMock(t, func(w http.ResponseWriter, r *http.Request) {
		captured = readForm(t, r)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"cs_lifetime","url":"https://stripe.test/lifetime","object":"checkout.session"}`))
	})

	res, err := p.CreateCheckout(context.Background(), domain.CheckoutInput{
		UserID:      "u",
		Email:       "e@x",
		ProductType: domain.ProductLifetime,
		SuccessURL:  "https://app.test/ok",
		CancelURL:   "https://app.test/cancel",
	})
	if err != nil {
		t.Fatalf("CreateCheckout: %v", err)
	}
	if res.SessionID != "cs_lifetime" {
		t.Fatalf("session id = %q", res.SessionID)
	}
	if got := captured.Get("mode"); got != "payment" {
		t.Errorf("mode = %q", got)
	}
	if got := captured.Get("line_items[0][price]"); got != "price_lifetime" {
		t.Errorf("price = %q", got)
	}
	if got := captured.Get("line_items[0][quantity]"); got != "1" {
		t.Errorf("quantity = %q", got)
	}
	if got := captured.Get("metadata[plan]"); got != "lifetime" {
		t.Errorf("plan metadata = %q", got)
	}
	if got := captured.Get("metadata[product_type]"); got != "lifetime" {
		t.Errorf("product_type metadata = %q", got)
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

func TestSnapshotFromSubscription_ExplicitCancelAtIsCanceling(t *testing.T) {
	p := NewProvider(newTestConfig())
	cancelAt := time.Now().Add(72 * time.Hour).UTC().Truncate(time.Second)
	snap := p.snapshotFromSubscription(&stripesdk.Subscription{
		ID:                "sub_cancel_at",
		Status:            stripesdk.SubscriptionStatusActive,
		CancelAt:          cancelAt.Unix(),
		CancelAtPeriodEnd: false,
	})
	if snap.Status != domain.StatusCanceling || snap.CancelAtPeriodEnd {
		t.Fatalf("snapshot = %+v, want canceling with cancel_at_period_end=false", snap)
	}
	if snap.CancelEffectiveAt == nil || !snap.CancelEffectiveAt.Equal(cancelAt) {
		t.Fatalf("cancel effective at = %v, want %v", snap.CancelEffectiveAt, cancelAt)
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
	if got := captured.Get("cancel_at"); got != "0" {
		t.Errorf("cancel_at = %q, want 0", got)
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

func TestSubscriptionMutationPathsRejectNilItems(t *testing.T) {
	tests := []struct {
		name string
		call func(*Provider) error
	}{
		{
			name: "change",
			call: func(p *Provider) error {
				_, err := p.ChangeSubscription(context.Background(), "sub_1", domain.SubscriptionChangeInput{
					Plan: domain.PlanPro, Interval: domain.IntervalMonthly,
				})
				return err
			},
		},
		{
			name: "schedule",
			call: func(p *Provider) error {
				_, err := p.ScheduleSubscriptionChange(context.Background(), "sub_1", domain.SubscriptionChangeInput{
					Plan: domain.PlanPro, Interval: domain.IntervalMonthly,
				})
				return err
			},
		},
		{
			name: "preview",
			call: func(p *Provider) error {
				_, err := p.PreviewSubscriptionChange(context.Background(), "cus_1", "sub_1", domain.SubscriptionPreviewInput{
					Plan: domain.PlanPro, Interval: domain.IntervalMonthly,
				})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := stripeMock(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" || !strings.HasPrefix(r.URL.Path, "/v1/subscriptions/sub_1") {
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"id":"sub_1","object":"subscription"}`))
			})

			err := tt.call(p)
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("error = %v, want ErrInvalidInput", err)
			}
		})
	}
}

func TestScheduleSourceSubscriptionItemRejectsMalformedResponses(t *testing.T) {
	validItem := func() *stripesdk.SubscriptionItem {
		return &stripesdk.SubscriptionItem{
			ID:       "si_1",
			Price:    &stripesdk.Price{ID: "price_starter_m"},
			Quantity: 1,
		}
	}
	withItem := func(item *stripesdk.SubscriptionItem) *stripesdk.Subscription {
		return &stripesdk.Subscription{Items: &stripesdk.SubscriptionItemList{Data: []*stripesdk.SubscriptionItem{item}}}
	}

	tests := []struct {
		name string
		sub  *stripesdk.Subscription
	}{
		{name: "nil subscription", sub: nil},
		{name: "nil items", sub: &stripesdk.Subscription{}},
		{name: "empty items", sub: &stripesdk.Subscription{Items: &stripesdk.SubscriptionItemList{}}},
		{name: "nil item", sub: withItem(nil)},
		{name: "empty item id", sub: withItem(&stripesdk.SubscriptionItem{Price: &stripesdk.Price{ID: "price_starter_m"}, Quantity: 1})},
		{name: "nil price", sub: withItem(&stripesdk.SubscriptionItem{ID: "si_1", Quantity: 1})},
		{name: "empty price id", sub: withItem(&stripesdk.SubscriptionItem{ID: "si_1", Price: &stripesdk.Price{}, Quantity: 1})},
		{name: "zero quantity", sub: withItem(&stripesdk.SubscriptionItem{ID: "si_1", Price: &stripesdk.Price{ID: "price_starter_m"}})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := scheduleSourceSubscriptionItem(tt.sub); !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("error = %v, want ErrInvalidInput", err)
			}
		})
	}
	if item, err := scheduleSourceSubscriptionItem(withItem(validItem())); err != nil || item.ID != "si_1" {
		t.Fatalf("valid item = (%+v, %v)", item, err)
	}
}

func TestValidSubscriptionScheduleIDRejectsNilAndEmpty(t *testing.T) {
	for _, schedule := range []*stripesdk.SubscriptionSchedule{nil, {}} {
		if _, err := validSubscriptionScheduleID(schedule); !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("schedule=%+v error=%v, want ErrInvalidInput", schedule, err)
		}
	}
	if id, err := validSubscriptionScheduleID(&stripesdk.SubscriptionSchedule{ID: " ss_1 "}); err != nil || id != "ss_1" {
		t.Fatalf("valid schedule = (%q, %v)", id, err)
	}
}

func TestScheduleSubscriptionChangeRejectsEmptyCreatedSchedule(t *testing.T) {
	requests := 0
	p := stripeMock(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		switch requests {
		case 1:
			if r.Method != "GET" || !strings.HasPrefix(r.URL.Path, "/v1/subscriptions/sub_1") {
				t.Fatalf("unexpected first request: %s %s", r.Method, r.URL.Path)
			}
			_, _ = w.Write([]byte(`{
				"id":"sub_1","object":"subscription",
				"items":{"object":"list","data":[{
					"id":"si_1","quantity":1,"price":{"id":"price_starter_m"}
				}]}
			}`))
		case 2:
			if r.Method != "POST" || r.URL.Path != "/v1/subscription_schedules" {
				t.Fatalf("unexpected second request: %s %s", r.Method, r.URL.Path)
			}
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Fatalf("unexpected extra request %d", requests)
		}
	})

	_, err := p.ScheduleSubscriptionChange(context.Background(), "sub_1", domain.SubscriptionChangeInput{
		Plan: domain.PlanPro, Interval: domain.IntervalMonthly,
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("error = %v, want ErrInvalidInput", err)
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
