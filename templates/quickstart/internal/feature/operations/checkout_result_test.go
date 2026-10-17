package operations

import (
	"encoding/json"
	"testing"

	"context"
	"github.com/brizenchi/go-modules/modules/billing"
	billingdomain "github.com/brizenchi/go-modules/modules/billing/domain"
	billingevent "github.com/brizenchi/go-modules/modules/billing/event"
	"github.com/brizenchi/go-modules/modules/billing/port"
	"github.com/brizenchi/quickstart-template/internal/checkoutlog"
	"github.com/brizenchi/quickstart-template/internal/creditpurchase"
	"github.com/brizenchi/quickstart-template/internal/platform"
	"github.com/brizenchi/quickstart-template/internal/user"
)

type checkoutReader struct {
	port.Provider
	result *port.CreditsCheckout
	calls  int
}

func (r *checkoutReader) GetCreditsCheckout(context.Context, string) (*port.CreditsCheckout, error) {
	r.calls++
	return r.result, nil
}

func TestCreditCheckoutReturnConfirmsPaidPurchaseBeforeWebhook(t *testing.T) {
	f := newFixture(t)
	if err := f.db.Create(&checkoutlog.Record{UserID: "owner", Provider: "stripe", SessionID: "cs_return", ProductType: "credits", Quantity: 1}).Error; err != nil {
		t.Fatal(err)
	}
	reader := &checkoutReader{result: &port.CreditsCheckout{SessionID: "cs_return", UserID: "owner", State: "complete", PaymentStatus: "paid", Purchase: &billingevent.CreditsPurchased{CheckoutSessionID: "cs_return", Quantity: 2, CreditsPerUnit: 100, TotalCredits: 200, PriceID: "price_example"}}}
	f.module.deps.Modules = &platform.Modules{Billing: &billing.Module{Provider: reader}}
	for range 2 {
		w := request(f, "POST", "/billing/credit-checkouts/cs_return/confirmation", "", "user", "cs_return")
		if w.Code != 200 || data(t, w)["state"] != "fulfilled" || data(t, w)["credits_granted"] != float64(200) {
			t.Fatalf("confirmation: %d %s", w.Code, w.Body)
		}
	}
	if reader.calls != 1 {
		t.Fatalf("provider calls=%d; fulfilled purchases must not contact Stripe again", reader.calls)
	}
	summary, err := f.module.deps.Credits.GetCreditSummary(t.Context(), "owner")
	if err != nil || summary.Balance != 200 {
		t.Fatalf("balance=%d err=%v", summary.Balance, err)
	}
	w := request(f, "GET", "/billing/credit-checkouts/cs_return", "", "user", "")
	if w.Code != 200 || data(t, w)["state"] != "fulfilled" {
		t.Fatalf("read after return: %d %s", w.Code, w.Body)
	}
	// The webhook/recovery can arrive after the payment-return confirmation.
	if err := f.db.Create(&billingdomain.BillingEvent{UserID: "owner", Provider: "stripe", ProviderEventID: "evt_after_return", EventType: "checkout.session.completed", Payload: json.RawMessage(`{"data":{"object":{"id":"cs_return","payment_status":"paid"}}}`)}).Error; err != nil {
		t.Fatal(err)
	}
	if err := creditpurchase.New(f.db, f.module.deps.Credits).Grant(t.Context(), billingevent.Envelope{UserID: "owner", Provider: "stripe", ProviderEventID: "evt_after_return"}, *reader.result.Purchase); err != nil {
		t.Fatal(err)
	}
	summary, err = f.module.deps.Credits.GetCreditSummary(t.Context(), "owner")
	if err != nil || summary.Balance != 200 {
		t.Fatalf("late webhook double-granted: %+v %v", summary, err)
	}
	ledger, err := f.module.deps.Credits.ListCreditTransactions(t.Context(), "owner", 1, 20)
	if err != nil || ledger.Total != 1 {
		t.Fatalf("late webhook ledger=%+v %v", ledger, err)
	}
}

func TestCreditCheckoutConfirmationRejectsUnpaidAndForeignPurchases(t *testing.T) {
	for _, tc := range []struct {
		name, owner, state, payment string
		role, key                   string
		status                      int
	}{
		{"unpaid", "owner", "complete", "unpaid", "user", "key", 200},
		{"foreign-provider-owner", "other", "complete", "paid", "user", "key", 404},
		{"expired", "owner", "expired", "unpaid", "user", "key", 200},
		{"no-key", "owner", "complete", "paid", "user", "", 400},
		{"signed-out", "owner", "complete", "paid", "", "key", 401},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			if err := f.db.Create(&checkoutlog.Record{UserID: "owner", Provider: "stripe", SessionID: "cs_check", ProductType: "credits"}).Error; err != nil {
				t.Fatal(err)
			}
			reader := &checkoutReader{result: &port.CreditsCheckout{SessionID: "cs_check", UserID: tc.owner, State: tc.state, PaymentStatus: tc.payment, Purchase: &billingevent.CreditsPurchased{CheckoutSessionID: "cs_check", Quantity: 1, CreditsPerUnit: 100, TotalCredits: 100}}}
			f.module.deps.Modules = &platform.Modules{Billing: &billing.Module{Provider: reader}}
			w := request(f, "POST", "/billing/credit-checkouts/cs_check/confirmation", "", tc.role, tc.key)
			if w.Code != tc.status {
				t.Fatalf("confirmation=%d %s", w.Code, w.Body)
			}
			summary, err := f.module.deps.Credits.GetCreditSummary(t.Context(), "owner")
			if err != nil || summary.Balance != 0 {
				t.Fatalf("unverified grant: balance=%d error=%v", summary.Balance, err)
			}
		})
	}
	f := newFixture(t)
	reader := &checkoutReader{}
	f.module.deps.Modules = &platform.Modules{Billing: &billing.Module{Provider: reader}}
	if err := f.db.Create(&checkoutlog.Record{UserID: "other", Provider: "stripe", SessionID: "cs_foreign", ProductType: "credits"}).Error; err != nil {
		t.Fatal(err)
	}
	w := request(f, "POST", "/billing/credit-checkouts/cs_foreign/confirmation", "", "user", "key")
	if w.Code != 404 || reader.calls != 0 {
		t.Fatalf("ownership was checked after provider lookup: %d calls=%d", w.Code, reader.calls)
	}
}

func TestCreditCheckoutResultRequiresThePurchasedCreditsLedger(t *testing.T) {
	f := newFixture(t)
	record := checkoutlog.Record{UserID: "owner", Provider: "stripe", SessionID: "cs_paid", ProductType: "credits", Quantity: 1}
	if err := f.db.Create(&record).Error; err != nil {
		t.Fatal(err)
	}
	check := func(state string, credits float64) {
		t.Helper()
		w := request(f, "GET", "/billing/credit-checkouts/cs_paid", "", "user", "")
		if w.Code != 200 || data(t, w)["state"] != state || data(t, w)["credits_granted"] != credits {
			t.Fatalf("result: %d %s", w.Code, w.Body)
		}
	}
	check("pending", 0)
	event := billingdomain.BillingEvent{UserID: "owner", Provider: "stripe", ProviderEventID: "evt_paid", EventType: "checkout.session.completed", Processed: true, Payload: json.RawMessage(`{"data":{"object":{"id":"cs_paid","payment_status":"paid","metadata":{"product_type":"credits"}}}}`)}
	if err := f.db.Create(&event).Error; err != nil {
		t.Fatal(err)
	}
	// Neither a processed payment nor an unrelated credit grant proves fulfillment.
	check("processing", 0)
	if err := f.module.deps.Credits.GrantCredits(t.Context(), "owner", "signup", "owner", 10); err != nil {
		t.Fatal(err)
	}
	check("processing", 0)
	if err := f.module.deps.Credits.GrantCredits(t.Context(), "owner", "stripe", "evt_paid", 500); err != nil {
		t.Fatal(err)
	}
	check("fulfilled", 500)
	// Spending the credits later must not turn a fulfilled purchase back into pending.
	if _, err := f.module.deps.Credits.ConsumeCredits(t.Context(), user.CreditConsumption{UserID: "owner", Source: "test", SourceID: "use", Reason: "test usage", Amount: 510}); err != nil {
		t.Fatal(err)
	}
	check("fulfilled", 500)
}

func TestCreditCheckoutResultChecksOwnershipAndAuthentication(t *testing.T) {
	f := newFixture(t)
	for _, row := range []checkoutlog.Record{
		{UserID: "other", Provider: "stripe", SessionID: "cs_other", ProductType: "credits"},
		{UserID: "owner", Provider: "stripe", SessionID: "cs_subscription", ProductType: "subscription"},
	} {
		if err := f.db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		id, role string
		status   int
	}{
		{"cs_other", "user", 404}, {"cs_missing", "user", 404}, {"cs_subscription", "user", 404}, {"cs_other", "", 401}, {"invalid", "user", 400},
	} {
		w := request(f, "GET", "/billing/credit-checkouts/"+tc.id, "", tc.role, "")
		if w.Code != tc.status {
			t.Fatalf("%s role=%s: %d %s", tc.id, tc.role, w.Code, w.Body)
		}
	}
}

func TestCreditCheckoutResultHandlesAsyncPaymentAndExpiry(t *testing.T) {
	for _, tc := range []struct{ eventType, payment, want string }{
		{"checkout.session.completed", "unpaid", "pending"},
		{"checkout.session.async_payment_succeeded", "", "processing"},
		{"checkout.session.async_payment_failed", "unpaid", "failed"},
		{"checkout.session.expired", "unpaid", "expired"},
	} {
		t.Run(tc.eventType, func(t *testing.T) {
			f := newFixture(t)
			if err := f.db.Create(&checkoutlog.Record{UserID: "owner", Provider: "stripe", SessionID: "cs_async", ProductType: "credits"}).Error; err != nil {
				t.Fatal(err)
			}
			payload, _ := json.Marshal(map[string]any{"data": map[string]any{"object": map[string]any{"id": "cs_async", "payment_status": tc.payment}}})
			if err := f.db.Create(&billingdomain.BillingEvent{UserID: "owner", Provider: "stripe", ProviderEventID: "evt_async", EventType: tc.eventType, Payload: payload}).Error; err != nil {
				t.Fatal(err)
			}
			w := request(f, "GET", "/billing/credit-checkouts/cs_async", "", "user", "")
			if w.Code != 200 || data(t, w)["state"] != tc.want {
				t.Fatalf("result: %d %s", w.Code, w.Body)
			}
		})
	}
}
