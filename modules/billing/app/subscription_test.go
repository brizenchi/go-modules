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

func TestCancel_PublishesCancelingEvent(t *testing.T) {
	prov := newMockProvider()
	store := newMockCustomerStore(port.Customer{
		UserID:                 "u1",
		ProviderCustomerID:     "cus_x",
		ProviderSubscriptionID: "sub_x",
	})
	bus := newMockBus()
	svc := NewSubscriptionService(prov, store, bus)

	res, err := svc.Cancel(context.Background(), "u1", domain.CancelAtPeriodEnd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prov.cancelCalls != 1 {
		t.Errorf("provider cancel calls = %d, want 1", prov.cancelCalls)
	}
	if res.Mode != domain.CancelAtPeriodEnd {
		t.Errorf("mode = %s, want end_of_period", res.Mode)
	}
	pub := bus.Published()
	if len(pub) != 1 || pub[0].Kind != event.KindSubscriptionCanceling {
		t.Errorf("expected one Canceling event, got %v", pub)
	}
}

func TestCancel_RejectsInvalidMode(t *testing.T) {
	svc := NewSubscriptionService(newMockProvider(), newMockCustomerStore(port.Customer{}), newMockBus())
	_, err := svc.Cancel(context.Background(), "u1", domain.CancelMode("immediate"))
	if !errors.Is(err, domain.ErrInvalidCancelMode) {
		t.Errorf("expected ErrInvalidCancelMode, got %v", err)
	}
}

func TestCancel_NoActiveSubscription(t *testing.T) {
	svc := NewSubscriptionService(newMockProvider(), newMockCustomerStore(port.Customer{UserID: "u1"}), newMockBus())
	_, err := svc.Cancel(context.Background(), "u1", domain.CancelAtPeriodEnd)
	if !errors.Is(err, domain.ErrNoActiveSubscription) {
		t.Errorf("expected ErrNoActiveSubscription, got %v", err)
	}
}

func TestCancel_ReturnsPublishError(t *testing.T) {
	bus := newMockBus()
	bus.publishErr = errors.New("publish failed")
	svc := NewSubscriptionService(newMockProvider(), newMockCustomerStore(port.Customer{
		UserID:                 "u1",
		ProviderSubscriptionID: "sub_x",
	}), bus)

	_, err := svc.Cancel(context.Background(), "u1", domain.CancelAtPeriodEnd)
	if !errors.Is(err, bus.publishErr) {
		t.Fatalf("error = %v, want publish error", err)
	}
}

func TestReactivate_PublishesReactivatedEvent(t *testing.T) {
	prov := newMockProvider()
	deadline := time.Now().UTC().Add(72 * time.Hour)
	prov.subSnapshot = &domain.SubscriptionSnapshot{
		ProviderSubscriptionID: "sub_x",
		Status:                 domain.StatusCanceling,
		CancelAtPeriodEnd:      false,
		CancelEffectiveAt:      &deadline,
	}
	store := newMockCustomerStore(port.Customer{
		UserID:                 "u1",
		ProviderSubscriptionID: "sub_x",
		SubscriptionStatus:     domain.StatusCanceling,
	})
	bus := newMockBus()
	svc := NewSubscriptionService(prov, store, bus)

	id, err := svc.Reactivate(context.Background(), "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "sub_x" {
		t.Errorf("returned subscription id = %q, want sub_x", id)
	}
	if prov.reactivateCalls != 1 {
		t.Errorf("provider reactivate calls = %d, want 1", prov.reactivateCalls)
	}
	pub := bus.Published()
	if len(pub) != 1 || pub[0].Kind != event.KindSubscriptionReactivated {
		t.Errorf("expected one Reactivated event, got %v", pub)
	}
}

func TestReactivate_NoSubscription(t *testing.T) {
	svc := NewSubscriptionService(newMockProvider(), newMockCustomerStore(port.Customer{UserID: "u1"}), newMockBus())
	_, err := svc.Reactivate(context.Background(), "u1")
	if !errors.Is(err, domain.ErrNoSubscriptionToReactive) {
		t.Errorf("expected ErrNoSubscriptionToReactive, got %v", err)
	}
}

func TestReactivate_ReturnsPublishError(t *testing.T) {
	bus := newMockBus()
	bus.publishErr = errors.New("publish failed")
	prov := newMockProvider()
	prov.subSnapshot = &domain.SubscriptionSnapshot{
		ProviderSubscriptionID: "sub_x",
		Status:                 domain.StatusCanceling,
		CancelAtPeriodEnd:      true,
	}
	svc := NewSubscriptionService(prov, newMockCustomerStore(port.Customer{
		UserID:                 "u1",
		ProviderSubscriptionID: "sub_x",
		SubscriptionStatus:     domain.StatusCanceling,
	}), bus)

	_, err := svc.Reactivate(context.Background(), "u1")
	if !errors.Is(err, bus.publishErr) {
		t.Fatalf("error = %v, want publish error", err)
	}
}

func TestReactivate_RejectsSubscriptionThatIsNotCanceling(t *testing.T) {
	prov := newMockProvider()
	svc := NewSubscriptionService(prov, newMockCustomerStore(port.Customer{
		UserID:                 "u1",
		ProviderSubscriptionID: "sub_x",
		SubscriptionStatus:     domain.StatusActive,
	}), newMockBus())

	_, err := svc.Reactivate(context.Background(), "u1")
	if !errors.Is(err, domain.ErrNoSubscriptionToReactive) {
		t.Fatalf("error=%v, want ErrNoSubscriptionToReactive", err)
	}
	if prov.reactivateCalls != 0 {
		t.Fatalf("provider reactivation calls=%d, want 0", prov.reactivateCalls)
	}
}

func TestReactivate_RejectsCanceledSubscriptionEvenWithHistoricalCancelTimestamp(t *testing.T) {
	prov := newMockProvider()
	future := time.Now().UTC().Add(time.Hour)
	prov.subSnapshot = &domain.SubscriptionSnapshot{
		ProviderSubscriptionID: "sub_x",
		Status:                 domain.StatusCanceled,
		CancelEffectiveAt:      &future,
	}
	svc := NewSubscriptionService(prov, newMockCustomerStore(port.Customer{
		UserID:                 "u1",
		ProviderSubscriptionID: "sub_x",
		SubscriptionStatus:     domain.StatusCanceled,
	}), newMockBus())

	_, err := svc.Reactivate(context.Background(), "u1")
	if !errors.Is(err, domain.ErrNoSubscriptionToReactive) {
		t.Fatalf("error=%v, want ErrNoSubscriptionToReactive", err)
	}
	if prov.reactivateCalls != 0 {
		t.Fatalf("provider reactivation calls=%d, want 0", prov.reactivateCalls)
	}
}

func TestChange_PublishesUpdatedEvent(t *testing.T) {
	prov := newMockProvider()
	store := newMockCustomerStore(port.Customer{
		UserID:                 "u1",
		ProviderSubscriptionID: "sub_x",
	})
	bus := newMockBus()
	svc := NewSubscriptionService(prov, store, bus)

	res, err := svc.Change(context.Background(), "u1", domain.SubscriptionChangeInput{
		Plan:     domain.PlanPro,
		Interval: domain.IntervalMonthly,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prov.changeCalls != 1 {
		t.Errorf("provider change calls = %d, want 1", prov.changeCalls)
	}
	if res.Snapshot.Plan != domain.PlanPro {
		t.Errorf("plan = %s, want pro", res.Snapshot.Plan)
	}
	pub := bus.Published()
	if len(pub) != 1 || pub[0].Kind != event.KindSubscriptionUpdated {
		t.Errorf("expected one Updated event, got %v", pub)
	}
}

func TestChange_NoActiveSubscription(t *testing.T) {
	svc := NewSubscriptionService(newMockProvider(), newMockCustomerStore(port.Customer{UserID: "u1"}), newMockBus())
	_, err := svc.Change(context.Background(), "u1", domain.SubscriptionChangeInput{
		Plan:     domain.PlanPro,
		Interval: domain.IntervalMonthly,
	})
	if !errors.Is(err, domain.ErrNoActiveSubscription) {
		t.Errorf("expected ErrNoActiveSubscription, got %v", err)
	}
}

func TestChange_ReturnsPublishError(t *testing.T) {
	bus := newMockBus()
	bus.publishErr = errors.New("publish failed")
	svc := NewSubscriptionService(newMockProvider(), newMockCustomerStore(port.Customer{
		UserID:                 "u1",
		ProviderSubscriptionID: "sub_x",
	}), bus)

	_, err := svc.Change(context.Background(), "u1", domain.SubscriptionChangeInput{
		Plan:     domain.PlanPro,
		Interval: domain.IntervalMonthly,
	})
	if !errors.Is(err, bus.publishErr) {
		t.Fatalf("error = %v, want publish error", err)
	}
}

func TestOpenBillingPortal_ReturnsURL(t *testing.T) {
	prov := newMockProvider()
	store := newMockCustomerStore(port.Customer{
		UserID:             "u1",
		ProviderCustomerID: "cus_x",
	})
	svc := NewSubscriptionService(prov, store, newMockBus())

	res, err := svc.OpenBillingPortal(context.Background(), "u1", "https://app.test/billing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.URL == "" {
		t.Fatal("expected portal URL")
	}
}

func TestOpenBillingPortal_NoCustomer(t *testing.T) {
	svc := NewSubscriptionService(newMockProvider(), newMockCustomerStore(port.Customer{UserID: "u1"}), newMockBus())
	_, err := svc.OpenBillingPortal(context.Background(), "u1", "https://app.test/billing")
	if !errors.Is(err, domain.ErrNoBillingCustomer) {
		t.Errorf("expected ErrNoBillingCustomer, got %v", err)
	}
}

func TestChange_ResolvesMonthlyToYearlyAsImmediateResetCycle(t *testing.T) {
	prov := newMockProvider()
	prov.subSnapshot = &domain.SubscriptionSnapshot{
		ProviderSubscriptionID: "sub_x",
		Plan:                   domain.PlanPro,
		Interval:               domain.IntervalMonthly,
		Status:                 domain.StatusActive,
	}
	store := newMockCustomerStore(port.Customer{
		UserID:                 "u1",
		ProviderSubscriptionID: "sub_x",
	})
	res, err := NewSubscriptionService(prov, store, newMockBus()).Change(context.Background(), "u1", domain.SubscriptionChangeInput{
		Plan:     domain.PlanPro,
		Interval: domain.IntervalYearly,
		Mode:     domain.ChangeModePeriodEnd, // untrusted caller override is ignored
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Mode != domain.ChangeModeImmediateResetCycle {
		t.Errorf("mode = %s", res.Mode)
	}
}

func TestPreviewChange_IgnoresCallerSuppliedMode(t *testing.T) {
	prov := newMockProvider()
	prov.subSnapshot = &domain.SubscriptionSnapshot{
		ProviderSubscriptionID: "sub_x",
		Plan:                   domain.PlanPro,
		Interval:               domain.IntervalYearly,
		Status:                 domain.StatusActive,
	}
	store := newMockCustomerStore(port.Customer{
		UserID:                 "u1",
		ProviderCustomerID:     "cus_x",
		ProviderSubscriptionID: "sub_x",
	})
	res, err := NewSubscriptionService(prov, store, newMockBus()).PreviewChange(context.Background(), "u1", domain.SubscriptionPreviewInput{
		Plan:     domain.PlanPro,
		Interval: domain.IntervalMonthly,
		Mode:     domain.ChangeModeImmediateResetCycle,
	})
	if err != nil {
		t.Fatalf("PreviewChange: %v", err)
	}
	if res.Mode != domain.ChangeModePeriodEnd {
		t.Fatalf("mode = %s, want server-selected period_end", res.Mode)
	}
}

func TestChange_ResolvesYearlyToMonthlyAsPeriodEnd(t *testing.T) {
	prov := newMockProvider()
	prov.subSnapshot = &domain.SubscriptionSnapshot{
		ProviderSubscriptionID: "sub_x",
		Plan:                   domain.PlanPro,
		Interval:               domain.IntervalYearly,
		Status:                 domain.StatusActive,
	}
	store := newMockCustomerStore(port.Customer{
		UserID:                 "u1",
		ProviderSubscriptionID: "sub_x",
	})
	res, err := NewSubscriptionService(prov, store, newMockBus()).Change(context.Background(), "u1", domain.SubscriptionChangeInput{
		Plan:     domain.PlanPro,
		Interval: domain.IntervalMonthly,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Mode != domain.ChangeModePeriodEnd {
		t.Errorf("mode = %s", res.Mode)
	}
	if prov.scheduleCalls != 1 {
		t.Errorf("schedule calls = %d, want 1", prov.scheduleCalls)
	}
}

func TestPreviewChange_UsesProviderPreview(t *testing.T) {
	prov := newMockProvider()
	store := newMockCustomerStore(port.Customer{
		UserID:                 "u1",
		ProviderCustomerID:     "cus_x",
		ProviderSubscriptionID: "sub_x",
	})
	res, err := NewSubscriptionService(prov, store, newMockBus()).PreviewChange(context.Background(), "u1", domain.SubscriptionPreviewInput{
		Plan:     domain.PlanPremium,
		Interval: domain.IntervalMonthly,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.TargetPlan != domain.PlanPremium {
		t.Errorf("target plan = %s", res.TargetPlan)
	}
}

func TestPendingChangeBlocksFurtherMutations(t *testing.T) {
	for _, snapshot := range []*domain.SubscriptionSnapshot{
		{ScheduledChange: &domain.ScheduledChange{Plan: domain.PlanStarter}},
		{PendingPayment: &domain.PendingPayment{AmountDue: 100}},
	} {
		prov := newMockProvider()
		prov.subSnapshot = snapshot
		svc := NewSubscriptionService(prov, newMockCustomerStore(port.Customer{UserID: "u1", ProviderSubscriptionID: "sub_x"}), newMockBus())
		if _, err := svc.Change(t.Context(), "u1", domain.SubscriptionChangeInput{Plan: domain.PlanPro, Interval: domain.IntervalMonthly}); !errors.Is(err, domain.ErrPendingSubscriptionChange) {
			t.Fatalf("change: %v", err)
		}
		if _, err := svc.Cancel(t.Context(), "u1", domain.CancelAtPeriodEnd); !errors.Is(err, domain.ErrPendingSubscriptionChange) {
			t.Fatalf("cancel: %v", err)
		}
		if prov.cancelCalls != 0 {
			t.Fatal("unexpected provider cancellation")
		}
	}
}

func TestCrossIntervalDowngradePreservesPaidTier(t *testing.T) {
	current := &domain.SubscriptionSnapshot{Plan: domain.PlanPremium, Interval: domain.IntervalMonthly}
	if mode := resolveChangeMode(current, domain.SubscriptionChangeInput{Plan: domain.PlanStarter, Interval: domain.IntervalYearly}); mode != domain.ChangeModePeriodEnd {
		t.Fatalf("downgrade mode: %s", mode)
	}
}

func TestTierUpgradesTakeEffectImmediatelyAcrossBillingIntervals(t *testing.T) {
	for _, currentInterval := range []domain.BillingInterval{domain.IntervalMonthly, domain.IntervalYearly} {
		for _, interval := range []domain.BillingInterval{domain.IntervalMonthly, domain.IntervalYearly} {
			provider := newMockProvider()
			provider.subSnapshot = &domain.SubscriptionSnapshot{Plan: domain.PlanPro, Interval: currentInterval, Status: domain.StatusActive}
			service := NewSubscriptionService(provider, newMockCustomerStore(port.Customer{UserID: "u1", ProviderCustomerID: "cus_x", ProviderSubscriptionID: "sub_x"}), newMockBus())
			selection := domain.SubscriptionChangeInput{Plan: domain.PlanPremium, Interval: interval}
			result, err := service.Change(t.Context(), "u1", selection)
			if err != nil {
				t.Fatal(err)
			}
			want := domain.ChangeModeImmediateProrated
			if currentInterval != interval {
				want = domain.ChangeModeImmediateResetCycle
			}
			if result.Mode != want || provider.changeCalls != 1 || provider.scheduleCalls != 0 {
				t.Fatalf("upgrade %s -> %s mode=%s, want %s", currentInterval, interval, result.Mode, want)
			}
			preview, err := service.PreviewChange(t.Context(), "u1", domain.SubscriptionPreviewInput(selection))
			if err != nil || preview.Mode != want {
				t.Fatalf("preview mode disagrees with confirmation: %+v, %v", preview, err)
			}
		}
	}
}

type cancelScheduleTestProvider struct {
	*mockProvider
	id string
}

func (p *cancelScheduleTestProvider) CancelScheduledChange(_ context.Context, id string) error {
	p.id = id
	return nil
}

func TestCancelScheduledChangeUsesCustomerSubscriptionAndRejectsMissingCapability(t *testing.T) {
	provider := &cancelScheduleTestProvider{mockProvider: newMockProvider()}
	store := newMockCustomerStore(port.Customer{UserID: "u1", ProviderSubscriptionID: "sub_owned"})
	service := NewSubscriptionService(provider, store, newMockBus())
	if err := service.CancelScheduledChange(t.Context(), "u1"); err != nil || provider.id != "sub_owned" {
		t.Fatalf("cancel schedule: %s / %v", provider.id, err)
	}
	store.loadErr = errors.New("db unavailable")
	if err := service.CancelScheduledChange(t.Context(), "u1"); !errors.Is(err, store.loadErr) {
		t.Fatalf("lookup error: %v", err)
	}
	store.loadErr = nil
	store.customer.ProviderSubscriptionID = ""
	if err := service.CancelScheduledChange(t.Context(), "u1"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("missing subscription: %v", err)
	}
	store.customer.ProviderSubscriptionID = "sub_owned"
	service = NewSubscriptionService(newMockProvider(), store, newMockBus())
	if err := service.CancelScheduledChange(t.Context(), "u1"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("unsupported provider: %v", err)
	}
}
