package domain

import "time"

// BillingCustomer stores the provider-side customer identity for a user.
//
// One row per (user, provider) keeps the shared user schema free of
// provider-specific linkage fields while still allowing fast lookups by
// provider customer ID.
type BillingCustomer struct {
	ID                 uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID             string `json:"user_id" gorm:"type:varchar(36);not null;uniqueIndex:uniq_billing_customer_user_provider;index"`
	Provider           string `json:"provider" gorm:"type:varchar(32);not null;uniqueIndex:uniq_billing_customer_user_provider;uniqueIndex:uniq_billing_customer_provider_customer;index"`
	ProviderCustomerID string `json:"provider_customer_id" gorm:"type:varchar(255);not null;uniqueIndex:uniq_billing_customer_provider_customer;index"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (BillingCustomer) TableName() string { return "billing_customers" }
