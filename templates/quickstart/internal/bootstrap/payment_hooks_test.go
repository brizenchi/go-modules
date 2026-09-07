package bootstrap

import (
	"context"
	"errors"
	"testing"

	"github.com/brizenchi/go-modules/modules/billing"
	"github.com/brizenchi/go-modules/modules/billing/adapter/eventbus"
	"github.com/brizenchi/go-modules/modules/billing/event"
	"github.com/brizenchi/quickstart-template/internal/hostapi"
	"github.com/brizenchi/quickstart-template/internal/platform"
)

func TestPaymentHooksAreBoundWithoutBusinessPolicy(t *testing.T) {
	bus := eventbus.NewInProc()
	deps := hostapi.Deps{Modules: &platform.Modules{Billing: &billing.Module{Bus: bus}}}
	subscribeModuleEvents(deps, AppConfig{})
	for _, envelope := range []event.Envelope{
		{Kind: event.KindRefundUpdated, Payload: event.RefundUpdated{}},
		{Kind: event.KindChargeRefunded, Payload: event.ChargeRefunded{}},
		{Kind: event.KindDisputeUpdated, Payload: event.DisputeUpdated{}},
	} {
		if err := bus.Publish(t.Context(), envelope); err != nil {
			t.Fatalf("no-op hook: %v", err)
		}
		envelope.Payload = "wrong payload"
		if err := bus.Publish(t.Context(), envelope); err == nil {
			t.Fatalf("%s hook not bound", envelope.Kind)
		}
	}
	failure := errors.New("retry this host failure")
	listener := billingListener(deps, func(context.Context, hostapi.Deps, event.Envelope, event.RefundUpdated) error { return failure })
	if err := listener(t.Context(), event.Envelope{Payload: event.RefundUpdated{}}); !errors.Is(err, failure) {
		t.Fatalf("hook error swallowed: %v", err)
	}
}
