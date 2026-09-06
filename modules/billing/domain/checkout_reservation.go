package domain

import "time"

// BillingCheckoutReservation serializes entitlement-bearing checkouts per
// user/provider until the provider-side Checkout Session expires.
type BillingCheckoutReservation struct {
	ID                     uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID                 string    `json:"user_id" gorm:"type:varchar(36);uniqueIndex:uniq_billing_checkout_reservation;not null"`
	Provider               string    `json:"provider" gorm:"type:varchar(32);uniqueIndex:uniq_billing_checkout_reservation;not null"`
	ProviderCustomerID     string    `json:"provider_customer_id" gorm:"type:varchar(255);not null;index"`
	ReservationID          string    `json:"reservation_id" gorm:"type:varchar(64);not null;index"`
	IntentKey              string    `json:"intent_key" gorm:"type:char(64);not null"`
	SessionID              string    `json:"session_id,omitempty" gorm:"type:varchar(255)"`
	ProviderSubscriptionID string    `json:"provider_subscription_id,omitempty" gorm:"type:varchar(255);index"`
	CheckoutURL            string    `json:"checkout_url,omitempty" gorm:"type:text"`
	ExpiresAt              time.Time `json:"expires_at" gorm:"not null;index"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

func (BillingCheckoutReservation) TableName() string { return "billing_checkout_reservations" }
