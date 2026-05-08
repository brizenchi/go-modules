package repo

import (
	"github.com/brizenchi/go-modules/modules/billing/domain"
	"gorm.io/gorm"
)

// AutoMigrateModels returns the standard billing persistence models.
func AutoMigrateModels() []any {
	return []any{
		&domain.BillingEvent{},
		&domain.BillingCustomer{},
		&domain.BillingSubscription{},
	}
}

// AutoMigrate creates or updates the standard billing tables.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(AutoMigrateModels()...)
}
