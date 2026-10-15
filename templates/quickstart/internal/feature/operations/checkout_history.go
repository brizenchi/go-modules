package operations

import (
	"context"
	"encoding/json"

	billingdomain "github.com/brizenchi/go-modules/modules/billing/domain"
	"github.com/brizenchi/quickstart-template/internal/checkoutlog"
	"gorm.io/gorm"
)

// Only the administrator projection includes intent. The paid reporting and
// customer purchase projection continue to use verified payment events alone.
func mergeCheckoutHistory(ctx context.Context, db *gorm.DB, payments map[string]OrderSummary) error {
	sessions := map[string]string{}
	for key, row := range payments {
		if row.CheckoutSessionID != "" {
			sessions[row.Provider+":"+row.CheckoutSessionID] = key
		}
	}
	lastID := uint(0)
	for {
		batch := []checkoutlog.Record{}
		if err := db.WithContext(ctx).Where("id > ?", lastID).Order("id ASC").Limit(500).Find(&batch).Error; err != nil {
			return err
		}
		if len(batch) == 0 {
			break
		}
		for _, record := range batch {
			lastID = record.ID
			sessionKey := record.Provider + ":" + record.SessionID
			key := sessionKey
			if matched, ok := sessions[sessionKey]; ok {
				key = matched
			}
			row, exists := payments[key]
			if !exists {
				row = OrderSummary{ID: sessionKey, UserID: record.UserID, Provider: record.Provider, CheckoutSessionID: record.SessionID, RecordType: "checkout", EventType: "checkout.created", Status: "unpaid", Livemode: record.Livemode, Currency: record.Currency, sortID: record.ID}
			}
			row.Plan = record.Plan
			row.BillingInterval = record.BillingInterval
			row.Quantity = record.Quantity
			row.ExpiresAt = record.ExpiresAt
			row.CreatedAt = record.CreatedAt
			if row.ProductType == "" {
				row.ProductType = record.ProductType
			}
			if row.Currency == "" {
				row.Currency = record.Currency
			}
			if row.Livemode == nil {
				row.Livemode = record.Livemode
			}
			if row.AmountDue == nil {
				row.AmountDue = record.AmountTotal
			}
			payments[key] = row
			sessions[sessionKey] = key
		}
	}
	// Expiry is taken only from persisted provider events, never guessed from a
	// local clock: a completed checkout may still be awaiting its payment webhook.
	lastID = 0
	for {
		events := []billingdomain.BillingEvent{}
		if err := db.WithContext(ctx).Where("id > ? AND event_type = ?", lastID, "checkout.session.expired").Order("id ASC").Limit(500).Find(&events).Error; err != nil {
			return err
		}
		if len(events) == 0 {
			break
		}
		for _, event := range events {
			lastID = event.ID
			var envelope paymentEnvelope
			var object paymentObject
			if json.Unmarshal(event.Payload, &envelope) != nil || json.Unmarshal(envelope.Data.Object, &object) != nil || object.ID == "" {
				continue
			}
			key, ok := sessions[event.Provider+":"+object.ID]
			if !ok {
				continue
			}
			row := payments[key]
			if row.Status == "unpaid" {
				row.Status = "expired"
				payments[key] = row
			}
		}
	}
	return nil
}
