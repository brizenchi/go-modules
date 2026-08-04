package bootstrap

import (
	"context"

	"github.com/brizenchi/go-modules/stacks/saascore"
	"gorm.io/gorm"
)

func buildHostHooks(db *gorm.DB) saascore.HostHooks {
	return saascore.HostHooks{
		OnUserSignedUp: func(ctx context.Context, event saascore.UserSignedUpEvent) error {
			return onUserSignedUp(ctx, db, event)
		},
		OnUserLoggedIn: func(ctx context.Context, event saascore.UserLoggedInEvent) error {
			return onUserLoggedIn(ctx, db, event)
		},
		OnSubscriptionActivated: func(ctx context.Context, event saascore.SubscriptionEvent) error {
			return onSubscriptionActivated(ctx, db, event)
		},
		OnSubscriptionRenewed: func(ctx context.Context, event saascore.SubscriptionEvent) error {
			return onSubscriptionRenewed(ctx, db, event)
		},
		OnSubscriptionUpdated: func(ctx context.Context, event saascore.SubscriptionEvent) error {
			return onSubscriptionUpdated(ctx, db, event)
		},
		OnSubscriptionReactivated: func(ctx context.Context, event saascore.SubscriptionEvent) error {
			return onSubscriptionReactivated(ctx, db, event)
		},
		OnSubscriptionCanceling: func(ctx context.Context, event saascore.SubscriptionCancelingEvent) error {
			return onSubscriptionCanceling(ctx, db, event)
		},
		OnSubscriptionCanceled: func(ctx context.Context, event saascore.SubscriptionCanceledEvent) error {
			return onSubscriptionCanceled(ctx, db, event)
		},
		OnPaymentFailed: func(ctx context.Context, event saascore.PaymentFailedEvent) error {
			return onPaymentFailed(ctx, db, event)
		},
		OnCreditsPurchased: func(ctx context.Context, event saascore.CreditsPurchasedEvent) error {
			return onCreditsPurchased(ctx, db, event)
		},
		OnReferralRegistered: func(ctx context.Context, event saascore.ReferralRegisteredEvent) error {
			return onReferralRegistered(ctx, db, event)
		},
		OnReferralActivated: func(ctx context.Context, event saascore.ReferralActivatedEvent) error {
			return onReferralActivated(ctx, db, event)
		},
	}
}

func onUserSignedUp(_ context.Context, _ *gorm.DB, _ saascore.UserSignedUpEvent) error {
	// Fill in host-specific signup side effects here.
	return nil
}

func onUserLoggedIn(_ context.Context, _ *gorm.DB, _ saascore.UserLoggedInEvent) error {
	// Fill in host-specific login side effects here.
	return nil
}

func onSubscriptionActivated(_ context.Context, _ *gorm.DB, _ saascore.SubscriptionEvent) error {
	// Fill in host-specific activation side effects here.
	return nil
}

func onSubscriptionRenewed(_ context.Context, _ *gorm.DB, _ saascore.SubscriptionEvent) error {
	// Fill in host-specific renewal side effects here.
	return nil
}

func onSubscriptionUpdated(_ context.Context, _ *gorm.DB, _ saascore.SubscriptionEvent) error {
	// Fill in host-specific plan-change side effects here.
	return nil
}

func onSubscriptionReactivated(_ context.Context, _ *gorm.DB, _ saascore.SubscriptionEvent) error {
	// Fill in host-specific reactivation side effects here.
	return nil
}

func onSubscriptionCanceling(_ context.Context, _ *gorm.DB, _ saascore.SubscriptionCancelingEvent) error {
	// Fill in host-specific pre-cancel side effects here.
	return nil
}

func onSubscriptionCanceled(_ context.Context, _ *gorm.DB, _ saascore.SubscriptionCanceledEvent) error {
	// Fill in host-specific cancel-complete side effects here.
	return nil
}

func onPaymentFailed(_ context.Context, _ *gorm.DB, _ saascore.PaymentFailedEvent) error {
	// Fill in host-specific payment-failure side effects here.
	return nil
}

func onCreditsPurchased(_ context.Context, _ *gorm.DB, _ saascore.CreditsPurchasedEvent) error {
	// Fill in host-specific credits-purchase side effects here.
	// Shared saascore already increments user credits before this hook runs.
	return nil
}

func onReferralRegistered(_ context.Context, _ *gorm.DB, _ saascore.ReferralRegisteredEvent) error {
	// Fill in host-specific referral registration side effects here.
	return nil
}

func onReferralActivated(ctx context.Context, db *gorm.DB, event saascore.ReferralActivatedEvent) error {
	// Replace or extend this default implementation with your own payout flow.
	return applyReferralReward(ctx, db, event)
}
