package repo

import (
	"context"
	"errors"
	"strings"

	"github.com/brizenchi/go-modules/modules/billing/domain"
	"github.com/brizenchi/go-modules/modules/billing/port"
	"gorm.io/gorm"
)

// UserResolver resolves webhook hints using billing-owned tables first,
// then falls back to the shared users table by email.
type UserResolver struct {
	db *gorm.DB
}

func NewUserResolver(db *gorm.DB) *UserResolver {
	return &UserResolver{db: db}
}

func (r *UserResolver) Resolve(ctx context.Context, h port.UserHint) (string, error) {
	if userID := strings.TrimSpace(h.UserID); userID != "" {
		return userID, nil
	}

	if customerID := strings.TrimSpace(h.ProviderCustomerID); customerID != "" {
		var customer domain.BillingCustomer
		err := r.db.WithContext(ctx).
			Where("provider_customer_id = ?", customerID).
			Order("updated_at DESC").
			Take(&customer).Error
		if err == nil {
			return customer.UserID, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", err
		}
	}

	if subscriptionID := strings.TrimSpace(h.ProviderSubscriptionID); subscriptionID != "" {
		var subscription domain.BillingSubscription
		err := r.db.WithContext(ctx).
			Where("provider_subscription_id = ?", subscriptionID).
			Order("updated_at DESC").
			Take(&subscription).Error
		if err == nil {
			return subscription.UserID, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", err
		}
	}

	if email := strings.TrimSpace(h.Email); email != "" {
		return loadUserIDByEmail(ctx, r.db, email)
	}

	return "", gorm.ErrRecordNotFound
}

var _ port.UserResolver = (*UserResolver)(nil)
