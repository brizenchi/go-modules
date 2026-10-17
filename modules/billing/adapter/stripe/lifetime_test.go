package stripe

import (
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/brizenchi/go-modules/modules/billing/domain"
)

func TestLifetimeRetirementRemovesScheduleAndPendingUpgradeWithoutProration(t *testing.T) {
	var calls []string
	p := stripeMock(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		call := r.Method + " " + r.URL.Path
		calls = append(calls, call)
		switch call {
		case "GET /v1/subscriptions/sub_old":
			_, _ = w.Write([]byte(`{"id":"sub_old","customer":"cus_owner","status":"active","pending_update":{"expires_at":1900000000},"latest_invoice":"in_pending","schedule":"sched_old"}`))
		case "POST /v1/invoices/in_pending/void":
			_, _ = w.Write([]byte(`{"id":"in_pending","status":"void"}`))
		case "POST /v1/subscription_schedules/sched_old/release":
			_, _ = w.Write([]byte(`{"id":"sched_old","status":"released"}`))
		case "DELETE /v1/subscriptions/sub_old":
			form := readForm(t, r)
			for k, v := range r.URL.Query() {
				form[k] = v
			}
			if form.Get("invoice_now") != "false" || form.Get("prorate") != "false" {
				t.Errorf("unexpected retirement fees: %v", form)
			}
			_, _ = w.Write([]byte(`{"id":"sub_old","status":"canceled"}`))
		default:
			t.Errorf("unexpected request %s", call)
			w.WriteHeader(500)
		}
	})
	if err := p.RetireSubscriptionForLifetime(t.Context(), "sub_old", "cus_owner"); err != nil {
		t.Fatal(err)
	}
	want := []string{"GET /v1/subscriptions/sub_old", "POST /v1/invoices/in_pending/void", "POST /v1/subscription_schedules/sched_old/release", "DELETE /v1/subscriptions/sub_old"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls=%v", calls)
	}
}

func TestLifetimeRetirementIsIdempotentAndChecksCustomerOwnership(t *testing.T) {
	for _, customer := range []string{"cus_owner", "cus_other"} {
		t.Run(customer, func(t *testing.T) {
			p := stripeMock(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Fatal("terminal subscription must not be mutated")
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"id":"sub_old","customer":"cus_owner","status":"canceled"}`))
			})
			err := p.RetireSubscriptionForLifetime(t.Context(), "sub_old", customer)
			if customer == "cus_other" {
				if !errors.Is(err, domain.ErrInvalidInput) {
					t.Fatalf("ownership error=%v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}
