package billing

import (
	"encoding/json"
	"time"
)

type StripeTopUpEvent struct {
	ID              uint            `gorm:"primaryKey;autoIncrement"`
	ProviderEventID string          `gorm:"type:varchar(255);uniqueIndex;not null"`
	PaymentIntentID string          `gorm:"type:varchar(255);uniqueIndex;not null"`
	UserID          string          `gorm:"type:varchar(36);index"`
	CustomerID      string          `gorm:"type:varchar(255);index"`
	AmountCents     int64           `gorm:"not null"`
	Credits         int64           `gorm:"not null"`
	Payload         json.RawMessage `gorm:"type:jsonb;not null"`
	Processed       bool            `gorm:"not null;default:false;index"`
	ProcessedAt     *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (StripeTopUpEvent) TableName() string { return "stripe_topup_events" }
