package app

import (
	"context"
	"strings"
	"time"

	"github.com/brizenchi/go-modules/billing/domain"
	"github.com/brizenchi/go-modules/billing/event"
	"github.com/brizenchi/go-modules/billing/port"
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
