package stripe

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/brizenchi/go-modules/modules/billing/domain"
)

func TestExpireCheckoutSession_UsesAtomicExpirationAndStableIdempotency(t *testing.T) {
	var key string
	calls := 0
	p := stripeMock(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodPost || r.URL.Path != "/v1/checkout/sessions/cs_abandoned/expire" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		got := r.Header.Get("Idempotency-Key")
		if got == "" || (key != "" && key != got) {
			t.Errorf("unstable expiration key")
		}
		key = got
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"cs_abandoned","object":"checkout.session","status":"expired","payment_status":"unpaid"}`))
	})
	for range 2 {
		got, err := p.ExpireCheckoutSession(context.Background(), "cs_abandoned")
		if err != nil || got == nil || got.State != domain.CheckoutSessionExpired || got.PaymentStatus != "unpaid" {
			t.Fatalf("snapshot=%+v error=%v", got, err)
		}
	}
	if calls != 2 {
		t.Fatalf("calls=%d", calls)
	}
}

func TestExpireCheckoutSession_RechecksAfterConcurrentProviderChange(t *testing.T) {
	for _, state := range []string{"expired", "complete", "open"} {
		t.Run(state, func(t *testing.T) {
			calls := 0
			p := stripeMock(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.Header().Set("Content-Type", "application/json")
				if r.Method == http.MethodPost {
					w.WriteHeader(http.StatusBadRequest)
					_, _ = w.Write([]byte(`{"error":{"type":"invalid_request_error","message":"session is not open"}}`))
					return
				}
				if r.Method != http.MethodGet || r.URL.Path != "/v1/checkout/sessions/cs_race" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				_, _ = fmt.Fprintf(w, `{"id":"cs_race","object":"checkout.session","status":%q,"payment_status":"unpaid"}`, state)
			})
			got, err := p.ExpireCheckoutSession(context.Background(), "cs_race")
			if state == "open" {
				if err == nil {
					t.Fatal("failed expiration accepted")
				}
			} else if err != nil || got == nil || string(got.State) != state {
				t.Fatalf("snapshot=%+v error=%v", got, err)
			}
			if calls != 2 {
				t.Fatalf("requests=%d, want expire then authoritative read", calls)
			}
		})
	}
}

func TestExpireCheckoutSession_RejectsMalformedResponse(t *testing.T) {
	for _, body := range []string{`{"id":"cs_other","status":"expired","payment_status":"unpaid"}`, `{"id":"cs_expected","status":"unknown"}`, `null`} {
		t.Run(body, func(t *testing.T) {
			p := stripeMock(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(body))
			})
			if _, err := p.ExpireCheckoutSession(context.Background(), "cs_expected"); err == nil {
				t.Fatal("accepted malformed expiration response")
			}
		})
	}
}

func TestExpireCheckoutSession_RespectsCancellation(t *testing.T) {
	p := stripeMock(t, func(w http.ResponseWriter, r *http.Request) { t.Error("canceled request reached provider") })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := p.ExpireCheckoutSession(ctx, "cs_abandoned"); err == nil {
		t.Fatal("canceled expiration succeeded")
	}
}
