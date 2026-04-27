package app

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/brizenchi/go-modules/referral/domain"
	"github.com/brizenchi/go-modules/referral/event"
	"github.com/brizenchi/go-modules/referral/port"
)

// AttributeService creates and activates referrer→referee links.
//
// AttributeReferral is typically called from an auth UserSignedUp
// listener: when a new user supplies a referral code at signup, this
// stores the link as Pending.
//
// ActivateReferral is typically called from a billing
// SubscriptionActivated listener: when the referee makes their first
// qualifying payment, the link transitions to Activated and a
// ReferralActivated event fires for downstream listeners (granting
// rewards via pkg/billing's credit ledger, sending notifications).
type AttributeService struct {
	codes         *CodeService
	referrals     port.ReferralRepository
	bus           port.EventBus
	defaultExpiry time.Duration // 0 = no deadline
}

// AttributeDeps gathers the dependencies AttributeService needs.
type AttributeDeps struct {
	Codes            *CodeService
	Referrals        port.ReferralRepository
	Bus              port.EventBus
	ActivationWindow time.Duration // 0 = no activation deadline
}

func NewAttributeService(d AttributeDeps) *AttributeService {
	return &AttributeService{
		codes:         d.Codes,
		referrals:     d.Referrals,
		bus:           d.Bus,
		defaultExpiry: d.ActivationWindow,
	}
}

// AttributeReferral records a referrer→referee link.
//
// Resolves codeValue → referrerID, validates the referee != referrer,
// stores the link in Pending state, fires ReferralRegistered.
//
// Errors:
//   - ErrInvalidCode      — code does not exist
//   - ErrSelfReferral     — referrer and referee are the same user
//   - ErrAlreadyAttributed — referee already has a referrer
func (s *AttributeService) AttributeReferral(ctx context.Context, refereeID, codeValue string) (*domain.Referral, error) {
	refereeID = strings.TrimSpace(refereeID)
	if refereeID == "" {
		return nil, domain.ErrInvalidUser
	}
	code, err := s.codes.Resolve(ctx, codeValue)
	if err != nil {
		return nil, err
	}
	if code.UserID == refereeID {
		return nil, domain.ErrSelfReferral
	}

	ref := domain.Referral{
		Code:       code.Value,
		ReferrerID: code.UserID,
		RefereeID:  refereeID,
		Status:     domain.StatusPending,
	}
	if s.defaultExpiry > 0 {
		expires := time.Now().UTC().Add(s.defaultExpiry)
		ref.ExpiresAt = &expires
	}
	stored, err := s.referrals.Create(ctx, ref)
	if err != nil {
		return nil, err
	}
	if s.bus != nil {
		s.bus.Publish(ctx, event.Envelope{
			Kind:       event.KindReferralRegistered,
			OccurredAt: time.Now().UTC(),
			Payload:    event.ReferralRegistered{Referral: *stored},
		})
	}
	return stored, nil
}

// ActivateReferral transitions a pending referral to activated and
// fires ReferralActivated. Idempotent: subsequent calls return
// ErrAlreadyActivated.
func (s *AttributeService) ActivateReferral(ctx context.Context, refereeID string, rewardCredits int) (*domain.Referral, error) {
	refereeID = strings.TrimSpace(refereeID)
	if refereeID == "" {
		return nil, domain.ErrInvalidUser
	}
	r, err := s.referrals.Activate(ctx, refereeID, rewardCredits)
	if err != nil {
		if errors.Is(err, domain.ErrAlreadyActivated) || errors.Is(err, domain.ErrNotFound) {
			return nil, err
		}
		return nil, err
	}
	if s.bus != nil {
		s.bus.Publish(ctx, event.Envelope{
			Kind:       event.KindReferralActivated,
			OccurredAt: time.Now().UTC(),
			Payload:    event.ReferralActivated{Referral: *r},
		})
	}
	return r, nil
}
