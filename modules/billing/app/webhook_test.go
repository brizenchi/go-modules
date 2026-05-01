package app

import (
	"context"
	"errors"
	"testing"

	"github.com/brizenchi/go-modules/modules/billing/event"
	"github.com/brizenchi/go-modules/modules/billing/port"
)

func TestWebhook_DispatchesEnvelopesAndMarksProcessed(t *testing.T) {
	prov := newMockProvider()
	prov.parseResult = &port.WebhookParseResult{
		ProviderEventID: "evt_1",
		Type:            "customer.subscription.updated",
		UserHint:        port.UserHint{UserID: "u1"},
		RawPayload:      []byte(`{"id":"evt_1"}`),
		Envelopes: []event.Envelope{
			{Kind: event.KindSubscriptionUpdated, ProviderEventID: "evt_1"},
		},
	}
	repo := newMockRepo()
	bus := newMockBus()
	resolver := &mockResolver{}
	svc := NewWebhookService(prov, repo, resolver, bus)

	res, err := svc.Process(context.Background(), []byte(`{"id":"evt_1"}`), "sig")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Duplicate {
		t.Error("expected not duplicate")
	}
	if got := bus.Published(); len(got) != 1 || got[0].Kind != event.KindSubscriptionUpdated {
		t.Errorf("published = %v, want one SubscriptionUpdated", got)
	}
	if !repo.rows["evt_1"].Processed {
		t.Error("expected processed=true after success")
	}
	if got := bus.Published()[0].UserID; got != "u1" {
		t.Errorf("envelope user_id = %q, want u1", got)
	}
}

func TestWebhook_DuplicateEventSkipsDispatch(t *testing.T) {
	prov := newMockProvider()
	prov.parseResult = &port.WebhookParseResult{
		ProviderEventID: "evt_dup",
		Type:            "x",
		Envelopes:       []event.Envelope{{Kind: event.KindSubscriptionUpdated}},
	}
	repo := newMockRepo()
	bus := newMockBus()
	svc := NewWebhookService(prov, repo, &mockResolver{}, bus)

	if _, err := svc.Process(context.Background(), nil, "sig"); err != nil {
		t.Fatalf("first call err: %v", err)
	}
	bus2 := newMockBus()
	svc2 := NewWebhookService(prov, repo, &mockResolver{}, bus2) // share repo
	res, err := svc2.Process(context.Background(), nil, "sig")
	if err != nil {
		t.Fatalf("second call err: %v", err)
	}
	if !res.Duplicate {
		t.Error("expected duplicate=true on second call")
	}
	if got := bus2.Published(); len(got) != 0 {
		t.Errorf("dup should not dispatch, got %d events", len(got))
	}
}

func TestWebhook_ResolverFillsUserID(t *testing.T) {
	prov := newMockProvider()
	prov.parseResult = &port.WebhookParseResult{
		ProviderEventID: "evt_resolved",
		Type:            "customer.subscription.updated",
		UserHint:        port.UserHint{ProviderCustomerID: "cus_x"},
		Envelopes:       []event.Envelope{{Kind: event.KindSubscriptionUpdated}}, // no UserID
	}
	resolver := &mockResolver{resolveTo: "u-from-cus"}
	bus := newMockBus()
	svc := NewWebhookService(prov, newMockRepo(), resolver, bus)

	if _, err := svc.Process(context.Background(), nil, "sig"); err != nil {
		t.Fatalf("err: %v", err)
	}
	if resolver.calls != 1 {
		t.Errorf("resolver calls = %d, want 1", resolver.calls)
	}
	pub := bus.Published()
	if len(pub) != 1 || pub[0].UserID != "u-from-cus" {
		t.Errorf("envelope user_id = %v, want u-from-cus", pub)
	}
}

func TestWebhook_PropagatesProviderError(t *testing.T) {
	prov := newMockProvider()
	prov.parseErr = errors.New("invalid sig")
	svc := NewWebhookService(prov, newMockRepo(), nil, newMockBus())
	if _, err := svc.Process(context.Background(), nil, "sig"); err == nil {
		t.Fatal("expected error from provider")
	}
}
