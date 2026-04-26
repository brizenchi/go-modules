// Package referral_glue wires the referral module + auto-migrates schema.
package referral_glue

import (
	"context"
	"log/slog"

	billingevent "github.com/brizenchi/go-modules/billing/event"
	"github.com/brizenchi/go-modules/referral"
	"github.com/brizenchi/go-modules/referral/adapter/codegen"
	"github.com/brizenchi/go-modules/referral/adapter/eventbus"
	"github.com/brizenchi/go-modules/referral/adapter/gormrepo"
	"github.com/brizenchi/go-modules/referral/event"

	"github.com/spf13/viper"
	"gorm.io/gorm"
)

// Init wires the referral module + auto-migrates its schema.
func Init(db *gorm.DB) *referral.Module {
	if err := db.AutoMigrate(gormrepo.AutoMigrateModels()...); err != nil {
		slog.Error("referral_glue: auto-migrate failed", "error", err)
	}

	prefix := viper.GetString("referral.prefix")
	if prefix == "" {
		prefix = "INV"
	}
	gen := codegen.NewDeterministic(prefix, 8)

	mod := referral.New(referral.Deps{
		Codes:     gormrepo.NewCodeRepo(db),
		Referrals: gormrepo.NewReferralRepo(db),
		Generator: gen,
		Bus:       eventbus.NewInProc(),
		BaseLink:  viper.GetString("referral.base_link"),
	})

	mod.Subscribe(event.KindReferralActivated, OnReferralActivated)
	return mod
}

// OnSubscriptionActivated bridges billing → referral activation.
//
// Wire in cmd/main.go:
//
//	billingMod.Subscribe(billingevent.KindSubscriptionActivated,
//	    referral_glue.OnSubscriptionActivated)
//
// Caller has access to the referral module via this package's mod var
// (set up Init to capture it if you need to call Attribute.ActivateReferral
// from here — the snippet below does that via package-level state).
var module *referral.Module

func OnSubscriptionActivated(ctx context.Context, env billingevent.Envelope) error {
	if module == nil || env.UserID == "" {
		return nil
	}
	reward := viper.GetInt("referral.activation_reward")
	_, err := module.Attribute.ActivateReferral(ctx, env.UserID, reward)
	if err != nil {
		// Most errors are "user wasn't referred" or "already activated"
		// — both are fine; just log at debug.
		slog.Debug("referral: activation skipped", "user_id", env.UserID, "error", err)
	}
	return nil
}

// OnReferralActivated is the host's chance to actually pay out the reward.
func OnReferralActivated(ctx context.Context, env event.Envelope) error {
	p, _ := env.Payload.(event.ReferralActivated)
	slog.Info("referral: activated",
		"referrer_id", p.Referral.ReferrerID,
		"referee_id", p.Referral.RefereeID,
		"reward_credits", p.Referral.RewardCredits,
	)
	// TODO: credit p.Referral.ReferrerID's wallet by p.Referral.RewardCredits.
	return nil
}
