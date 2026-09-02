package repo

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/brizenchi/go-modules/modules/billing/domain"
	"github.com/brizenchi/go-modules/modules/billing/port"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CustomerStore persists provider linkage in billing-owned tables and gets
// the host account's ID/email through port.AccountLookup.
type CustomerStore struct {
	db       *gorm.DB
	accounts port.AccountLookup
}

func NewCustomerStore(db *gorm.DB, accounts port.AccountLookup) *CustomerStore {
	return &CustomerStore{db: db, accounts: accounts}
}

func (s *CustomerStore) LoadCustomer(ctx context.Context, userID string) (port.Customer, error) {
	if s.accounts == nil {
		return port.Customer{}, fmt.Errorf("billing: account lookup required")
	}
	account, err := s.accounts.FindBillingAccount(ctx, userID)
	if err != nil {
		return port.Customer{}, err
	}

	out := port.Customer{
		UserID: account.UserID,
		Email:  account.Email,
		Plan:   string(domain.PlanFree),
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
		if strings.TrimSpace(subscription.Plan) != "" {
			out.Plan = subscription.Plan
		}
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

func (s *CustomerStore) HasUsedTrial(ctx context.Context, userID string) (bool, error) {
	var count int64
	err := s.db.WithContext(ctx).
		Model(&domain.BillingSubscription{}).
		Where("user_id = ? AND status IN ?", strings.TrimSpace(userID), []string{
			string(domain.StatusTrialing),
			string(domain.StatusActive),
			string(domain.StatusCanceling),
			string(domain.StatusCanceled),
			string(domain.StatusPastDue),
		}).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

var _ port.CustomerStore = (*CustomerStore)(nil)
