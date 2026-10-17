package creditpurchase

import (
	"context"
	"encoding/json"

	billingdomain "github.com/brizenchi/go-modules/modules/billing/domain"
	"github.com/brizenchi/quickstart-template/internal/checkoutlog"
	"gorm.io/gorm"
)

type repository struct{ db *gorm.DB }

func (r repository) owned(ctx context.Context, owner, id string) error {
	var row checkoutlog.Record
	return r.db.WithContext(ctx).Where("user_id = ? AND provider = ? AND session_id = ? AND product_type = ?", owner, "stripe", id, "credits").Take(&row).Error
}

// related recognizes pre-session-key grants and derives payment state only from
// verified, owner-scoped provider events. It never trusts the browser's return URL.
func (r repository) related(ctx context.Context, owner, provider, id string) ([]string, string, error) {
	ids := []string{}
	state := "pending"
	paid := false
	lastID := uint(0)
	for {
		var events []billingdomain.BillingEvent
		if err := r.db.WithContext(ctx).Where("user_id = ? AND provider = ? AND id > ? AND event_type IN ?", owner, provider, lastID,
			[]string{"checkout.session.completed", "checkout.session.async_payment_succeeded", "checkout.session.async_payment_failed", "checkout.session.expired"}).Order("id ASC").Limit(100).Find(&events).Error; err != nil {
			return nil, "", err
		}
		if len(events) == 0 {
			break
		}
		for _, e := range events {
			lastID = e.ID
			var body struct {
				Data struct {
					Object struct {
						ID            string `json:"id"`
						PaymentStatus string `json:"payment_status"`
					} `json:"object"`
				} `json:"data"`
			}
			if json.Unmarshal(e.Payload, &body) != nil || body.Data.Object.ID != id {
				continue
			}
			ids = append(ids, e.ProviderEventID)
			if len(ids) > 100 {
				return nil, "", ErrConflict
			}
			if e.EventType == "checkout.session.async_payment_succeeded" || body.Data.Object.PaymentStatus == "paid" {
				paid = true
				state = "processing"
			} else if !paid && e.EventType == "checkout.session.async_payment_failed" {
				state = "failed"
			} else if !paid && e.EventType == "checkout.session.expired" {
				state = "expired"
			}
		}
	}
	return ids, state, nil
}
