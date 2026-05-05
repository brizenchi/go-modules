package stripe

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/brizenchi/go-modules/modules/billing/domain"
	"github.com/brizenchi/go-modules/modules/billing/event"
)

const testWebhookSecret = "whsec_test_secret_for_unit_tests_0123456789"

// signTestPayload computes a Stripe-compatible signature header.
// Format: t=<unix>,v1=<hex(hmac_sha256(t.payload, secret))>
func signTestPayload(t *testing.T, payload []byte, secret string) string {
	t.Helper()
	ts := time.Now().Unix()
	mac := hmac.New(sha256.New, []byte(secret))
	fmt.Fprintf(mac, "%d.%s", ts, payload)
	sig := hex.EncodeToString(mac.Sum(nil))
	return fmt.Sprintf("t=%d,v1=%s", ts, sig)
}

func newWebhookTestProvider() *Provider {
	cfg := newTestConfig()
	cfg.WebhookSecret = testWebhookSecret
	return NewProvider(cfg)
}

func TestVerifyAndParseWebhook_RejectsBadSignature(t *testing.T) {
	p := newWebhookTestProvider()
	_, err := p.VerifyAndParseWebhook([]byte(`{"id":"evt_1","type":"x"}`), "t=1,v1=deadbeef")
	if err == nil {
		t.Fatal("expected error for bad signature")
	}
	if !errors.Is(err, domain.ErrSignatureInvalid) {
		t.Errorf("expected ErrSignatureInvalid, got %v", err)
	}
}

func TestVerifyAndParseWebhook_DisabledProvider(t *testing.T) {
	cfg := newTestConfig()
	cfg.Enabled = false
	cfg.WebhookSecret = testWebhookSecret
	p := NewProvider(cfg)
	_, err := p.VerifyAndParseWebhook([]byte(`{}`), "t=1,v1=x")
	if !errors.Is(err, domain.ErrProviderDisabled) {
		t.Errorf("expected ErrProviderDisabled, got %v", err)
	}
}

func TestVerifyAndParseWebhook_SubscriptionUpdated_Cancelling(t *testing.T) {
	p := newWebhookTestProvider()
	payload := []byte(`{
		"id": "evt_test_sub_cancel",
		"object": "event",
		"type": "customer.subscription.updated",
		"created": 1700000000,
		"data": {
			"object": {
				"id": "sub_123",
				"customer": "cus_123",
				"status": "active",
				"cancel_at_period_end": true,
				"current_period_start": 1699000000,
				"current_period_end": 1701600000,
				"items": {
					"data": [
						{"price": {"id": "price_starter_m", "product": "prod_starter"}}
					]
				}
			}
		}
	}`)
	sig := signTestPayload(t, payload, testWebhookSecret)
	res, err := p.VerifyAndParseWebhook(payload, sig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ProviderEventID != "evt_test_sub_cancel" {
		t.Errorf("event id = %q, want evt_test_sub_cancel", res.ProviderEventID)
	}
	if len(res.Envelopes) != 1 {
		t.Fatalf("expected 1 envelope, got %d", len(res.Envelopes))
	}
	env := res.Envelopes[0]
	if env.Kind != event.KindSubscriptionCanceling {
		t.Errorf("kind = %s, want %s", env.Kind, event.KindSubscriptionCanceling)
	}
	cancelEv, ok := env.Payload.(event.SubscriptionCanceling)
	if !ok {
		t.Fatalf("payload type = %T, want SubscriptionCanceling", env.Payload)
	}
	if cancelEv.Snapshot.Plan != domain.PlanStarter {
		t.Errorf("plan = %s, want starter", cancelEv.Snapshot.Plan)
	}
	if cancelEv.Mode != domain.CancelAtPeriodEnd {
		t.Errorf("mode = %s, want end_of_period", cancelEv.Mode)
	}
	if cancelEv.Snapshot.CancelEffectiveAt == nil {
		t.Error("expected CancelEffectiveAt to be set (period end)")
	}
}

func TestVerifyAndParseWebhook_InvoicePaid_RenewalEmitsEvent(t *testing.T) {
	p := newWebhookTestProvider()
	payload := []byte(`{
		"id": "evt_invoice_paid",
		"type": "invoice.payment_succeeded",
		"created": 1700000000,
		"data": {
			"object": {
				"id": "in_123",
				"customer": "cus_x",
				"subscription": "sub_x",
				"billing_reason": "subscription_cycle",
				"lines": {
					"data": [
						{
							"price": {"id": "price_pro_m", "product": "prod_pro"},
							"period": {"start": 1699000000, "end": 1701600000}
						}
					]
				}
			}
		}
	}`)
	sig := signTestPayload(t, payload, testWebhookSecret)
	res, err := p.VerifyAndParseWebhook(payload, sig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Envelopes) != 1 {
		t.Fatalf("expected 1 envelope, got %d", len(res.Envelopes))
	}
	if res.Envelopes[0].Kind != event.KindSubscriptionRenewed {
		t.Errorf("kind = %s, want %s", res.Envelopes[0].Kind, event.KindSubscriptionRenewed)
	}
	renewed := res.Envelopes[0].Payload.(event.SubscriptionRenewed)
	if renewed.Snapshot.Plan != domain.PlanPro {
		t.Errorf("plan = %s, want pro", renewed.Snapshot.Plan)
	}
}

func TestVerifyAndParseWebhook_InvoicePaid_FirstInvoiceSkipped(t *testing.T) {
	// First invoice (subscription_create) must NOT emit a renewal event;
	// checkout.session.completed handles activation.
	p := newWebhookTestProvider()
	payload := []byte(`{
		"id": "evt_first_invoice",
		"type": "invoice.payment_succeeded",
		"created": 1700000000,
		"data": {
			"object": {
				"id": "in_first",
				"customer": "cus_x",
				"subscription": "sub_x",
				"billing_reason": "subscription_create",
				"lines": {"data": [{"price": {"id": "price_pro_m"}}]}
			}
		}
	}`)
	sig := signTestPayload(t, payload, testWebhookSecret)
	res, err := p.VerifyAndParseWebhook(payload, sig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Envelopes) != 0 {
		t.Errorf("expected no envelopes for first invoice, got %d", len(res.Envelopes))
	}
}

func TestVerifyAndParseWebhook_CheckoutCompleted_Subscription(t *testing.T) {
	p := newWebhookTestProvider()
	payload := []byte(`{
		"id": "evt_checkout_sub",
		"type": "checkout.session.completed",
		"created": 1700000000,
		"data": {
			"object": {
				"id": "cs_123",
				"mode": "subscription",
				"customer": "cus_x",
				"subscription": "sub_x",
				"client_reference_id": "user_42",
				"metadata": {
					"user_id": "user_42",
					"email": "u@example.com",
					"product_type": "subscription",
					"plan": "premium",
					"interval": "yearly",
					"price_id": "price_premium_y"
				}
			}
		}
	}`)
	sig := signTestPayload(t, payload, testWebhookSecret)
	res, err := p.VerifyAndParseWebhook(payload, sig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Envelopes) != 1 {
		t.Fatalf("expected 1 envelope, got %d", len(res.Envelopes))
	}
	env := res.Envelopes[0]
	if env.Kind != event.KindSubscriptionActivated {
		t.Errorf("kind = %s, want %s", env.Kind, event.KindSubscriptionActivated)
	}
	if res.UserHint.UserID != "user_42" {
		t.Errorf("user hint = %q, want user_42", res.UserHint.UserID)
	}
	activated := env.Payload.(event.SubscriptionActivated)
	if activated.Snapshot.Plan != domain.PlanPremium {
		t.Errorf("plan = %s, want premium", activated.Snapshot.Plan)
	}
	if activated.Snapshot.Interval != domain.IntervalYearly {
		t.Errorf("interval = %s, want yearly", activated.Snapshot.Interval)
	}
}

func TestVerifyAndParseWebhook_CheckoutCompleted_Lifetime(t *testing.T) {
	p := newWebhookTestProvider()
	payload := []byte(`{
		"id": "evt_checkout_lifetime",
		"type": "checkout.session.completed",
		"created": 1700000000,
		"data": {
			"object": {
				"id": "cs_lifetime",
				"mode": "payment",
				"customer": "cus_life",
				"client_reference_id": "user_life",
				"metadata": {
					"user_id": "user_life",
					"email": "life@example.com",
					"product_type": "lifetime",
					"plan": "lifetime",
					"price_id": "price_lifetime"
				}
			}
		}
	}`)
	sig := signTestPayload(t, payload, testWebhookSecret)
	res, err := p.VerifyAndParseWebhook(payload, sig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Envelopes) != 1 {
		t.Fatalf("expected 1 envelope, got %d", len(res.Envelopes))
	}
	if res.Envelopes[0].Kind != event.KindSubscriptionActivated {
		t.Fatalf("kind = %s", res.Envelopes[0].Kind)
	}
	activated := res.Envelopes[0].Payload.(event.SubscriptionActivated)
	if activated.Snapshot.Plan != domain.PlanLifetime {
		t.Fatalf("plan = %s", activated.Snapshot.Plan)
	}
	if activated.Snapshot.ProductType != domain.ProductLifetime {
		t.Fatalf("product_type = %s", activated.Snapshot.ProductType)
	}
	if activated.Snapshot.ProviderPriceID != "price_lifetime" {
		t.Fatalf("price_id = %s", activated.Snapshot.ProviderPriceID)
	}
}

func TestVerifyAndParseWebhook_PaymentFailed(t *testing.T) {
	p := newWebhookTestProvider()
	payload := []byte(`{
		"id": "evt_payfail",
		"type": "invoice.payment_failed",
		"created": 1700000000,
		"data": {
			"object": {
				"id": "in_x",
				"customer": "cus_x",
				"subscription": "sub_x"
			}
		}
	}`)
	sig := signTestPayload(t, payload, testWebhookSecret)
	res, err := p.VerifyAndParseWebhook(payload, sig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Envelopes) != 1 || res.Envelopes[0].Kind != event.KindPaymentFailed {
		t.Errorf("expected 1 PaymentFailed, got %v", res.Envelopes)
	}
}

func TestVerifyAndParseWebhook_SubscriptionDeleted(t *testing.T) {
	p := newWebhookTestProvider()
	payload := []byte(`{
		"id": "evt_subdel",
		"type": "customer.subscription.deleted",
		"created": 1700000000,
		"data": {
			"object": {"id": "sub_x", "customer": "cus_x"}
		}
	}`)
	sig := signTestPayload(t, payload, testWebhookSecret)
	res, err := p.VerifyAndParseWebhook(payload, sig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Envelopes) != 1 || res.Envelopes[0].Kind != event.KindSubscriptionCanceled {
		t.Errorf("expected 1 SubscriptionCanceled, got %v", res.Envelopes)
	}
}

func TestVerifyAndParseWebhook_UnhandledEventReturnsZeroEnvelopes(t *testing.T) {
	p := newWebhookTestProvider()
	payload := []byte(`{
		"id": "evt_other",
		"type": "customer.created",
		"created": 1700000000,
		"data": {"object": {"id": "cus_x"}}
	}`)
	sig := signTestPayload(t, payload, testWebhookSecret)
	res, err := p.VerifyAndParseWebhook(payload, sig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Envelopes) != 0 {
		t.Errorf("expected 0 envelopes for unhandled event, got %d", len(res.Envelopes))
	}
	if res.ProviderEventID != "evt_other" {
		t.Errorf("event id = %q, want evt_other", res.ProviderEventID)
	}
}
