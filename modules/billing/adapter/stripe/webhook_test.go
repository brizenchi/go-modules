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
	stripesdk "github.com/stripe/stripe-go/v76"
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

func TestNormalizeStripeStatus_PreservesTerminalIncompleteExpired(t *testing.T) {
	if got := normalizeStripeStatus("incomplete_expired", false); got != domain.StatusIncompleteExpired {
		t.Fatalf("status = %q, want %q", got, domain.StatusIncompleteExpired)
	}
	if got := normalizeStripeStatus("canceled", true); got != domain.StatusCanceled {
		t.Fatalf("terminal status = %q, want %q", got, domain.StatusCanceled)
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

func TestVerifyAndParseWebhook_SubscriptionUpdated_ExplicitCancelAt(t *testing.T) {
	p := newWebhookTestProvider()
	payload := []byte(`{
		"id":"evt_cancel_at",
		"type":"customer.subscription.updated",
		"created":1700000000,
		"data":{"object":{
			"id":"sub_cancel_at",
			"customer":"cus_123",
			"status":"active",
			"cancel_at_period_end":false,
			"cancel_at":1700259200,
			"items":{"data":[{"price":{"id":"price_starter_m","product":"prod_starter"}}]}
		}}
	}`)
	res, err := p.VerifyAndParseWebhook(payload, signTestPayload(t, payload, testWebhookSecret))
	if err != nil {
		t.Fatalf("VerifyAndParseWebhook: %v", err)
	}
	if len(res.Envelopes) != 1 || res.Envelopes[0].Kind != event.KindSubscriptionCanceling {
		t.Fatalf("envelopes = %+v, want canceling", res.Envelopes)
	}
	canceling := res.Envelopes[0].Payload.(event.SubscriptionCanceling)
	if canceling.Mode != domain.CancelIn3Days || canceling.Snapshot.Status != domain.StatusCanceling || canceling.Snapshot.CancelAtPeriodEnd {
		t.Fatalf("canceling payload = %+v", canceling)
	}
	want := time.Unix(1700259200, 0).UTC()
	if canceling.EffectiveAt == nil || !canceling.EffectiveAt.Equal(want) || canceling.Snapshot.CancelEffectiveAt == nil || !canceling.Snapshot.CancelEffectiveAt.Equal(want) {
		t.Fatalf("canceling deadline = %+v, want %v", canceling, want)
	}
}

func TestVerifyAndParseWebhook_InvoicePaid_RenewalEmitsEvent(t *testing.T) {
	p := newWebhookTestProvider()
	payload := []byte(`{
		"id": "evt_invoice_paid",
		"type": "invoice.paid",
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

func TestVerifyAndParseWebhook_PositiveCycleInvoiceDoesNotMislabelEveryRenewalAsTrialConversion(t *testing.T) {
	p := newWebhookTestProvider()
	payload := []byte(`{
		"id":"evt_paid_conversion",
		"type":"invoice.paid",
		"created":1700000000,
		"data":{"object":{
			"id":"in_paid",
			"customer":"cus_x",
				"subscription":"sub_x",
				"billing_reason":"subscription_cycle",
				"amount_paid":1000,
				"currency":"usd",
			"lines":{"data":[{"price":{"id":"price_pro_m","product":"prod_pro"},"period":{"start":1699000000,"end":1701600000}}]}
		}}
	}`)
	res, err := p.VerifyAndParseWebhook(payload, signTestPayload(t, payload, testWebhookSecret))
	if err != nil {
		t.Fatalf("VerifyAndParseWebhook: %v", err)
	}
	if len(res.Envelopes) != 1 || res.Envelopes[0].Kind != event.KindSubscriptionRenewed {
		t.Fatalf("envelopes = %+v, want only renewal", res.Envelopes)
	}
	renewed := res.Envelopes[0].Payload.(event.SubscriptionRenewed)
	if renewed.AmountPaid != 1000 || renewed.Currency != "usd" {
		t.Fatalf("renewal payment fact = %+v", renewed)
	}
}

func TestVerifyAndParseWebhook_ZeroAmountCycleIsNotQualifyingPayment(t *testing.T) {
	p := newWebhookTestProvider()
	payload := []byte(`{
		"id":"evt_zero_cycle",
		"type":"invoice.paid",
		"created":1700000000,
		"data":{"object":{
			"id":"in_zero",
			"customer":"cus_x",
			"subscription":"sub_x",
			"billing_reason":"subscription_cycle",
			"amount_paid":0,
			"currency":"usd",
			"lines":{"data":[{"price":{"id":"price_pro_m","product":"prod_pro"}}]}
		}}
	}`)
	res, err := p.VerifyAndParseWebhook(payload, signTestPayload(t, payload, testWebhookSecret))
	if err != nil {
		t.Fatalf("VerifyAndParseWebhook: %v", err)
	}
	if len(res.Envelopes) != 1 || res.Envelopes[0].Kind != event.KindSubscriptionRenewed {
		t.Fatalf("envelopes = %+v, want renewal lifecycle fact", res.Envelopes)
	}
	renewed := res.Envelopes[0].Payload.(event.SubscriptionRenewed)
	if renewed.AmountPaid != 0 {
		t.Fatalf("amount_paid=%d, want zero and therefore not referral-qualified", renewed.AmountPaid)
	}
}

func TestVerifyAndParseWebhook_InvoicePaid_FirstInvoiceSkipped(t *testing.T) {
	// First invoice (subscription_create) must NOT emit a renewal event;
	// checkout.session.completed handles activation.
	p := newWebhookTestProvider()
	payload := []byte(`{
		"id": "evt_first_invoice",
		"type": "invoice.paid",
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

func TestInvoicePaid_IgnoresProrationAndIncompleteRenewalSnapshots(t *testing.T) {
	p := newWebhookTestProvider()
	mk := func(kind event.Kind, payload any) event.Envelope {
		return event.Envelope{Kind: kind, Payload: payload}
	}
	for _, data := range []map[string]any{
		{
			"subscription":   "sub_x",
			"billing_reason": "subscription_update",
			"lines":          map[string]any{"data": []any{map[string]any{"price": map[string]any{"id": "price_pro_m"}}}},
		},
		{
			"subscription":   "sub_x",
			"billing_reason": "subscription_cycle",
			"lines":          map[string]any{"data": []any{}},
		},
		{
			"subscription":   "sub_x",
			"billing_reason": "subscription_cycle",
			"lines":          map[string]any{"data": []any{map[string]any{"price": map[string]any{"id": "price_unknown"}}}},
		},
	} {
		if got := p.onInvoicePaid(data, mk); len(got) != 0 {
			t.Fatalf("unsafe invoice emitted renewal: %#v", got)
		}
	}
}

func TestSubscriptionUpdated_ReactivationRequiresClearedCancelFlag(t *testing.T) {
	p := newWebhookTestProvider()
	mk := func(kind event.Kind, payload any) event.Envelope {
		return event.Envelope{Kind: kind, Payload: payload}
	}
	data := map[string]any{
		"id":                   "sub_x",
		"customer":             "cus_x",
		"status":               "active",
		"cancel_at_period_end": false,
		"items":                map[string]any{"data": []any{map[string]any{"price": map[string]any{"id": "price_pro_m"}}}},
	}
	ordinary := p.onSubscriptionUpdated(data, map[string]any{"metadata": map[string]any{}}, mk)
	if len(ordinary) != 1 || ordinary[0].Kind != event.KindSubscriptionUpdated {
		t.Fatalf("ordinary update = %#v, want SubscriptionUpdated", ordinary)
	}
	zeroDollarTrialEnd := p.onSubscriptionUpdated(data, map[string]any{"status": "trialing"}, mk)
	if len(zeroDollarTrialEnd) != 1 || zeroDollarTrialEnd[0].Kind != event.KindSubscriptionUpdated {
		t.Fatalf("unpaid trial transition = %#v, want SubscriptionUpdated", zeroDollarTrialEnd)
	}
	reactivated := p.onSubscriptionUpdated(data, map[string]any{"cancel_at_period_end": true}, mk)
	if len(reactivated) != 1 || reactivated[0].Kind != event.KindSubscriptionReactivated {
		t.Fatalf("cleared cancellation = %#v, want SubscriptionReactivated", reactivated)
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
				"payment_status": "paid",
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

func TestVerifyAndParseWebhook_CheckoutCompletedTrialIsNotMarkedPaid(t *testing.T) {
	p := newWebhookTestProvider()
	payload := []byte(`{
		"id":"evt_checkout_trial",
		"type":"checkout.session.completed",
		"created":1700000000,
		"data":{"object":{
			"id":"cs_trial",
			"mode":"subscription",
			"payment_status":"no_payment_required",
			"customer":"cus_trial",
			"subscription":"sub_trial",
			"client_reference_id":"user_trial",
			"metadata":{
				"user_id":"user_trial",
				"product_type":"subscription",
				"plan":"starter",
				"interval":"monthly",
				"price_id":"price_starter_m",
				"trial_days":"14"
			}
		}}
	}`)
	sig := signTestPayload(t, payload, testWebhookSecret)
	res, err := p.VerifyAndParseWebhook(payload, sig)
	if err != nil {
		t.Fatalf("VerifyAndParseWebhook: %v", err)
	}
	if len(res.Envelopes) != 1 {
		t.Fatalf("envelopes = %d, want 1", len(res.Envelopes))
	}
	activated := res.Envelopes[0].Payload.(event.SubscriptionActivated)
	if activated.Snapshot.Status != domain.StatusTrialing {
		t.Fatalf("status = %q, want trialing", activated.Snapshot.Status)
	}
}

func TestVerifyAndParseWebhook_UnpaidSubscriptionWaitsForSettlement(t *testing.T) {
	p := newWebhookTestProvider()
	payload := []byte(`{
		"id":"evt_checkout_unpaid_subscription",
		"type":"checkout.session.completed",
		"created":1700000000,
		"data":{"object":{
			"id":"cs_unpaid_subscription",
			"mode":"subscription",
			"payment_status":"unpaid",
			"customer":"cus_subscription",
			"subscription":"sub_subscription",
			"client_reference_id":"user_subscription",
			"metadata":{"user_id":"user_subscription","product_type":"subscription","plan":"starter","interval":"monthly"}
		}}
	}`)
	sig := signTestPayload(t, payload, testWebhookSecret)
	res, err := p.VerifyAndParseWebhook(payload, sig)
	if err != nil {
		t.Fatalf("VerifyAndParseWebhook: %v", err)
	}
	if len(res.Envelopes) != 0 {
		t.Fatalf("unpaid subscription emitted entitlement: %#v", res.Envelopes)
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
				"payment_status": "paid",
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

func TestVerifyAndParseWebhook_DelayedLifetimeWaitsForAsyncSuccess(t *testing.T) {
	p := newWebhookTestProvider()
	for _, tt := range []struct {
		name      string
		eventType string
		want      int
	}{
		{name: "completed while unpaid", eventType: "checkout.session.completed", want: 0},
		{name: "asynchronous payment succeeded", eventType: "checkout.session.async_payment_succeeded", want: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			payload := []byte(fmt.Sprintf(`{
				"id":"evt_delayed_lifetime",
				"type":%q,
				"created":1700000000,
				"data":{"object":{
					"id":"cs_delayed",
					"mode":"payment",
					"payment_status":"unpaid",
					"customer":"cus_life",
					"client_reference_id":"user_life",
					"metadata":{
						"user_id":"user_life",
						"product_type":"lifetime",
						"plan":"lifetime",
						"price_id":"price_lifetime"
					}
				}}
			}`, tt.eventType))
			sig := signTestPayload(t, payload, testWebhookSecret)
			res, err := p.VerifyAndParseWebhook(payload, sig)
			if err != nil {
				t.Fatalf("VerifyAndParseWebhook: %v", err)
			}
			if len(res.Envelopes) != tt.want {
				t.Fatalf("envelopes = %d, want %d", len(res.Envelopes), tt.want)
			}
			if tt.want == 1 && res.Envelopes[0].Kind != event.KindSubscriptionActivated {
				t.Fatalf("kind = %s, want subscription activated", res.Envelopes[0].Kind)
			}
		})
	}
}

func TestVerifyAndParseWebhook_UnpaidCreditsDoNotGrantEntitlement(t *testing.T) {
	p := newWebhookTestProvider()
	payload := []byte(`{
		"id":"evt_unpaid_credits",
		"type":"checkout.session.completed",
		"created":1700000000,
		"data":{"object":{
			"id":"cs_unpaid_credits",
			"mode":"payment",
			"payment_status":"unpaid",
			"customer":"cus_credits",
			"client_reference_id":"user_credits",
			"metadata":{"user_id":"user_credits","product_type":"credits","price_id":"price_1CreditsAlpha123456789"}
		}}
	}`)
	sig := signTestPayload(t, payload, testWebhookSecret)
	res, err := p.VerifyAndParseWebhook(payload, sig)
	if err != nil {
		t.Fatalf("VerifyAndParseWebhook: %v", err)
	}
	if len(res.Envelopes) != 0 {
		t.Fatalf("unpaid credits emitted entitlements: %#v", res.Envelopes)
	}
}

func TestVerifyAndParseWebhook_NoPaymentRequiredOnlyStartsConfiguredTrial(t *testing.T) {
	p := newWebhookTestProvider()
	for _, tt := range []struct {
		name        string
		mode        string
		productType string
		priceID     string
		trialDays   string
	}{
		{name: "lifetime", mode: "payment", productType: "lifetime", priceID: "price_lifetime"},
		{name: "credits", mode: "payment", productType: "credits", priceID: "price_1CreditsAlpha123456789"},
		{name: "non-trial subscription", mode: "subscription", productType: "subscription", priceID: "price_starter_m", trialDays: "0"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			payload := []byte(fmt.Sprintf(`{
				"id":"evt_free_checkout",
				"type":"checkout.session.completed",
				"created":1700000000,
				"data":{"object":{
					"id":"cs_free",
					"mode":%q,
					"payment_status":"no_payment_required",
					"customer":"cus_free",
					"subscription":"sub_free",
					"metadata":{"user_id":"user_free","product_type":%q,"price_id":%q,"trial_days":%q}
				}}
			}`, tt.mode, tt.productType, tt.priceID, tt.trialDays))
			res, err := p.VerifyAndParseWebhook(payload, signTestPayload(t, payload, testWebhookSecret))
			if err != nil {
				t.Fatalf("VerifyAndParseWebhook: %v", err)
			}
			if len(res.Envelopes) != 0 {
				t.Fatalf("zero-payment checkout emitted entitlements: %+v", res.Envelopes)
			}
		})
	}
}

func TestVerifyAndParseWebhook_AsyncCreditsSuccessGrantsEntitlement(t *testing.T) {
	p := newWebhookTestProvider()
	payload := []byte(`{
		"id":"evt_paid_credits",
		"type":"checkout.session.async_payment_succeeded",
		"created":1700000000,
		"data":{"object":{
			"id":"cs_paid_credits",
			"mode":"payment",
			"payment_status":"paid",
			"customer":"cus_credits",
			"client_reference_id":"user_credits",
			"metadata":{"user_id":"user_credits","product_type":"credits","price_id":"price_1CreditsAlpha123456789"},
			"line_items":{"data":[{"quantity":2,"price":{"id":"price_1CreditsAlpha123456789"}}]}
		}}
	}`)
	sig := signTestPayload(t, payload, testWebhookSecret)
	res, err := p.VerifyAndParseWebhook(payload, sig)
	if err != nil {
		t.Fatalf("VerifyAndParseWebhook: %v", err)
	}
	if len(res.Envelopes) != 1 || res.Envelopes[0].Kind != event.KindCreditsPurchased {
		t.Fatalf("envelopes = %#v, want one credits purchase", res.Envelopes)
	}
	purchased := res.Envelopes[0].Payload.(event.CreditsPurchased)
	if purchased.Quantity != 2 {
		t.Fatalf("quantity = %d, want 2", purchased.Quantity)
	}
}

func TestVerifyAndParseWebhook_TerminalCheckoutCarriesReservationReleaseHint(t *testing.T) {
	for _, eventType := range []string{"checkout.session.async_payment_failed", "checkout.session.expired"} {
		t.Run(eventType, func(t *testing.T) {
			p := newWebhookTestProvider()
			payload := []byte(fmt.Sprintf(`{
				"id":"evt_terminal",
				"type":%q,
				"created":1700000000,
				"data":{"object":{"id":"cs_terminal","mode":"payment","payment_status":"unpaid","metadata":{"checkout_reservation_id":"reservation-1"}}}
			}`, eventType))
			sig := signTestPayload(t, payload, testWebhookSecret)
			res, err := p.VerifyAndParseWebhook(payload, sig)
			if err != nil {
				t.Fatalf("VerifyAndParseWebhook: %v", err)
			}
			if !res.ReleaseCheckoutReservation || res.CheckoutSessionID != "cs_terminal" || res.CheckoutReservationID != "reservation-1" || len(res.Envelopes) != 0 {
				t.Fatalf("parse result = %+v", res)
			}
		})
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
			"object": {
				"id": "sub_x",
				"customer": "cus_x",
				"status": "canceled",
				"items": {"data": [{"price": {"id": "price_starter_m", "product": "prod_starter"}}]}
			}
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
	canceled, ok := res.Envelopes[0].Payload.(event.SubscriptionCanceled)
	if !ok {
		t.Fatalf("payload=%T", res.Envelopes[0].Payload)
	}
	if canceled.Snapshot.Status != domain.StatusCanceled || canceled.Snapshot.Plan != domain.PlanStarter {
		t.Fatalf("snapshot=%+v", canceled.Snapshot)
	}
}

func TestCheckoutCreditsRequiresReliableQuantity(t *testing.T) {
	p := newWebhookTestProvider()
	mk := func(kind event.Kind, payload any) event.Envelope {
		return event.Envelope{Kind: kind, Payload: payload}
	}
	_, err := p.onCheckoutCompleted(map[string]any{
		"mode":           "payment",
		"payment_status": "paid",
		"metadata": map[string]any{
			"product_type": string(domain.ProductCredits),
			"price_id":     "price_1CreditsAlpha123456789",
			"quantity":     "100",
		},
	}, false, mk)
	if err == nil {
		t.Fatal("expected missing quantity to fail instead of silently granting one package")
	}

	envelopes, err := p.onCheckoutCompleted(map[string]any{
		"id":             "cs_123",
		"mode":           "payment",
		"payment_status": "paid",
		"metadata": map[string]any{
			"product_type": string(domain.ProductCredits),
			"price_id":     "price_1CreditsAlpha123456789",
		},
		"line_items": map[string]any{"data": []any{
			map[string]any{
				"quantity": float64(3),
				"price":    map[string]any{"id": "price_1CreditsAlpha123456789"},
			},
		}},
	}, false, mk)
	if err != nil {
		t.Fatalf("inline quantity: %v", err)
	}
	purchased := envelopes[0].Payload.(event.CreditsPurchased)
	if purchased.Quantity != 3 {
		t.Fatalf("quantity=%d, want 3", purchased.Quantity)
	}
}

func TestCheckoutCreditsValidatesAuthoritativeLineItem(t *testing.T) {
	p := newWebhookTestProvider()
	mk := func(kind event.Kind, payload any) event.Envelope {
		return event.Envelope{Kind: kind, Payload: payload}
	}

	// The Checkout Session was created with quantity 100, but the buyer used
	// adjustable quantity to pay for one. Fulfillment must use the line item.
	envelopes, err := p.onCheckoutCompleted(map[string]any{
		"id":             "cs_adjusted",
		"mode":           "payment",
		"payment_status": "paid",
		"metadata": map[string]any{
			"product_type": string(domain.ProductCredits),
			"price_id":     "price_1CreditsAlpha123456789",
			"quantity":     "100",
		},
		"line_items": map[string]any{"data": []any{map[string]any{
			"quantity": float64(1),
			"price":    map[string]any{"id": "price_1CreditsAlpha123456789"},
		}}},
	}, false, mk)
	if err != nil {
		t.Fatalf("adjusted quantity: %v", err)
	}
	purchased := envelopes[0].Payload.(event.CreditsPurchased)
	if purchased.Quantity != 1 {
		t.Fatalf("quantity=%d, want authoritative line-item quantity 1", purchased.Quantity)
	}

	_, err = p.onCheckoutCompleted(map[string]any{
		"id":             "cs_wrong_price",
		"mode":           "payment",
		"payment_status": "paid",
		"metadata": map[string]any{
			"product_type": string(domain.ProductCredits),
			"price_id":     "price_1CreditsAlpha123456789",
		},
		"line_items": map[string]any{"data": []any{map[string]any{
			"quantity": float64(1),
			"price":    map[string]any{"id": "price_different"},
		}}},
	}, false, mk)
	if !errors.Is(err, domain.ErrInvalidPriceID) {
		t.Fatalf("mismatched price error=%v, want ErrInvalidPriceID", err)
	}
}

func TestCheckoutQuantityFromProviderSessionFailsClosed(t *testing.T) {
	expectedPriceID := "price_1CreditsAlpha123456789"
	lineItems := func(items ...*stripesdk.LineItem) *stripesdk.LineItemList {
		return &stripesdk.LineItemList{Data: items}
	}
	tests := []struct {
		name string
		sess *stripesdk.CheckoutSession
	}{
		{name: "nil session"},
		{name: "nil line items", sess: &stripesdk.CheckoutSession{}},
		{name: "empty line items", sess: &stripesdk.CheckoutSession{LineItems: lineItems()}},
		{name: "nil line item", sess: &stripesdk.CheckoutSession{LineItems: lineItems(nil)}},
		{name: "nil price", sess: &stripesdk.CheckoutSession{LineItems: lineItems(&stripesdk.LineItem{Quantity: 1})}},
		{name: "wrong price", sess: &stripesdk.CheckoutSession{LineItems: lineItems(&stripesdk.LineItem{
			Quantity: 1, Price: &stripesdk.Price{ID: "price_other"},
		})}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := checkoutQuantityFromProviderSession(tt.sess, expectedPriceID); err == nil {
				t.Fatal("expected malformed provider response to fail closed")
			}
		})
	}

	valid := &stripesdk.CheckoutSession{
		Metadata: map[string]string{"quantity": "100"},
		LineItems: lineItems(&stripesdk.LineItem{
			Quantity: 1, Price: &stripesdk.Price{ID: expectedPriceID},
		}),
	}
	quantity, err := checkoutQuantityFromProviderSession(valid, expectedPriceID)
	if err != nil || quantity != 1 {
		t.Fatalf("valid provider line item = (%d, %v), want (1, nil)", quantity, err)
	}
}

func TestCheckoutCompleted_IgnoresUnownedPaymentAndRejectsUnconfiguredEntitlements(t *testing.T) {
	p := newWebhookTestProvider()
	mk := func(kind event.Kind, payload any) event.Envelope {
		return event.Envelope{Kind: kind, Payload: payload}
	}

	unowned, err := p.onCheckoutCompleted(map[string]any{
		"mode":           "payment",
		"payment_status": "paid",
		"metadata":       map[string]any{},
	}, false, mk)
	if err != nil || len(unowned) != 0 {
		t.Fatalf("unowned payment=(%#v, %v), want ignored", unowned, err)
	}

	for _, data := range []map[string]any{
		{
			"mode":           "payment",
			"payment_status": "paid",
			"metadata": map[string]any{
				"product_type": string(domain.ProductLifetime),
				"price_id":     "price_not_lifetime",
			},
		},
		{
			"mode":           "payment",
			"payment_status": "paid",
			"metadata": map[string]any{
				"product_type": string(domain.ProductCredits),
				"price_id":     "price_not_credits",
			},
			"line_items": map[string]any{"data": []any{map[string]any{
				"quantity": float64(1),
				"price":    map[string]any{"id": "price_not_credits"},
			}}},
		},
		{
			"mode":           "subscription",
			"payment_status": "paid",
			"metadata": map[string]any{
				"product_type": string(domain.ProductSubscription),
				"price_id":     "price_not_subscription",
			},
		},
	} {
		if envelopes, err := p.onCheckoutCompleted(data, false, mk); err == nil || len(envelopes) != 0 {
			t.Fatalf("unconfigured entitlement=(%#v, %v), want fail closed", envelopes, err)
		}
	}
}

func TestCheckoutCreditsRejectsProviderQuantityOutsideConfiguredBounds(t *testing.T) {
	p := newWebhookTestProvider()
	mk := func(kind event.Kind, payload any) event.Envelope {
		return event.Envelope{Kind: kind, Payload: payload}
	}
	for _, quantity := range []float64{101} {
		_, err := p.onCheckoutCompleted(map[string]any{
			"id":             "cs_quantity_bounds",
			"mode":           "payment",
			"payment_status": "paid",
			"metadata": map[string]any{
				"product_type": string(domain.ProductCredits),
				"price_id":     "price_1CreditsAlpha123456789",
			},
			"line_items": map[string]any{"data": []any{map[string]any{
				"quantity": quantity,
				"price":    map[string]any{"id": "price_1CreditsAlpha123456789"},
			}}},
		}, false, mk)
		if err == nil {
			t.Fatalf("quantity=%v should fail closed", quantity)
		}
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
