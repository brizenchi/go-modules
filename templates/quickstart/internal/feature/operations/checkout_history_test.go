package operations

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	billingdomain "github.com/brizenchi/go-modules/modules/billing/domain"
	"github.com/brizenchi/quickstart-template/internal/checkoutlog"
)

func TestAdminCheckoutHistoryIncludesUnpaidIntentsAndDeduplicatesPaidInvoices(t *testing.T) {
	f := newFixture(t)
	mode := false
	amount := int64(1900)
	created := time.Date(2026, 10, 1, 1, 0, 0, 0, time.UTC)
	records := []checkoutlog.Record{
		{UserID: "owner", Provider: "stripe", SessionID: "cs_open", ProductType: "subscription", Plan: "pro", BillingInterval: "monthly", Quantity: 1, AmountTotal: &amount, Currency: "usd", Livemode: &mode, CreatedAt: created},
		{UserID: "owner", Provider: "stripe", SessionID: "cs_paid", ProductType: "subscription", Plan: "premium", BillingInterval: "yearly", Quantity: 1, AmountTotal: &amount, Currency: "usd", Livemode: &mode, CreatedAt: created},
		{UserID: "other", Provider: "stripe", SessionID: "cs_other", ProductType: "credits", Quantity: 1, Livemode: &mode, CreatedAt: created},
	}
	if err := f.db.Create(&records).Error; err != nil {
		t.Fatal(err)
	}
	for _, row := range []billingdomain.BillingEvent{
		{Provider: "stripe", UserID: "owner", ProviderEventID: "evt_session_paid", EventType: "checkout.session.completed", Payload: json.RawMessage(`{"livemode":false,"data":{"object":{"id":"cs_paid","invoice":"in_paid","payment_status":"paid","amount_total":1900,"currency":"usd"}}}`)},
		{Provider: "stripe", UserID: "owner", ProviderEventID: "evt_invoice_paid", EventType: "invoice.paid", Payload: json.RawMessage(`{"livemode":false,"data":{"object":{"id":"in_paid","amount_paid":1900,"currency":"usd"}}}`)},
	} {
		if err := f.db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	w := request(f, "GET", "/admin/orders?user_id=owner&mode=test", "", "admin", "")
	if w.Code != 200 || data(t, w)["total"] != float64(2) || strings.Contains(w.Body.String(), "cs_other") {
		t.Fatalf("history: %s", w.Body)
	}
	for _, raw := range data(t, w)["items"].([]any) {
		row := raw.(map[string]any)
		switch row["checkout_session_id"] {
		case "cs_open":
			if row["status"] != "unpaid" || row["amount"] != nil || row["amount_due"] != float64(1900) || row["plan"] != "pro" {
				t.Fatalf("intent mistaken for payment: %+v", row)
			}
		case "cs_paid":
			if row["status"] != "paid" || row["amount"] != float64(1900) || row["plan"] != "premium" {
				t.Fatalf("paid invoice correlation: %+v", row)
			}
		default:
			t.Fatalf("missing checkout correlation: %+v", row)
		}
	}
	w = request(f, "GET", "/billing/purchases?user_id=other", "", "user", "")
	if w.Code != 200 || data(t, w)["total"] != float64(1) || strings.Contains(w.Body.String(), "cs_open") || strings.Contains(w.Body.String(), "cs_other") {
		t.Fatalf("customer history changed: %s", w.Body)
	}
	w = request(f, "GET", "/admin/orders?user_id=owner&status=unpaid", "", "admin", "")
	if data(t, w)["total"] != float64(1) {
		t.Fatalf("unpaid filter: %s", w.Body)
	}
	verified, err := loadOrderSummaries(t.Context(), f.db)
	if err != nil || len(verified) != 1 {
		t.Fatalf("unpaid intent entered the paid projection: %d %v", len(verified), err)
	}
	for _, session := range []string{"cs_open", "cs_paid"} {
		event := billingdomain.BillingEvent{Provider: "stripe", UserID: "owner", ProviderEventID: "evt_expired_" + session, EventType: "checkout.session.expired", Payload: json.RawMessage(`{"data":{"object":{"id":"` + session + `"}}}`)}
		if err := f.db.Create(&event).Error; err != nil {
			t.Fatal(err)
		}
	}
	w = request(f, "GET", "/admin/orders?user_id=owner&status=expired", "", "admin", "")
	if data(t, w)["total"] != float64(1) || strings.Contains(w.Body.String(), "cs_paid") {
		t.Fatalf("expiry overrode a confirmed payment: %s", w.Body)
	}
}
