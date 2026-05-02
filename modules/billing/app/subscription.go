package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/brizenchi/go-modules/modules/billing/domain"
	"github.com/brizenchi/go-modules/modules/billing/event"
	"github.com/brizenchi/go-modules/modules/billing/port"
)

// SubscriptionService handles user-initiated subscription mutations.
type SubscriptionService struct {
	provider  port.Provider
	customers port.CustomerStore
	bus       port.EventBus
}

func NewSubscriptionService(p port.Provider, c port.CustomerStore, b port.EventBus) *SubscriptionService {
	return &SubscriptionService{provider: p, customers: c, bus: b}
}

// CancelResult describes when a cancellation will take effect.
type CancelResult struct {
	ProviderSubscriptionID string
	Mode                   domain.CancelMode
	EffectiveAt            *time.Time
}

// ChangeResult describes an in-place subscription mutation.
type ChangeResult struct {
	ProviderSubscriptionID string
	Snapshot               domain.SubscriptionSnapshot
	Mode                   domain.SubscriptionChangeMode
}

// Cancel schedules a cancellation. The subscription remains active until
// the effective time; a SubscriptionCanceling event is published so
// listeners can revoke benefits incrementally if desired.
func (s *SubscriptionService) Cancel(ctx context.Context, userID string, mode domain.CancelMode) (*CancelResult, error) {
	userID = strings.TrimSpace(userID)
	if !mode.Valid() {
		return nil, domain.ErrInvalidCancelMode
	}

	cust, err := s.customers.LoadCustomer(ctx, userID)
	if err != nil {
		return nil, err
	}
	subID := strings.TrimSpace(cust.ProviderSubscriptionID)
	if subID == "" {
		return nil, domain.ErrNoActiveSubscription
	}

	if err := s.provider.CancelSubscription(ctx, subID, mode); err != nil {
		return nil, err
	}

	var effectiveAt *time.Time
	if mode == domain.CancelIn3Days {
		t := time.Now().Add(3 * 24 * time.Hour).UTC()
		effectiveAt = &t
	}

	snap, _ := s.provider.GetSubscription(ctx, subID)
	if snap == nil {
		snap = &domain.SubscriptionSnapshot{ProviderSubscriptionID: subID, ProviderCustomerID: cust.ProviderCustomerID}
	}
	if effectiveAt == nil {
		effectiveAt = snap.CancelEffectiveAt
	}

	s.bus.Publish(ctx, event.Envelope{
		Kind:       event.KindSubscriptionCanceling,
		UserID:     userID,
		Provider:   s.provider.Name(),
		OccurredAt: time.Now().UTC(),
		Payload: event.SubscriptionCanceling{
			Snapshot:    *snap,
			Mode:        mode,
			EffectiveAt: effectiveAt,
		},
	})

	return &CancelResult{
		ProviderSubscriptionID: subID,
		Mode:                   mode,
		EffectiveAt:            effectiveAt,
	}, nil
}

// Change switches the user onto another paid subscription price.
//
// Professional default:
// - immediately apply the new plan
// - prorate the current cycle
// - only keep the update if payment succeeds
func (s *SubscriptionService) Change(ctx context.Context, userID string, in domain.SubscriptionChangeInput) (*ChangeResult, error) {
	userID = strings.TrimSpace(userID)
	if !in.Plan.Valid() || in.Plan == domain.PlanFree {
		return nil, fmt.Errorf("%w: paid plan required", domain.ErrInvalidInput)
	}
	if !in.Interval.Valid() {
		return nil, fmt.Errorf("%w: billing interval required", domain.ErrInvalidInput)
	}

	cust, err := s.customers.LoadCustomer(ctx, userID)
	if err != nil {
		return nil, err
	}
	subID := strings.TrimSpace(cust.ProviderSubscriptionID)
	if subID == "" {
		return nil, domain.ErrNoActiveSubscription
	}

	current, err := s.provider.GetSubscription(ctx, subID)
	if err != nil {
		return nil, err
	}
	mode := resolveChangeMode(current, in)
	in.Mode = mode

	var snap *domain.SubscriptionSnapshot
	switch mode {
	case domain.ChangeModePeriodEnd:
		snap, err = s.provider.ScheduleSubscriptionChange(ctx, subID, in)
	default:
		snap, err = s.provider.ChangeSubscription(ctx, subID, in)
	}
	if err != nil {
		return nil, err
	}
	if snap == nil {
		snap = &domain.SubscriptionSnapshot{ProviderSubscriptionID: subID}
	}

	s.bus.Publish(ctx, event.Envelope{
		Kind:       event.KindSubscriptionUpdated,
		UserID:     userID,
		Provider:   s.provider.Name(),
		OccurredAt: time.Now().UTC(),
		Payload:    event.SubscriptionUpdated{Snapshot: *snap},
	})

	return &ChangeResult{
		ProviderSubscriptionID: subID,
		Snapshot:               *snap,
		Mode:                   mode,
	}, nil
}

// Reactivate clears a pending cancellation.
func (s *SubscriptionService) Reactivate(ctx context.Context, userID string) (string, error) {
	userID = strings.TrimSpace(userID)
	cust, err := s.customers.LoadCustomer(ctx, userID)
	if err != nil {
		return "", err
	}
	subID := strings.TrimSpace(cust.ProviderSubscriptionID)
	if subID == "" {
		return "", domain.ErrNoSubscriptionToReactive
	}
	if err := s.provider.ReactivateSubscription(ctx, subID); err != nil {
		return "", err
	}

	snap, _ := s.provider.GetSubscription(ctx, subID)
	if snap == nil {
		snap = &domain.SubscriptionSnapshot{ProviderSubscriptionID: subID, ProviderCustomerID: cust.ProviderCustomerID, Status: domain.StatusActive}
	}

	s.bus.Publish(ctx, event.Envelope{
		Kind:       event.KindSubscriptionReactivated,
		UserID:     userID,
		Provider:   s.provider.Name(),
		OccurredAt: time.Now().UTC(),
		Payload:    event.SubscriptionReactivated{Snapshot: *snap},
	})

	return subID, nil
}

// OpenBillingPortal returns a hosted self-serve billing management URL.
func (s *SubscriptionService) OpenBillingPortal(ctx context.Context, userID, returnURL string) (*domain.PortalSessionResult, error) {
	userID = strings.TrimSpace(userID)
	returnURL = strings.TrimSpace(returnURL)
	if returnURL == "" {
		return nil, fmt.Errorf("%w: return_url required", domain.ErrInvalidInput)
	}

	cust, err := s.customers.LoadCustomer(ctx, userID)
	if err != nil {
		return nil, err
	}
	customerID := strings.TrimSpace(cust.ProviderCustomerID)
	if customerID == "" {
		return nil, domain.ErrNoBillingCustomer
	}
	return s.provider.CreateBillingPortalSession(ctx, customerID, returnURL)
}

// PreviewChange returns a user-facing preview of a plan or interval switch.
func (s *SubscriptionService) PreviewChange(ctx context.Context, userID string, in domain.SubscriptionPreviewInput) (*domain.SubscriptionPreview, error) {
	userID = strings.TrimSpace(userID)
	if !in.Plan.Valid() || in.Plan == domain.PlanFree {
		return nil, fmt.Errorf("%w: paid plan required", domain.ErrInvalidInput)
	}
	if !in.Interval.Valid() {
		return nil, fmt.Errorf("%w: billing interval required", domain.ErrInvalidInput)
	}

	cust, err := s.customers.LoadCustomer(ctx, userID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(cust.ProviderSubscriptionID) == "" {
		return &domain.SubscriptionPreview{
			Currency:         "usd",
			AmountDueNow:     0,
			TargetPlan:       in.Plan,
			TargetInterval:   in.Interval,
			Mode:             domain.ChangeModeImmediateResetCycle,
			ImmediateCharge:  true,
			Message:          "new subscription will be created through checkout",
		}, nil
	}

	current, err := s.provider.GetSubscription(ctx, cust.ProviderSubscriptionID)
	if err != nil {
		return nil, err
	}
	in.Mode = resolvePreviewMode(current, in)
	return s.provider.PreviewSubscriptionChange(ctx, cust.ProviderCustomerID, cust.ProviderSubscriptionID, in)
}

func resolvePreviewMode(current *domain.SubscriptionSnapshot, in domain.SubscriptionPreviewInput) domain.SubscriptionChangeMode {
	return resolveChangeMode(current, domain.SubscriptionChangeInput{
		Plan:     in.Plan,
		Interval: in.Interval,
		Mode:     in.Mode,
	})
}

func resolveChangeMode(current *domain.SubscriptionSnapshot, in domain.SubscriptionChangeInput) domain.SubscriptionChangeMode {
	if in.Mode.Valid() {
		return in.Mode
	}
	if current == nil {
		return domain.ChangeModeImmediateProrated
	}

	// Monthly -> yearly should apply the annual commitment immediately.
	if current.Interval == domain.IntervalMonthly && in.Interval == domain.IntervalYearly {
		return domain.ChangeModeImmediateResetCycle
	}
	// Yearly -> monthly should not create partial-year refunds by default.
	if current.Interval == domain.IntervalYearly && in.Interval == domain.IntervalMonthly {
		return domain.ChangeModePeriodEnd
	}
	// Same interval: upgrades are immediate, downgrades wait until period end.
	if rankPlan(in.Plan) > rankPlan(current.Plan) {
		return domain.ChangeModeImmediateProrated
	}
	if rankPlan(in.Plan) < rankPlan(current.Plan) {
		return domain.ChangeModePeriodEnd
	}
	return domain.ChangeModeImmediateProrated
}

func rankPlan(plan domain.PlanType) int {
	switch plan {
	case domain.PlanStarter:
		return 1
	case domain.PlanPro:
		return 2
	case domain.PlanPremium:
		return 3
	default:
		return 0
	}
}
