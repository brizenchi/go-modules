package stripe

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/brizenchi/go-modules/modules/billing/domain"
)

func TestMonthlyToYearlyPreviewUsesNewPeriodEndAndUnusedCredit(t *testing.T) {
	const renewal int64 = 1823817600
	p := stripeMock(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/subscriptions/sub_1":
			fmt.Fprint(w, `{"id":"sub_1","currency":"usd","current_period_end":1794873600,"items":{"data":[{"id":"si_1","price":{"id":"price_pro_m","recurring":{"interval":"month"}}}]}}`)
		case "/v1/invoices/upcoming":
			if r.URL.Query().Get("subscription_billing_cycle_anchor") != "now" || r.URL.Query().Get("subscription_proration_behavior") != "always_invoice" {
				t.Errorf("incorrect annual preview: %v", r.URL.Query())
			}
			fmt.Fprintf(w, `{"amount_due":13200,"currency":"usd","next_payment_attempt":1792281600,"lines":{"has_more":false,"data":[{"amount":-700,"proration":true,"subscription_item":"si_1","price":{"id":"price_pro_m"}},{"amount":-100,"proration":false,"description":"unrelated adjustment"},{"amount":14000,"proration":false,"subscription_item":"si_1","price":{"id":%q},"period":{"end":%d}}]}}`, "price_pro_y", renewal)
		default:
			t.Errorf("unexpected request %s", r.URL.Path)
			w.WriteHeader(404)
		}
	})
	p.cfg.SubscriptionPrices[domain.PlanPro][domain.IntervalYearly] = "price_pro_y"
	preview, err := p.PreviewSubscriptionChange(context.Background(), "cus_1", "sub_1", domain.SubscriptionPreviewInput{Plan: domain.PlanPro, Interval: domain.IntervalYearly, Mode: domain.ChangeModeImmediateResetCycle})
	if err != nil {
		t.Fatal(err)
	}
	if preview.NextBillingAt == nil || !preview.NextBillingAt.Equal(time.Unix(renewal, 0)) {
		t.Fatalf("annual renewal must use new period, got %+v", preview.NextBillingAt)
	}
	if preview.UnusedCreditAmount == nil || *preview.UnusedCreditAmount != 700 {
		t.Fatalf("unused credit: %+v", preview.UnusedCreditAmount)
	}
	if preview.AmountDueNow != 132 || !preview.ImmediateCharge {
		t.Fatalf("preview: %+v", preview)
	}
}

func TestMonthlyToYearlyUpdatesExistingSubscriptionAndWaitsForPayment(t *testing.T) {
	for _, pending := range []bool{false, true} {
		t.Run(fmt.Sprint("pending=", pending), func(t *testing.T) {
			writes := 0
			p := stripeMock(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path != "/v1/subscriptions/sub_1" {
					t.Errorf("must update existing subscription: %s", r.URL.Path)
					w.WriteHeader(404)
					return
				}
				if r.Method == http.MethodGet {
					fmt.Fprint(w, `{"id":"sub_1","customer":"cus_1","status":"active","items":{"data":[{"id":"si_1","price":{"id":"price_pro_m","recurring":{"interval":"month"}}}]}}`)
					return
				}
				writes++
				form := readForm(t, r)
				if form.Get("billing_cycle_anchor") != "now" || form.Get("proration_behavior") != "always_invoice" || form.Get("payment_behavior") != "pending_if_incomplete" || form.Get("items[0][id]") != "si_1" || form.Get("items[0][price]") != "price_pro_y" {
					t.Errorf("unsafe annual switch: %v", form)
				}
				if pending {
					fmt.Fprint(w, `{"id":"sub_1","customer":"cus_1","status":"active","pending_update":{"expires_at":1900000000},"latest_invoice":{"amount_due":13300,"currency":"usd"},"items":{"data":[{"id":"si_1","price":{"id":"price_pro_m","recurring":{"interval":"month"}}}]}}`)
				} else {
					fmt.Fprint(w, `{"id":"sub_1","customer":"cus_1","status":"active","items":{"data":[{"id":"si_1","price":{"id":"price_pro_y","recurring":{"interval":"year"}}}]}}`)
				}
			})
			p.cfg.SubscriptionPrices[domain.PlanPro][domain.IntervalYearly] = "price_pro_y"
			snapshot, err := p.ChangeSubscription(context.Background(), "sub_1", domain.SubscriptionChangeInput{Plan: domain.PlanPro, Interval: domain.IntervalYearly, Mode: domain.ChangeModeImmediateResetCycle})
			if err != nil {
				t.Fatal(err)
			}
			if writes != 1 || (snapshot.PendingPayment != nil) != pending {
				t.Fatalf("payment outcome: %+v writes=%d", snapshot, writes)
			}
			want := domain.IntervalYearly
			if pending {
				want = domain.IntervalMonthly
			}
			if snapshot.Interval != want {
				t.Fatalf("interval %s, want %s", snapshot.Interval, want)
			}
		})
	}
}
