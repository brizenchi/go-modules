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

func TestTrialPlanChangesPreserveTrial(t *testing.T) {
	// The original three-day trial is already partly used: switching must not
	// replace its remaining time with a fresh trial or end it immediately.
	trialEnd := time.Now().Add(36 * time.Hour).Unix()
	type selection struct {
		plan           domain.PlanType
		interval       domain.BillingInterval
		stripeInterval string
	}
	var selections []selection
	for _, plan := range []domain.PlanType{domain.PlanStarter, domain.PlanPro, domain.PlanPremium} {
		selections = append(selections, selection{plan, domain.IntervalMonthly, "month"}, selection{plan, domain.IntervalYearly, "year"})
	}
	for _, from := range selections {
		for _, to := range selections {
			for _, action := range []string{"preview", "change"} {
				t.Run(string(from.plan)+"/"+string(from.interval)+"/to/"+string(to.plan)+"/"+string(to.interval)+"/"+action, func(t *testing.T) {
					writes := 0
					var currentPrice, targetPrice string
					p := stripeMock(t, func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						if r.URL.Path != "/v1/subscriptions/sub_trial" {
							t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
							w.WriteHeader(http.StatusBadRequest)
							return
						}
						price, interval := currentPrice, from.stripeInterval
						if r.Method == http.MethodPost {
							writes++
							form := readForm(t, r)
							if form.Get("items[0][id]") != "si_trial" || form.Get("items[0][price]") != targetPrice {
								t.Errorf("must replace price on existing item: %v", form)
							}
							if form.Get("billing_cycle_anchor") != "" || form.Get("trial_end") != fmt.Sprint(trialEnd) || form.Get("trial_period_days") != "" {
								t.Errorf("must preserve original trial end without resetting cycle: %v", form)
							}
							if form.Get("proration_behavior") != "none" || form.Get("payment_behavior") != "" {
								t.Errorf("trial change must not request immediate or pending payment: %v", form)
							}
							price, interval = targetPrice, to.stripeInterval
						}
						fmt.Fprintf(w, `{"id":"sub_trial","customer":"cus_trial","status":"trialing","currency":"usd","trial_end":%d,"current_period_end":%d,"items":{"data":[{"id":"si_trial","price":{"id":%q,"recurring":{"interval":%q}}}]}}`, trialEnd, trialEnd, price, interval)
					})
					p.cfg.SubscriptionPrices[domain.PlanPro][domain.IntervalYearly] = "price_pro_y"
					currentPrice = p.cfg.PriceFor(from.plan, from.interval)
					targetPrice = p.cfg.PriceFor(to.plan, to.interval)
					if action == "preview" {
						preview, err := p.PreviewSubscriptionChange(t.Context(), "cus_trial", "sub_trial", domain.SubscriptionPreviewInput{
							Plan: to.plan, Interval: to.interval, Mode: domain.ChangeModeImmediateProrated,
						})
						if err != nil {
							t.Fatal(err)
						}
						if preview.AmountDueNow != 0 || preview.ImmediateCharge || preview.EffectiveAtPeriodEnd || writes != 0 {
							t.Fatalf("preview must be read-only with nothing due now: %+v writes=%d", preview, writes)
						}
						if preview.NextBillingAt == nil || preview.NextBillingAt.Unix() != trialEnd || preview.CurrentPeriodEnd == nil || preview.CurrentPeriodEnd.Unix() != trialEnd {
							t.Fatalf("first paid billing must remain at trial end: %+v", preview)
						}
						if preview.TargetPlan != to.plan || preview.TargetInterval != to.interval || preview.Mode != domain.ChangeModeImmediateProrated || preview.Currency != "usd" {
							t.Fatalf("incorrect preview selection: %+v", preview)
						}
						return
					}
					snapshot, err := p.ChangeSubscription(t.Context(), "sub_trial", domain.SubscriptionChangeInput{
						Plan: to.plan, Interval: to.interval, Mode: domain.ChangeModeImmediateProrated,
					})
					if err != nil {
						t.Fatal(err)
					}
					if writes != 1 || snapshot.ProviderSubscriptionID != "sub_trial" || snapshot.Status != domain.StatusTrialing || snapshot.Interval != to.interval || snapshot.Plan != to.plan || snapshot.PendingPayment != nil {
						t.Fatalf("must retain trial on existing subscription: %+v writes=%d", snapshot, writes)
					}
					if snapshot.PeriodEnd == nil || snapshot.PeriodEnd.Unix() != trialEnd {
						t.Fatalf("trial end changed: %+v", snapshot.PeriodEnd)
					}
				})
			}
		}
	}
}
