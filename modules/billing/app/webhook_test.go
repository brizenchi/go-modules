package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/brizenchi/go-modules/modules/billing/domain"
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
			{
				Kind:            event.KindSubscriptionUpdated,
				ProviderEventID: "evt_1",
				OccurredAt:      time.Unix(100, 0).UTC(),
				Payload: event.SubscriptionUpdated{Snapshot: domain.SubscriptionSnapshot{
					ProviderSubscriptionID: "sub_1",
					Status:                 domain.StatusActive,
				}},
			},
		},
	}
	repo := newMockRepo()
	bus := newMockBus()
	resolver := &mockResolver{}
	snapshots := &mockSubscriptionRepo{}
	svc := NewWebhookService(prov, repo, snapshots, resolver, bus)

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
	if len(snapshots.writes) != 1 || snapshots.writes[0].userID != "u1" {
		t.Fatalf("snapshot writes = %+v", snapshots.writes)
	}
	if snapshots.writes[0].providerEventID != "evt_1" || !snapshots.writes[0].occurredAt.Equal(time.Unix(100, 0)) {
		t.Fatalf("snapshot version = %+v", snapshots.writes[0])
	}
}

func TestWebhook_StaleSnapshotStillPublishesBusinessEvent(t *testing.T) {
	prov := newMockProvider()
	prov.parseResult = &port.WebhookParseResult{
		ProviderEventID: "evt_stale",
		Type:            "customer.subscription.deleted",
		UserHint:        port.UserHint{UserID: "u1"},
		Envelopes: []event.Envelope{{
			Kind:       event.KindSubscriptionCanceled,
			OccurredAt: time.Unix(100, 0).UTC(),
			Payload: event.SubscriptionCanceled{Snapshot: domain.SubscriptionSnapshot{
				ProviderSubscriptionID: "sub_old",
				Status:                 domain.StatusCanceled,
			}},
		}},
	}
	repo := newMockRepo()
	snapshots := &mockSubscriptionRepo{skip: true}
	bus := newMockBus()
	svc := NewWebhookService(prov, repo, snapshots, &mockResolver{}, bus)

	if _, err := svc.Process(context.Background(), nil, "sig"); err != nil {
		t.Fatalf("Process: %v", err)
	}
	if got := bus.Published(); len(got) != 1 || got[0].Kind != event.KindSubscriptionCanceled {
		t.Fatalf("published = %+v, want stale event delivered once", got)
	}
	if !repo.rows["evt_stale"].Processed {
		t.Fatal("stale event should be marked processed")
	}
}

func TestWebhook_InvoiceRenewalDoesNotReplaceCanonicalSubscriptionSnapshot(t *testing.T) {
	prov := newMockProvider()
	prov.parseResult = &port.WebhookParseResult{
		ProviderEventID: "evt_invoice_renewal",
		Type:            "invoice.paid",
		UserHint:        port.UserHint{UserID: "u1"},
		Envelopes: []event.Envelope{{
			Kind:       event.KindSubscriptionRenewed,
			OccurredAt: time.Unix(200, 0).UTC(),
			Payload: event.SubscriptionRenewed{Snapshot: domain.SubscriptionSnapshot{
				ProviderSubscriptionID: "sub_x",
				Plan:                   domain.PlanPro,
				Interval:               domain.IntervalMonthly,
				Status:                 domain.StatusActive,
			}},
		}},
	}
	repo := newMockRepo()
	snapshots := &mockSubscriptionRepo{}
	bus := newMockBus()
	svc := NewWebhookService(prov, repo, snapshots, &mockResolver{}, bus)

	if _, err := svc.Process(context.Background(), nil, "sig"); err != nil {
		t.Fatalf("Process: %v", err)
	}
	if len(snapshots.writes) != 0 {
		t.Fatalf("invoice-derived snapshot wrote canonical state: %+v", snapshots.writes)
	}
	if got := bus.Published(); len(got) != 1 || got[0].Kind != event.KindSubscriptionRenewed {
		t.Fatalf("published = %+v, want renewal fact", got)
	}
}

func TestWebhook_TerminalCheckoutReleasesSessionlessReservationByMetadata(t *testing.T) {
	for _, eventType := range []string{"checkout.session.async_payment_failed", "checkout.session.expired"} {
		t.Run(eventType, func(t *testing.T) {
			prov := newMockProvider()
			prov.name = "stripe"
			prov.parseResult = &port.WebhookParseResult{
				ProviderEventID:            "evt_terminal_" + eventType,
				Type:                       eventType,
				CheckoutSessionID:          "cs_terminal",
				CheckoutReservationID:      "reservation-1",
				ReleaseCheckoutReservation: true,
			}
			reservations := newMockCustomerStore(port.Customer{})
			reservations.reservation = &domain.BillingCheckoutReservation{
				UserID: "u1", Provider: "stripe", ReservationID: "reservation-1",
				IntentKey: "intent-1", SessionID: "", ExpiresAt: time.Now().Add(time.Hour),
			}
			repo := newMockRepo()
			svc := NewWebhookService(prov, repo, &mockSubscriptionRepo{}, &mockResolver{}, newMockBus(), reservations)

			if _, err := svc.Process(context.Background(), nil, "sig"); err != nil {
				t.Fatalf("Process: %v", err)
			}
			if reservations.reservation != nil {
				t.Fatalf("terminal checkout reservation was not released: %+v", reservations.reservation)
			}
			if !repo.rows[prov.parseResult.ProviderEventID].Processed {
				t.Fatal("terminal checkout should be marked processed after release")
			}
		})
	}
}

func TestWebhook_LinksCompletedCheckoutAndReleasesOnTerminalSubscription(t *testing.T) {
	prov := newMockProvider()
	prov.name = "stripe"
	reservations := newMockCustomerStore(port.Customer{})
	reservations.reservation = &domain.BillingCheckoutReservation{
		UserID: "u1", Provider: "stripe", ReservationID: "reservation-1",
		IntentKey: "intent-1", SessionID: "cs_complete", ExpiresAt: time.Now().Add(time.Hour),
	}
	prov.parseResult = &port.WebhookParseResult{
		ProviderEventID:        "evt_checkout_complete",
		Type:                   "checkout.session.completed",
		CheckoutSessionID:      "cs_complete",
		CheckoutSubscriptionID: "sub_1",
	}
	svc := NewWebhookService(prov, newMockRepo(), &mockSubscriptionRepo{}, &mockResolver{}, newMockBus(), reservations)
	if _, err := svc.Process(context.Background(), nil, "sig"); err != nil {
		t.Fatalf("complete Process: %v", err)
	}
	if reservations.reservation.ProviderSubscriptionID != "sub_1" {
		t.Fatalf("linked reservation = %+v", reservations.reservation)
	}

	prov.parseResult = &port.WebhookParseResult{
		ProviderEventID: "evt_subscription_canceled",
		Type:            "customer.subscription.deleted",
		UserHint:        port.UserHint{UserID: "u1"},
		Envelopes: []event.Envelope{{
			Kind:       event.KindSubscriptionCanceled,
			Provider:   "stripe",
			OccurredAt: time.Now().UTC(),
			Payload: event.SubscriptionCanceled{Snapshot: domain.SubscriptionSnapshot{
				ProviderSubscriptionID: "sub_1",
				Status:                 domain.StatusCanceled,
			}},
		}},
	}
	if _, err := svc.Process(context.Background(), nil, "sig"); err != nil {
		t.Fatalf("cancel Process: %v", err)
	}
	if reservations.reservation != nil {
		t.Fatalf("terminal subscription did not release reservation: %+v", reservations.reservation)
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
	svc := NewWebhookService(prov, repo, &mockSubscriptionRepo{}, &mockResolver{}, bus)

	if _, err := svc.Process(context.Background(), nil, "sig"); err != nil {
		t.Fatalf("first call err: %v", err)
	}
	bus2 := newMockBus()
	svc2 := NewWebhookService(prov, repo, &mockSubscriptionRepo{}, &mockResolver{}, bus2) // share repo
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
	svc := NewWebhookService(prov, newMockRepo(), &mockSubscriptionRepo{}, resolver, bus)

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
	svc := NewWebhookService(prov, newMockRepo(), &mockSubscriptionRepo{}, nil, newMockBus())
	if _, err := svc.Process(context.Background(), nil, "sig"); err == nil {
		t.Fatal("expected error from provider")
	}
}

func TestWebhook_ListenerFailureLeavesEventForStripeRetry(t *testing.T) {
	prov := newMockProvider()
	prov.parseResult = &port.WebhookParseResult{
		ProviderEventID: "evt_retry",
		Type:            "checkout.session.completed",
		UserHint:        port.UserHint{UserID: "u1"},
		Envelopes: []event.Envelope{{
			Kind:            event.KindCreditsPurchased,
			ProviderEventID: "evt_retry",
			Payload:         event.CreditsPurchased{TotalCredits: 100},
		}},
	}
	repo := newMockRepo()
	bus := newMockBus()
	bus.publishErr = errors.New("database unavailable")
	svc := NewWebhookService(prov, repo, &mockSubscriptionRepo{}, &mockResolver{}, bus)

	if _, err := svc.Process(context.Background(), nil, "sig"); err == nil {
		t.Fatal("expected listener failure")
	}
	if repo.rows["evt_retry"].Processed {
		t.Fatal("failed listener must not mark webhook processed")
	}

	bus.publishErr = nil
	if _, err := svc.Process(context.Background(), nil, "sig"); err != nil {
		t.Fatalf("retry: %v", err)
	}
	if !repo.rows["evt_retry"].Processed {
		t.Fatal("successful retry should mark webhook processed")
	}
}
