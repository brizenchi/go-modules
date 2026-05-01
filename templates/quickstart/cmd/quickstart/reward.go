package main

import (
	"context"

	"github.com/brizenchi/go-modules/modules/user/adapter/gormrepo"
	"github.com/brizenchi/go-modules/stacks/saascore"
	"gorm.io/gorm"
)

func referralReward(ctx context.Context, db *gorm.DB, event saascore.ReferralActivatedEvent) error {
	return gormrepo.New(db).AddCredits(ctx, event.Referral.ReferrerID, event.Referral.RewardCredits)
}
