package repo

import (
	"context"
	"errors"
	"strings"

	"github.com/brizenchi/go-modules/modules/billing/domain"
	"github.com/brizenchi/go-modules/modules/billing/port"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CustomerStore persists provider customer linkage outside the shared
// users table while still projecting email/plan from users.
type CustomerStore struct {
	db *gorm.DB
}

func NewCustomerStore(db *gorm.DB) *CustomerStore {
	return &CustomerStore{db: db}
}

func (s *CustomerStore) LoadCustomer(ctx context.Context, userID string) (port.Customer, error) {
	user, err := loadUserSummaryByID(ctx, s.db, userID)
	if err != nil {
		return port.Customer{}, err
	}

	out := port.Customer{
		UserID: user.ID,
		Email:  user.Email,
		Plan:   user.Plan,
	}

	var customer domain.BillingCustomer
	err = s.db.WithContext(ctx).
		Where("user_id = ?", strings.TrimSpace(userID)).
		Order("updated_at DESC").
		Take(&customer).Error
	switch {
	case err == nil:
		out.ProviderCustomerID = customer.ProviderCustomerID
	case errors.Is(err, gorm.ErrRecordNotFound):
		// New users may not have provider linkage yet.
	default:
		return port.Customer{}, err
	}

	var subscription domain.BillingSubscription
	query := s.db.WithContext(ctx).
		Where("user_id = ?", strings.TrimSpace(userID))
	if out.ProviderCustomerID != "" {
		query = query.Where("provider_customer_id = ?", out.ProviderCustomerID)
	}
	err = query.Order("updated_at DESC").Take(&subscription).Error
	switch {
	case err == nil:
		if out.ProviderCustomerID == "" {
			out.ProviderCustomerID = subscription.ProviderCustomerID
		}
		out.ProviderSubscriptionID = subscription.ProviderSubscriptionID
	case errors.Is(err, gorm.ErrRecordNotFound):
		// No active or synced subscription yet.
	default:
		return port.Customer{}, err
	}

	return out, nil
}

func (s *CustomerStore) SaveCustomerID(ctx context.Context, userID, provider, customerID string) error {
	row := &domain.BillingCustomer{
		UserID:             strings.TrimSpace(userID),
		Provider:           strings.TrimSpace(provider),
		ProviderCustomerID: strings.TrimSpace(customerID),
	}
	return s.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "user_id"},
				{Name: "provider"},
			},
			DoUpdates: clause.AssignmentColumns([]string{
				"provider_customer_id",
				"updated_at",
			}),
		}).
		Create(row).Error
}

var _ port.CustomerStore = (*CustomerStore)(nil)
