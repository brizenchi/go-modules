package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/brizenchi/go-modules/modules/referral/domain"
	"github.com/brizenchi/go-modules/modules/referral/event"
	"github.com/brizenchi/go-modules/modules/referral/port"
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
// rewards via modules/billing's credit ledger, sending notifications).
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
	alreadyAttributed := false
	if err != nil {
		if !errors.Is(err, domain.ErrAlreadyAttributed) {
			return nil, err
		}
		alreadyAttributed = true
		// The link can commit before a listener fails. Replaying only the same
		// attribution lets an upstream login retry finish downstream work while
		// preserving the one-referrer-per-user contract.
		stored, err = s.referrals.FindByReferee(ctx, refereeID)
		if err != nil {
			return nil, err
		}
		if stored.Code != code.Value || stored.ReferrerID != code.UserID {
			return stored, domain.ErrAlreadyAttributed
		}
	}
	if s.bus != nil {
		if err := s.bus.Publish(ctx, event.Envelope{
			Kind:       event.KindReferralRegistered,
			OccurredAt: referralOccurredAt(stored),
			Payload:    event.ReferralRegistered{Referral: *stored},
		}); err != nil {
			return stored, fmt.Errorf("referral: publish registered event: %w", err)
		}
	}
	if alreadyAttributed {
		return stored, domain.ErrAlreadyAttributed
	}
	return stored, nil
}

func referralOccurredAt(referral *domain.Referral) time.Time {
	if referral != nil && !referral.CreatedAt.IsZero() {
		return referral.CreatedAt.UTC()
	}
	return time.Now().UTC()
}

// ActivateReferral transitions a pending referral to activated and
// fires ReferralActivated. Subsequent calls still return ErrAlreadyActivated,
// but first reload and republish the stored event. That lets an upstream
// webhook retry a listener failure without changing the public idempotency
// contract; listeners must make repeated delivery safe.
func (s *AttributeService) ActivateReferral(ctx context.Context, refereeID string, rewardCredits int) (*domain.Referral, error) {
	refereeID = strings.TrimSpace(refereeID)
	if refereeID == "" {
		return nil, domain.ErrInvalidUser
	}
	r, err := s.referrals.Activate(ctx, refereeID, rewardCredits)
	alreadyActivated := false
	if err != nil {
		if !errors.Is(err, domain.ErrAlreadyActivated) {
			return nil, err
		}
		alreadyActivated = true
		// The state transition can commit before a listener fails. Reload and
		// replay the stored reward so the upstream webhook retry can complete.
		r, err = s.referrals.FindByReferee(ctx, refereeID)
		if err != nil {
			return nil, err
		}
		if r.Status != domain.StatusActivated {
			return nil, domain.ErrAlreadyActivated
		}
	}
	if s.bus != nil {
		if err := s.bus.Publish(ctx, event.Envelope{
			Kind:       event.KindReferralActivated,
			OccurredAt: time.Now().UTC(),
			Payload:    event.ReferralActivated{Referral: *r},
		}); err != nil {
			return r, fmt.Errorf("referral: publish activated event: %w", err)
		}
	}
	if alreadyActivated {
		return r, domain.ErrAlreadyActivated
	}
	return r, nil
}
