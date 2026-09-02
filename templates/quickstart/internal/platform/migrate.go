package platform

import (
	"fmt"

	authgormstore "github.com/brizenchi/go-modules/modules/auth/adapter/gormstore"
	billingrepo "github.com/brizenchi/go-modules/modules/billing/adapter/repo"
	referralgormrepo "github.com/brizenchi/go-modules/modules/referral/adapter/gormrepo"
	"github.com/brizenchi/quickstart-template/internal/user"
	"gorm.io/gorm"
)

// Migrate migrates the host user schema and only the enabled modules' tables.
func Migrate(db *gorm.DB, cfg Config) error {
	if db == nil {
		return fmt.Errorf("platform: db required")
	}
	cfg = cfg.withDefaults()
	if err := user.AutoMigrate(db); err != nil {
		return fmt.Errorf("migrate host users: %w", err)
	}
	if cfg.AuthEnabled() {
		if err := authgormstore.AutoMigrate(db); err != nil {
			return fmt.Errorf("migrate auth: %w", err)
		}
	}
	if cfg.BillingEnabled() {
		if err := billingrepo.AutoMigrate(db); err != nil {
			return fmt.Errorf("migrate billing: %w", err)
		}
	}
	if cfg.ReferralEnabled() {
		if err := db.AutoMigrate(referralgormrepo.AutoMigrateModels()...); err != nil {
			return fmt.Errorf("migrate referral: %w", err)
		}
	}
	return nil
}
