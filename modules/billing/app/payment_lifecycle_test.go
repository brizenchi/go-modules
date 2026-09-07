package app

import (
	"errors"
	"testing"

	"github.com/brizenchi/go-modules/modules/billing/event"
	"github.com/brizenchi/go-modules/modules/billing/port"
)

func TestPaymentLifecycleHooksRetryWithoutChangingSubscriptions(t *testing.T) {
	for _, envelope := range []event.Envelope{
		{Kind: event.KindRefundUpdated, Payload: event.RefundUpdated{ProviderRefundID: "re_test"}},
		{Kind: event.KindChargeRefunded, Payload: event.ChargeRefunded{ProviderChargeID: "ch_test"}},
		{Kind: event.KindDisputeUpdated, Payload: event.DisputeUpdated{ProviderDisputeID: "dp_test"}},
	} {
		t.Run(string(envelope.Kind), func(t *testing.T) {
			provider := newMockProvider()
			provider.parseResult = &port.WebhookParseResult{
				ProviderEventID: "evt_lifecycle", Envelopes: []event.Envelope{envelope},
			}
			repository := newMockRepo()
			snapshots := &mockSubscriptionRepo{}
			bus := newMockBus()
			bus.publishErr = errors.New("host hook failed")
			service := NewWebhookService(provider, repository, snapshots, nil, bus)
			if _, err := service.Process(t.Context(), nil, "signature"); err == nil {
				t.Fatal("expected retryable error")
			}
			if repository.rows["evt_lifecycle"].Processed {
				t.Fatal("failed hook marked processed")
			}
			bus.publishErr = nil
			if _, err := service.Process(t.Context(), nil, "signature"); err != nil {
				t.Fatal(err)
			}
			if !repository.rows["evt_lifecycle"].Processed {
				t.Fatal("successful retry not persisted")
			}
			published := bus.Published()
			last := published[len(published)-1]
			if last.Kind != envelope.Kind || last.UserID != "" || last.ProviderEventID != "evt_lifecycle" {
				t.Fatalf("unmapped hook: %+v", last)
			}
			result, err := service.Process(t.Context(), nil, "signature")
			if err != nil || !result.Duplicate || len(bus.Published()) != len(published) {
				t.Fatalf("duplicate: %+v %v", result, err)
			}
			if len(snapshots.writes) != 0 {
				t.Fatal("payment lifecycle changed subscription state")
			}
		})
	}
}
