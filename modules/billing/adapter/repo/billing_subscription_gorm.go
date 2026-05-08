package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/brizenchi/go-modules/modules/billing/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SubscriptionRepo persists the current billing snapshot for each
// (user, provider) pair.
type SubscriptionRepo struct {
	db *gorm.DB
}

func NewSubscriptionRepo(db *gorm.DB) *SubscriptionRepo {
	return &SubscriptionRepo{db: db}
}

func (r *SubscriptionRepo) UpsertSnapshot(ctx context.Context, userID, provider string, snapshot domain.SubscriptionSnapshot) error {
	userID = strings.TrimSpace(userID)
	provider = strings.TrimSpace(provider)
	if userID == "" {
		return fmt.Errorf("billing: user_id required")
	}
	if provider == "" {
		return fmt.Errorf("billing: provider required")
	}

	raw, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}

	row := &domain.BillingSubscription{
		UserID:                 userID,
		Provider:               provider,
		ProviderCustomerID:     strings.TrimSpace(snapshot.ProviderCustomerID),
		ProviderSubscriptionID: strings.TrimSpace(snapshot.ProviderSubscriptionID),
		ProviderPriceID:        strings.TrimSpace(snapshot.ProviderPriceID),
		ProviderProductID:      strings.TrimSpace(snapshot.ProviderProductID),
		ProductType:            strings.TrimSpace(string(snapshot.ProductType)),
		Plan:                   strings.TrimSpace(string(snapshot.Plan)),
		BillingInterval:        strings.TrimSpace(string(snapshot.Interval)),
		Status:                 strings.TrimSpace(string(snapshot.Status)),
		CancelAtPeriodEnd:      snapshot.CancelAtPeriodEnd,
		PeriodStart:            snapshot.PeriodStart,
		PeriodEnd:              snapshot.PeriodEnd,
		CancelEffectiveAt:      snapshot.CancelEffectiveAt,
		RawSnapshotJSON:        raw,
	}

	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "user_id"},
				{Name: "provider"},
			},
			DoUpdates: clause.AssignmentColumns([]string{
				"provider_customer_id",
				"provider_subscription_id",
				"provider_price_id",
				"provider_product_id",
				"product_type",
				"plan",
				"billing_interval",
				"status",
				"cancel_at_period_end",
				"period_start",
				"period_end",
				"cancel_effective_at",
				"raw_snapshot_json",
				"updated_at",
			}),
		}).
		Create(row).Error
}

func (r *SubscriptionRepo) FindByUser(ctx context.Context, userID string) (*domain.BillingSubscription, error) {
	var row domain.BillingSubscription
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", strings.TrimSpace(userID)).
		Order("updated_at DESC").
		Take(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *SubscriptionRepo) FindByProviderSubscriptionID(ctx context.Context, subscriptionID string) (*domain.BillingSubscription, error) {
	var row domain.BillingSubscription
	if err := r.db.WithContext(ctx).
		Where("provider_subscription_id = ?", strings.TrimSpace(subscriptionID)).
		Order("updated_at DESC").
		Take(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}
