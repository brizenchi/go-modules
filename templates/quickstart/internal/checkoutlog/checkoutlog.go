// Package checkoutlog keeps host-owned checkout intent history. Provider events
// remain authoritative for payment; no checkout URL or credential is persisted.
package checkoutlog

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	billingapp "github.com/brizenchi/go-modules/modules/billing/app"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Record struct {
	ID              uint   `gorm:"primaryKey"`
	UserID          string `gorm:"type:varchar(36);not null;index"`
	Provider        string `gorm:"type:varchar(32);not null;uniqueIndex:uniq_checkout_history_session"`
	SessionID       string `gorm:"type:varchar(255);not null;uniqueIndex:uniq_checkout_history_session"`
	ProductType     string `gorm:"type:varchar(32);not null"`
	Plan            string `gorm:"type:varchar(32);not null"`
	BillingInterval string `gorm:"type:varchar(16);not null"`
	Quantity        int64  `gorm:"not null"`
	AmountTotal     *int64
	Currency        string `gorm:"type:varchar(8);not null"`
	Livemode        *bool
	ExpiresAt       *time.Time
	CreatedAt       time.Time
}

func (Record) TableName() string { return "checkout_history" }

func Save(ctx context.Context, db *gorm.DB, provider string, in billingapp.CheckoutInput, result billingapp.CheckoutResult) error {
	if db == nil || strings.TrimSpace(in.UserID) == "" || strings.TrimSpace(result.SessionID) == "" || strings.TrimSpace(provider) == "" {
		return fmt.Errorf("checkout history: missing identity")
	}
	created := time.Now().UTC()
	if result.CreatedAt != nil {
		created = result.CreatedAt.UTC()
	}
	row := Record{UserID: in.UserID, Provider: provider, SessionID: result.SessionID, ProductType: string(in.ProductType), Plan: string(in.Plan), BillingInterval: string(in.Interval), Quantity: in.Quantity, AmountTotal: result.AmountTotal, Currency: strings.ToLower(result.Currency), Livemode: result.Livemode, ExpiresAt: result.ExpiresAt, CreatedAt: created}
	// Reusing a session is not another order and cannot rewrite its original owner,
	// chosen offer, amount or creation time.
	return db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "provider"}, {Name: "session_id"}}, DoNothing: true}).Create(&row).Error
}
func Observer(db *gorm.DB, provider string) billingapp.CheckoutObserver {
	return func(ctx context.Context, in billingapp.CheckoutInput, result billingapp.CheckoutResult) {
		if err := Save(ctx, db, provider, in, result); err != nil {
			slog.ErrorContext(ctx, "checkout history recording failed", "error", err)
		}
	}
}
