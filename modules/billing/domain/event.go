package domain

import (
	"encoding/json"
	"time"
)

// BillingEvent is the persistent record of a provider webhook delivery.
// Used for auditing and idempotency. One row per (provider, provider_event_id).
type BillingEvent struct {
	ID              uint            `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID          string          `json:"user_id,omitempty" gorm:"type:varchar(36);index"`
	Provider        string          `json:"provider" gorm:"type:varchar(32);index;not null"`
	ProviderEventID string          `json:"provider_event_id" gorm:"type:varchar(255);uniqueIndex:uniq_provider_event;not null;column:stripe_event_id"`
	EventType       string          `json:"event_type" gorm:"type:varchar(128);index;not null"`
	Payload         json.RawMessage `json:"payload" gorm:"type:jsonb;not null"`
	Processed       bool            `json:"processed" gorm:"not null;default:false;index"`
	ProcessedAt     *time.Time      `json:"processed_at,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

func (BillingEvent) TableName() string { return "billing_events" }
