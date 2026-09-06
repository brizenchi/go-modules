package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/brizenchi/go-modules/foundation/resilience"
	authevent "github.com/brizenchi/go-modules/modules/auth/event"
	billingdomain "github.com/brizenchi/go-modules/modules/billing/domain"
	billingevent "github.com/brizenchi/go-modules/modules/billing/event"
	referraldomain "github.com/brizenchi/go-modules/modules/referral/domain"
	referralevent "github.com/brizenchi/go-modules/modules/referral/event"
	"github.com/brizenchi/quickstart-template/internal/hostapi"
	"github.com/brizenchi/quickstart-template/internal/platform"
)

// subscribeModuleEvents is the template-owned composition point.
// Delete, reorder or replace any subscription in a copied SaaS.
func subscribeModuleEvents(deps hostapi.Deps, cfg AppConfig) {
	modules := deps.Modules
	if modules.Auth != nil {
		modules.Auth.Subscribe(authevent.KindUserSignedUp, func(ctx context.Context, envelope authevent.Envelope) error {
			payload, ok := envelope.Payload.(authevent.UserSignedUp)
			if !ok {
				return fmt.Errorf("unexpected user signup payload %T", envelope.Payload)
			}
			// Run both independent effects even if one fails. The auth bus logs the
			// aggregate without turning a completed identity login into an outage.
			return errors.Join(
				attributeReferralFromLogin(ctx, deps, envelope.UserID),
				onUserSignedUp(ctx, deps, envelope, payload),
			)
		})
		modules.Auth.Subscribe(authevent.KindUserLoggedIn, func(ctx context.Context, envelope authevent.Envelope) error {
			payload, ok := envelope.Payload.(authevent.UserLoggedIn)
			if !ok {
				return fmt.Errorf("unexpected user login payload %T", envelope.Payload)
			}
			// UserLoggedIn also fires on the first login. Retrying attribution only
			// for that request closes the gap where UserSignedUp persisted the
			// user/referral but a transient listener failed, without letting an
			// existing account attach an invite code on a later login.
			if payload.Identity.IsNew {
				return errors.Join(
					attributeReferralFromLogin(ctx, deps, envelope.UserID),
					onUserLoggedIn(ctx, deps, envelope, payload),
				)
			}
			return onUserLoggedIn(ctx, deps, envelope, payload)
		})
	}

	if modules.Billing != nil {
		modules.Billing.Subscribe(billingevent.KindSubscriptionActivated, func(ctx context.Context, envelope billingevent.Envelope) error {
			payload, ok := envelope.Payload.(billingevent.SubscriptionActivated)
			if !ok {
				return fmt.Errorf("unexpected subscription activated payload %T", envelope.Payload)
			}
			// A Checkout Session can complete while a free trial is still unpaid.
			// Referral rewards qualify only once the subscription is actually paid.
			if payload.Snapshot.Status == billingdomain.StatusActive {
				if err := activateReferral(ctx, deps, envelope.UserID, cfg.Referral.ActivationReward); err != nil {
					return err
				}
			}
			return onSubscriptionActivated(ctx, deps, envelope, payload)
		})
		modules.Billing.Subscribe(billingevent.KindTrialConverted, func(ctx context.Context, envelope billingevent.Envelope) error {
			payload, ok := envelope.Payload.(billingevent.TrialConverted)
			if !ok {
				return fmt.Errorf("unexpected trial converted payload %T", envelope.Payload)
			}
			if err := activateReferral(ctx, deps, envelope.UserID, cfg.Referral.ActivationReward); err != nil {
				return err
			}
			return onTrialConverted(ctx, deps, envelope, payload)
		})
		modules.Billing.Subscribe(billingevent.KindSubscriptionRenewed, func(ctx context.Context, envelope billingevent.Envelope) error {
			payload, ok := envelope.Payload.(billingevent.SubscriptionRenewed)
			if !ok {
				return fmt.Errorf("unexpected subscription renewed payload %T", envelope.Payload)
			}
			// A paid invoice may still have amount_paid=0 (for example, a 100%
			// coupon). Only a positive charge proves the referral's paid
			// qualification. Later renewals are absorbed by the referral and
			// credit-ledger idempotency guards.
			var referralErr error
			if payload.AmountPaid > 0 {
				referralErr = activateReferral(ctx, deps, envelope.UserID, cfg.Referral.ActivationReward)
			}
			return errors.Join(referralErr, onSubscriptionRenewed(ctx, deps, envelope, payload))
		})
		modules.Billing.Subscribe(billingevent.KindSubscriptionUpdated, billingListener(deps, onSubscriptionUpdated))
		modules.Billing.Subscribe(billingevent.KindSubscriptionReactivated, billingListener(deps, onSubscriptionReactivated))
		modules.Billing.Subscribe(billingevent.KindSubscriptionCanceling, billingListener(deps, onSubscriptionCanceling))
		modules.Billing.Subscribe(billingevent.KindSubscriptionCanceled, billingListener(deps, onSubscriptionCanceled))
		modules.Billing.Subscribe(billingevent.KindPaymentFailed, billingListener(deps, onPaymentFailed))
		modules.Billing.Subscribe(billingevent.KindCreditsPurchased, func(ctx context.Context, envelope billingevent.Envelope) error {
			payload, ok := envelope.Payload.(billingevent.CreditsPurchased)
			if !ok {
				return fmt.Errorf("unexpected credits purchased payload %T", envelope.Payload)
			}
			return onCreditsPurchased(ctx, deps, envelope, payload)
		})
	}

	if modules.Referral != nil {
		modules.Referral.Subscribe(referralevent.KindReferralRegistered, func(ctx context.Context, envelope referralevent.Envelope) error {
			payload, ok := envelope.Payload.(referralevent.ReferralRegistered)
			if !ok {
				return fmt.Errorf("unexpected referral registered payload %T", envelope.Payload)
			}
			return onReferralRegistered(ctx, deps, envelope, payload)
		})
		modules.Referral.Subscribe(referralevent.KindReferralActivated, func(ctx context.Context, envelope referralevent.Envelope) error {
			payload, ok := envelope.Payload.(referralevent.ReferralActivated)
			if !ok {
				return fmt.Errorf("unexpected referral activated payload %T", envelope.Payload)
			}
			return onReferralActivated(ctx, deps, envelope, payload)
		})
	}
}

func attributeReferralFromLogin(ctx context.Context, deps hostapi.Deps, refereeID string) error {
	if deps.Modules == nil || deps.Modules.Referral == nil {
		return nil
	}
	code := platform.ReferralCodeFromContext(ctx)
	if code == "" {
		return nil
	}

	policy := resilience.Constant(3, 20*time.Millisecond)
	policy.Retryable = func(err error) bool {
		return !errors.Is(err, referraldomain.ErrInvalidUser) &&
			!errors.Is(err, referraldomain.ErrInvalidCode) &&
			!errors.Is(err, referraldomain.ErrSelfReferral)
	}
	err := resilience.Do(ctx, func(ctx context.Context) error {
		_, err := deps.Modules.Referral.Attribute.AttributeReferral(ctx, refereeID, code)
		if errors.Is(err, referraldomain.ErrAlreadyAttributed) {
			return nil
		}
		return err
	}, policy)
	if err != nil {
		return fmt.Errorf("attribute referral: %w", err)
	}
	return nil
}

func activateReferral(ctx context.Context, deps hostapi.Deps, refereeID string, reward int) error {
	if deps.Modules.Referral == nil {
		return nil
	}
	_, err := deps.Modules.Referral.Attribute.ActivateReferral(ctx, refereeID, reward)
	if err == nil || errors.Is(err, referraldomain.ErrNotFound) || errors.Is(err, referraldomain.ErrAlreadyActivated) || errors.Is(err, referraldomain.ErrExpired) {
		return nil
	}
	return fmt.Errorf("activate referral: %w", err)
}

type billingPayload interface {
	billingevent.SubscriptionRenewed |
		billingevent.SubscriptionUpdated |
		billingevent.SubscriptionReactivated |
		billingevent.SubscriptionCanceling |
		billingevent.SubscriptionCanceled |
		billingevent.PaymentFailed
}

func billingListener[T billingPayload](deps hostapi.Deps, callback func(context.Context, hostapi.Deps, billingevent.Envelope, T) error) func(context.Context, billingevent.Envelope) error {
	return func(ctx context.Context, envelope billingevent.Envelope) error {
		payload, ok := envelope.Payload.(T)
		if !ok {
			return fmt.Errorf("unexpected billing payload %T", envelope.Payload)
		}
		return callback(ctx, deps, envelope, payload)
	}
}
