package domain

import (
	"encoding/json"
	"time"
)

// BillingSubscription stores the current provider-derived commercial
// snapshot for a user and provider.
//
// This is a read model for current state, not an append-only audit
// history. Webhook history remains in BillingEvent.
type BillingSubscription struct {
	ID                     uint            `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID                 string          `json:"user_id" gorm:"type:varchar(36);not null;uniqueIndex:uniq_billing_subscription_user_provider;index"`
	Provider               string          `json:"provider" gorm:"type:varchar(32);not null;uniqueIndex:uniq_billing_subscription_user_provider;index"`
	ProviderCustomerID     string          `json:"provider_customer_id" gorm:"type:varchar(255);index"`
	ProviderSubscriptionID string          `json:"provider_subscription_id" gorm:"type:varchar(255);index"`
	ProviderPriceID        string          `json:"provider_price_id" gorm:"type:varchar(255)"`
	ProviderProductID      string          `json:"provider_product_id" gorm:"type:varchar(255)"`
	ProductType            string          `json:"product_type" gorm:"type:varchar(32)"`
	Plan                   string          `json:"plan" gorm:"type:varchar(32);index"`
	BillingInterval        string          `json:"billing_interval" gorm:"type:varchar(32)"`
	Status                 string          `json:"status" gorm:"type:varchar(64);index"`
	CancelAtPeriodEnd      bool            `json:"cancel_at_period_end" gorm:"not null;default:false"`
	PeriodStart            *time.Time      `json:"period_start,omitempty"`
	PeriodEnd              *time.Time      `json:"period_end,omitempty"`
	CancelEffectiveAt      *time.Time      `json:"cancel_effective_at,omitempty"`
	RawSnapshotJSON        json.RawMessage `json:"raw_snapshot_json,omitempty" gorm:"type:jsonb"`
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

func (BillingSubscription) TableName() string { return "billing_subscriptions" }
