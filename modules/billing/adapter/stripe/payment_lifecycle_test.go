package stripe

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/brizenchi/go-modules/modules/billing/domain"
	"github.com/brizenchi/go-modules/modules/billing/event"
)

func TestVerifiedPaymentLifecycleEvents(t *testing.T) {
	for _, eventType := range []string{
		"refund.created", "refund.updated", "refund.failed", "charge.refund.updated",
		"charge.dispute.created", "charge.dispute.updated", "charge.dispute.closed",
		"charge.dispute.funds_withdrawn", "charge.dispute.funds_reinstated",
	} {
		for _, expanded := range []bool{false, true} {
			name := eventType + "/string"
			if expanded {
				name = eventType + "/expanded"
			}
			t.Run(name, func(t *testing.T) {
				var charge, paymentIntent any = "ch_test", "pi_test"
				if expanded {
					charge, paymentIntent = map[string]any{"id": "ch_test"}, map[string]any{"id": "pi_test"}
				}
				object := map[string]any{
					"id": "object_test", "charge": charge, "payment_intent": paymentIntent,
					"amount": 250, "currency": "usd", "status": "pending", "reason": "requested_by_customer",
					"failure_reason": "declined", "metadata": map[string]any{"user_id": "user_test"},
				}
				payload, err := json.Marshal(map[string]any{"id": "evt_test", "type": eventType, "created": 1700000000, "data": map[string]any{"object": object}})
				if err != nil {
					t.Fatal(err)
				}
				provider := newWebhookTestProvider()
				parsed, err := provider.VerifyAndParseWebhook(payload, signTestPayload(t, payload, testWebhookSecret))
				if err != nil {
					t.Fatal(err)
				}
				if len(parsed.Envelopes) != 1 {
					t.Fatalf("events: %+v", parsed.Envelopes)
				}
				envelope := parsed.Envelopes[0]
				if envelope.Provider != "stripe" || envelope.ProviderEventID != "evt_test" || envelope.UserID != "user_test" || !envelope.OccurredAt.Equal(time.Unix(1700000000, 0)) {
					t.Fatalf("provenance: %+v", envelope)
				}
				switch lifecycle := envelope.Payload.(type) {
				case event.RefundUpdated:
					if envelope.Kind != event.KindRefundUpdated || lifecycle.ProviderEventType != eventType || lifecycle.ProviderRefundID != "object_test" || lifecycle.ProviderChargeID != "ch_test" || lifecycle.ProviderPaymentIntentID != "pi_test" || lifecycle.Amount != 250 || lifecycle.Currency != "usd" || lifecycle.Status != "pending" || lifecycle.FailureReason != "declined" {
						t.Fatalf("refund: %+v", lifecycle)
					}
				case event.DisputeUpdated:
					if envelope.Kind != event.KindDisputeUpdated || lifecycle.ProviderEventType != eventType || lifecycle.ProviderDisputeID != "object_test" || lifecycle.ProviderChargeID != "ch_test" || lifecycle.ProviderPaymentIntentID != "pi_test" || lifecycle.Amount != 250 || lifecycle.Currency != "usd" || lifecycle.Status != "pending" || lifecycle.Reason != "requested_by_customer" {
						t.Fatalf("dispute: %+v", lifecycle)
					}
				default:
					t.Fatalf("unexpected business event: %T", lifecycle)
				}
				if _, err := provider.VerifyAndParseWebhook(payload, signTestPayload(t, payload, "wrong-secret")); !errors.Is(err, domain.ErrSignatureInvalid) {
					t.Fatalf("unverified lifecycle accepted: %v", err)
				}
			})
		}
	}
}

func TestChargeRefundedCarriesCumulativeAmounts(t *testing.T) {
	for _, refunded := range []int64{250, 1000} {
		fullyRefunded := refunded == 1000
		payload, err := json.Marshal(map[string]any{
			"id": "evt_charge", "type": "charge.refunded", "data": map[string]any{"object": map[string]any{
				"id": "ch_test", "payment_intent": "pi_test", "customer": map[string]any{"id": "cus_test"},
				"amount": 1000, "amount_refunded": refunded, "refunded": fullyRefunded, "currency": "usd",
			}},
		})
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := newWebhookTestProvider().VerifyAndParseWebhook(payload, signTestPayload(t, payload, testWebhookSecret))
		if err != nil {
			t.Fatal(err)
		}
		if len(parsed.Envelopes) != 1 || parsed.Envelopes[0].Kind != event.KindChargeRefunded {
			t.Fatalf("events: %+v", parsed.Envelopes)
		}
		charge := parsed.Envelopes[0].Payload.(event.ChargeRefunded)
		if charge.Amount != 1000 || charge.AmountRefunded != refunded || charge.FullyRefunded != fullyRefunded || charge.ProviderCustomerID != "cus_test" || charge.ProviderChargeID != "ch_test" || charge.ProviderPaymentIntentID != "pi_test" || charge.Currency != "usd" {
			t.Fatalf("cumulative charge: %+v", charge)
		}
		if parsed.UserHint.ProviderCustomerID != "cus_test" || parsed.Envelopes[0].UserID != "" {
			t.Fatalf("user mapping: %+v", parsed.UserHint)
		}
	}
}
