package stripe

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/brizenchi/go-modules/modules/billing/domain"
	"github.com/brizenchi/go-modules/modules/billing/event"
	stripesdk "github.com/stripe/stripe-go/v76"
)

func TestModernSubscriptionPeriodAndFailedCancellation(t *testing.T) {
	p := &Provider{}
	data := map[string]any{"status": "past_due", "cancel_at_period_end": true, "items": map[string]any{"data": []any{map[string]any{"current_period_start": float64(1700000000), "current_period_end": float64(1700100000)}}}}
	snap := p.snapshotFromMap(data)
	if snap.PeriodEnd == nil || snap.PeriodEnd.Unix() != 1700100000 || snap.CancelEffectiveAt == nil {
		t.Fatalf("missing item period: %+v", snap)
	}
	if snap.Status != domain.StatusPastDue {
		t.Fatalf("cancellation masked payment failure: %s", snap.Status)
	}
}

func TestModernInvoiceSubscriptionHint(t *testing.T) {
	data := map[string]any{"parent": map[string]any{"subscription_details": map[string]any{"subscription": "sub_test"}}}
	if got := extractUserHint(data, "invoice.paid").ProviderSubscriptionID; got != "sub_test" {
		t.Fatalf("subscription = %q", got)
	}
}

func TestPendingPaymentAndScheduledPhase(t *testing.T) {
	p := NewProvider(newTestConfig())
	var sub stripesdk.Subscription
	err := json.Unmarshal([]byte(`{"id":"sub_test","status":"active","pending_update":{"expires_at":1700000000},"latest_invoice":{"id":"in_test","amount_due":1234,"currency":"usd","hosted_invoice_url":"https://invoice.stripe.com/i/test"},"schedule":{"id":"sched_test","current_phase":{"start_date":1,"end_date":1700100000},"phases":[{"start_date":1,"end_date":1700100000},{"start_date":1700100000,"items":[{"price":{"id":"price_unknown"}}]}]}}`), &sub)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := p.snapshotFromSubscription(&sub)
	if snapshot.PendingPayment == nil || snapshot.PendingPayment.AmountDue != 1234 || snapshot.PendingPayment.InvoiceURL == "" {
		t.Fatalf("missing pending payment: %+v", snapshot)
	}
	if snapshot.ScheduledChange == nil || snapshot.ScheduledChange.EffectiveAt.Unix() != 1700100000 {
		t.Fatalf("missing phase: %+v", snapshot)
	}
}

func TestCancelScheduledChangeReleasesWithoutCancellation(t *testing.T) {
	released := false
	p := stripeMock(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method + " " + r.URL.Path {
		case "GET /v1/subscriptions/sub_test":
			_, _ = w.Write([]byte(`{"id":"sub_test","schedule":"sub_sched_test"}`))
		case "POST /v1/subscription_schedules/sub_sched_test/release":
			released = true
			_, _ = w.Write([]byte(`{"id":"sub_sched_test","status":"released"}`))
		default:
			t.Fatalf("unexpected mutation: %s %s", r.Method, r.URL.Path)
		}
	})
	if err := p.CancelScheduledChange(context.Background(), "sub_test"); err != nil {
		t.Fatal(err)
	}
	if !released {
		t.Fatal("schedule was not released")
	}
}

func TestModernInvoiceRenewal(t *testing.T) {
	p := newWebhookTestProvider()
	payload := []byte(`{"id":"evt_modern","type":"invoice.paid","created":1700000000,"data":{"object":{"customer":"cus_test","parent":{"subscription_details":{"subscription":"sub_test"}},"billing_reason":"subscription_cycle","amount_paid":1200,"currency":"usd","lines":{"data":[{"pricing":{"price_details":{"price":"price_pro_m","product":"prod_test"}},"period":{"start":1700000000,"end":1700100000}}]}}}}`)
	result, err := p.VerifyAndParseWebhookContext(t.Context(), payload, signTestPayload(t, payload, testWebhookSecret))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Envelopes) != 1 {
		t.Fatalf("renewals: %+v", result.Envelopes)
	}
	renewal := result.Envelopes[0].Payload.(event.SubscriptionRenewed)
	if renewal.Snapshot.Plan != domain.PlanPro || renewal.Snapshot.ProviderSubscriptionID != "sub_test" || renewal.AmountPaid != 1200 {
		t.Fatalf("renewal: %+v", renewal)
	}
}

func TestScheduleFromSubscriptionUsesCompatibleCreateParameters(t *testing.T) {
	p := stripeMock(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method + " " + r.URL.Path {
		case "GET /v1/subscriptions/sub_test":
			_, _ = w.Write([]byte(`{"id":"sub_test","status":"active","items":{"data":[{"id":"si_test","quantity":1,"price":{"id":"price_pro_m"}}]}}`))
		case "POST /v1/subscription_schedules":
			form := readForm(t, r)
			if form.Get("from_subscription") != "sub_test" || form.Has("end_behavior") {
				w.WriteHeader(400)
				_, _ = w.Write([]byte(`{"error":{"message":"from_subscription excludes end_behavior","type":"invalid_request_error"}}`))
				return
			}
			_, _ = w.Write([]byte(`{"id":"sub_sched_test"}`))
		case "GET /v1/subscription_schedules/sub_sched_test":
			_, _ = w.Write([]byte(`{"id":"sub_sched_test","subscription":"sub_test","current_phase":{"start_date":1700000000,"end_date":1700100000}}`))
		case "POST /v1/subscription_schedules/sub_sched_test":
			form := readForm(t, r)
			if form.Get("end_behavior") != "release" {
				t.Error("update must set release behavior")
			}
			_, _ = w.Write([]byte(`{"id":"sub_sched_test"}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	result, err := p.ScheduleSubscriptionChange(t.Context(), "sub_test", domain.SubscriptionChangeInput{Plan: domain.PlanStarter, Interval: domain.IntervalMonthly, Mode: domain.ChangeModePeriodEnd})
	if err != nil {
		t.Fatal(err)
	}
	if result.ScheduledChange == nil || result.ScheduledChange.Plan != domain.PlanStarter {
		t.Fatalf("missing future plan: %+v", result)
	}
}

func TestCancellationAfterDowngradeReleasesCompletedSchedule(t *testing.T) {
	released := false
	p := stripeMock(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method + " " + r.URL.Path {
		case "GET /v1/subscriptions/sub_test":
			_, _ = w.Write([]byte(`{"id":"sub_test","status":"active","schedule":{"id":"sub_sched_test","current_phase":{"start_date":1700100000,"end_date":1700200000},"phases":[{"start_date":1700000000,"end_date":1700100000},{"start_date":1700100000,"end_date":1700200000}]}}`))
		case "POST /v1/subscription_schedules/sub_sched_test/release":
			released = true
			_, _ = w.Write([]byte(`{"id":"sub_sched_test","status":"released"}`))
		case "POST /v1/subscriptions/sub_test":
			if !released {
				w.WriteHeader(400)
				_, _ = w.Write([]byte(`{"error":{"message":"subscription managed by schedule","type":"invalid_request_error"}}`))
				return
			}
			_, _ = w.Write([]byte(`{"id":"sub_test","cancel_at_period_end":true}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	if err := p.CancelSubscription(t.Context(), "sub_test", domain.CancelAtPeriodEnd); err != nil {
		t.Fatal(err)
	}
	if !released {
		t.Fatal("completed schedule not released")
	}
}

func TestFutureScheduleCannotBeSilentlyReleased(t *testing.T) {
	p := &Provider{}
	sub := &stripesdk.Subscription{Schedule: &stripesdk.SubscriptionSchedule{ID: "sub_sched_test", CurrentPhase: &stripesdk.SubscriptionScheduleCurrentPhase{StartDate: 1, EndDate: 2}, Phases: []*stripesdk.SubscriptionSchedulePhase{{StartDate: 1}, {StartDate: 2}}}}
	if err := p.releaseCompletedSchedule(t.Context(), sub); !errors.Is(err, domain.ErrPendingSubscriptionChange) {
		t.Fatalf("future schedule: %v", err)
	}
}
