package bootstrap

import (
	"context"
	"errors"
	"fmt"

	authevent "github.com/brizenchi/go-modules/modules/auth/event"
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
			if modules.Referral != nil {
				if code := platform.ReferralCodeFromContext(ctx); code != "" {
					if _, err := modules.Referral.Attribute.AttributeReferral(ctx, envelope.UserID, code); err != nil &&
						!errors.Is(err, referraldomain.ErrAlreadyAttributed) {
						return fmt.Errorf("attribute referral: %w", err)
					}
				}
			}
			return onUserSignedUp(ctx, deps, envelope, payload)
		})
		modules.Auth.Subscribe(authevent.KindUserLoggedIn, func(ctx context.Context, envelope authevent.Envelope) error {
			payload, ok := envelope.Payload.(authevent.UserLoggedIn)
			if !ok {
				return fmt.Errorf("unexpected user login payload %T", envelope.Payload)
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
			if modules.Referral != nil {
				_, err := modules.Referral.Attribute.ActivateReferral(ctx, envelope.UserID, cfg.Referral.ActivationReward)
				if err != nil && !errors.Is(err, referraldomain.ErrNotFound) && !errors.Is(err, referraldomain.ErrAlreadyActivated) {
					return fmt.Errorf("activate referral: %w", err)
				}
			}
			return onSubscriptionActivated(ctx, deps, envelope, payload)
		})
		modules.Billing.Subscribe(billingevent.KindSubscriptionRenewed, billingListener(deps, onSubscriptionRenewed))
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
