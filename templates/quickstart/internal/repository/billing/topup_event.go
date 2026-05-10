package billing

import (
	"errors"
	"strings"
	"time"

	"github.com/brizenchi/go-modules/modules/user/adapter/gormrepo"
	billingentity "github.com/brizenchi/quickstart-template/internal/model/entity/billing"
	"gorm.io/gorm"
)

type StripeTopUpEventRepository struct{}

func NewStripeTopUpEventRepository() *StripeTopUpEventRepository {
	return &StripeTopUpEventRepository{}
}

func (r *StripeTopUpEventRepository) Find(tx *gorm.DB, providerEventID, paymentIntentID string) (*billingentity.StripeTopUpEvent, error) {
	var eventRow billingentity.StripeTopUpEvent
	err := tx.
		Where("provider_event_id = ? OR payment_intent_id = ?", strings.TrimSpace(providerEventID), strings.TrimSpace(paymentIntentID)).
		First(&eventRow).
		Error
	if err != nil {
		return nil, err
	}
	return &eventRow, nil
}

func (r *StripeTopUpEventRepository) SavePending(
	tx *gorm.DB,
	providerEventID, paymentIntentID, userID, customerID string,
	amountCents, credits int64,
	payload []byte,
) (*billingentity.StripeTopUpEvent, error) {
	eventRow := &billingentity.StripeTopUpEvent{
		ProviderEventID: strings.TrimSpace(providerEventID),
		PaymentIntentID: strings.TrimSpace(paymentIntentID),
		UserID:          strings.TrimSpace(userID),
		CustomerID:      strings.TrimSpace(customerID),
		AmountCents:     amountCents,
		Credits:         credits,
		Payload:         append([]byte(nil), payload...),
	}
	if err := tx.Create(eventRow).Error; err != nil {
		return nil, err
	}
	return eventRow, nil
}

func (r *StripeTopUpEventRepository) Finish(
	tx *gorm.DB,
	eventRow *billingentity.StripeTopUpEvent,
	userID, customerID string,
	amountCents, credits int64,
	now time.Time,
) error {
	if eventRow == nil {
		return errors.New("stripe top-up: event row required")
	}
	if strings.TrimSpace(userID) == "" {
		return errors.New("stripe top-up: user_id required")
	}

	updates := map[string]any{}
	if strings.TrimSpace(customerID) != "" {
		updates["stripe_customer_id"] = strings.TrimSpace(customerID)
	}
	if len(updates) > 0 {
		if err := tx.Model(&gormrepo.UserRow{}).
			Where("id = ?", strings.TrimSpace(userID)).
			Updates(updates).Error; err != nil {
			return err
		}
	}

	res := tx.Model(&gormrepo.UserRow{}).
		Where("id = ?", strings.TrimSpace(userID)).
		UpdateColumn("credits", gorm.Expr("credits + ?", int(credits)))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return tx.Model(eventRow).
		Updates(map[string]any{
			"user_id":      strings.TrimSpace(userID),
			"customer_id":  strings.TrimSpace(customerID),
			"amount_cents": amountCents,
			"credits":      credits,
			"processed":    true,
			"processed_at": &now,
		}).
		Error
}

func IsUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "unique failed")
}
