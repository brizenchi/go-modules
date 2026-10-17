package bootstrap

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/brizenchi/go-modules/modules/billing"
	"github.com/brizenchi/go-modules/modules/billing/adapter/eventbus"
	billingdomain "github.com/brizenchi/go-modules/modules/billing/domain"
	"github.com/brizenchi/go-modules/modules/billing/event"
	"github.com/brizenchi/quickstart-template/internal/feature/membership"
	"github.com/brizenchi/quickstart-template/internal/hostapi"
	"github.com/brizenchi/quickstart-template/internal/platform"
	"github.com/brizenchi/quickstart-template/internal/user"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPaymentHooksAreBoundWithoutBusinessPolicy(t *testing.T) {
	bus := eventbus.NewInProc()
	deps := hostapi.Deps{Modules: &platform.Modules{Billing: &billing.Module{Bus: bus}}}
	subscribeModuleEvents(deps)
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

func TestCancelRenewalPreservesPaidPeriodAndPurchasedCredits(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := user.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	users := user.NewRepository(db)
	if err := users.Create(t.Context(), &user.User{ID: "buyer", Email: "buyer@example.test"}); err != nil {
		t.Fatal(err)
	}
	deps := hostapi.Deps{DB: db, Users: users}.WithServices()
	if err := deps.Credits.GrantCredits(t.Context(), "buyer", "stripe", "cs_purchase", 100); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	end := now.Add(24 * time.Hour)
	snapshot := billingdomain.SubscriptionSnapshot{Plan: billingdomain.PlanPro, Status: billingdomain.StatusCanceling, PeriodEnd: &end, CancelAtPeriodEnd: true}
	if err := onSubscriptionCanceling(t.Context(), deps, event.Envelope{UserID: "buyer"}, event.SubscriptionCanceling{Snapshot: snapshot, EffectiveAt: &end}); err != nil {
		t.Fatal(err)
	}
	row := billingdomain.BillingSubscription{Plan: "pro", Status: "canceling", PeriodEnd: &end, CancelAtPeriodEnd: true}
	if !membership.Evaluate(row, now).Active || membership.Evaluate(row, end).Active {
		t.Fatal("cancellation must retain this period and end at its boundary")
	}
	if err := onSubscriptionCanceled(t.Context(), deps, event.Envelope{UserID: "buyer"}, event.SubscriptionCanceled{}); err != nil {
		t.Fatal(err)
	}
	summary, err := deps.Credits.GetCreditSummary(t.Context(), "buyer")
	if err != nil || summary.Balance != 100 {
		t.Fatalf("cancellation changed purchased credits: %+v %v", summary, err)
	}
	transactions, err := deps.Credits.ListCreditTransactions(t.Context(), "buyer", 1, 20)
	if err != nil || transactions.Total != 1 {
		t.Fatalf("cancellation created a credit refund/deduction: %+v %v", transactions, err)
	}
}
