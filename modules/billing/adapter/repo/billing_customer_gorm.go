package repo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

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
		out.SubscriptionStatus = domain.SubscriptionStatus(subscription.Status)
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

func (s *CustomerStore) ReserveCheckout(ctx context.Context, userID, provider, providerCustomerID, reservationID, intentKey string, now, expiresAt time.Time) (*domain.BillingCheckoutReservation, bool, error) {
	userID = strings.TrimSpace(userID)
	provider = strings.TrimSpace(provider)
	providerCustomerID = strings.TrimSpace(providerCustomerID)
	reservationID = strings.TrimSpace(reservationID)
	intentKey = strings.TrimSpace(intentKey)
	if userID == "" || provider == "" || providerCustomerID == "" || reservationID == "" || intentKey == "" || expiresAt.IsZero() || !expiresAt.After(now) {
		return nil, false, fmt.Errorf("billing: invalid checkout reservation")
	}
	now = now.UTC()
	expiresAt = expiresAt.UTC()
	row := &domain.BillingCheckoutReservation{
		UserID:             userID,
		Provider:           provider,
		ProviderCustomerID: providerCustomerID,
		ReservationID:      reservationID,
		IntentKey:          intentKey,
		ExpiresAt:          expiresAt,
	}
	result := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "provider"}},
		DoNothing: true,
	}).Create(row)
	if result.Error != nil {
		return nil, false, result.Error
	}
	if result.RowsAffected == 1 {
		return row, true, nil
	}

	var existing domain.BillingCheckoutReservation
	if err := s.db.WithContext(ctx).
		Where("user_id = ? AND provider = ?", userID, provider).
		Take(&existing).Error; err != nil {
		return nil, false, err
	}
	return &existing, false, nil
}

func (s *CustomerStore) ReplaceCheckoutReservation(ctx context.Context, userID, provider, providerCustomerID, expectedReservationID, reservationID, intentKey string, expiresAt time.Time) (bool, error) {
	userID = strings.TrimSpace(userID)
	provider = strings.TrimSpace(provider)
	providerCustomerID = strings.TrimSpace(providerCustomerID)
	expectedReservationID = strings.TrimSpace(expectedReservationID)
	reservationID = strings.TrimSpace(reservationID)
	intentKey = strings.TrimSpace(intentKey)
	if userID == "" || provider == "" || providerCustomerID == "" || expectedReservationID == "" || reservationID == "" || intentKey == "" || expiresAt.IsZero() {
		return false, fmt.Errorf("billing: invalid checkout reservation replacement")
	}
	result := s.db.WithContext(ctx).
		Model(&domain.BillingCheckoutReservation{}).
		Where("user_id = ? AND provider = ? AND reservation_id = ?", userID, provider, expectedReservationID).
		Updates(map[string]any{
			"provider_customer_id":     providerCustomerID,
			"reservation_id":           reservationID,
			"intent_key":               intentKey,
			"session_id":               "",
			"provider_subscription_id": "",
			"checkout_url":             "",
			"expires_at":               expiresAt.UTC(),
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected == 1, nil
}

func (s *CustomerStore) CompleteCheckoutReservation(ctx context.Context, userID, provider, reservationID string, checkout domain.CheckoutResult) error {
	result := s.db.WithContext(ctx).
		Model(&domain.BillingCheckoutReservation{}).
		Where("user_id = ? AND provider = ? AND reservation_id = ?", strings.TrimSpace(userID), strings.TrimSpace(provider), strings.TrimSpace(reservationID)).
		Updates(map[string]any{
			"session_id":   strings.TrimSpace(checkout.SessionID),
			"checkout_url": strings.TrimSpace(checkout.CheckoutURL),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("billing: checkout reservation was lost")
	}
	return nil
}

func (s *CustomerStore) LinkCheckoutSubscription(ctx context.Context, provider, providerSessionID, providerSubscriptionID string) error {
	provider = strings.TrimSpace(provider)
	providerSessionID = strings.TrimSpace(providerSessionID)
	providerSubscriptionID = strings.TrimSpace(providerSubscriptionID)
	if provider == "" || providerSessionID == "" || providerSubscriptionID == "" {
		return nil
	}
	return s.db.WithContext(ctx).
		Model(&domain.BillingCheckoutReservation{}).
		Where("provider = ? AND session_id = ?", provider, providerSessionID).
		Update("provider_subscription_id", providerSubscriptionID).Error
}

func (s *CustomerStore) ReleaseCheckoutReservation(ctx context.Context, provider, providerSessionID string) (bool, error) {
	provider = strings.TrimSpace(provider)
	providerSessionID = strings.TrimSpace(providerSessionID)
	if provider == "" || providerSessionID == "" {
		return false, nil
	}
	result := s.db.WithContext(ctx).
		Where("provider = ? AND session_id = ?", provider, providerSessionID).
		Delete(&domain.BillingCheckoutReservation{})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (s *CustomerStore) ReleaseCheckoutReservationByReservationID(ctx context.Context, provider, reservationID string) (bool, error) {
	provider = strings.TrimSpace(provider)
	reservationID = strings.TrimSpace(reservationID)
	if provider == "" || reservationID == "" {
		return false, nil
	}
	result := s.db.WithContext(ctx).
		Where("provider = ? AND reservation_id = ?", provider, reservationID).
		Delete(&domain.BillingCheckoutReservation{})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (s *CustomerStore) ReleaseCheckoutReservationBySubscription(ctx context.Context, provider, providerSubscriptionID string) (bool, error) {
	provider = strings.TrimSpace(provider)
	providerSubscriptionID = strings.TrimSpace(providerSubscriptionID)
	if provider == "" || providerSubscriptionID == "" {
		return false, nil
	}
	result := s.db.WithContext(ctx).
		Where("provider = ? AND provider_subscription_id = ?", provider, providerSubscriptionID).
		Delete(&domain.BillingCheckoutReservation{})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

var _ port.CustomerStore = (*CustomerStore)(nil)
var _ port.CheckoutReservationRepository = (*CustomerStore)(nil)
