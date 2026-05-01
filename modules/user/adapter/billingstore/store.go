package billingstore

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/brizenchi/go-modules/modules/billing/domain"
	billingport "github.com/brizenchi/go-modules/modules/billing/port"
	"github.com/brizenchi/go-modules/modules/user/adapter/gormrepo"
	userdomain "github.com/brizenchi/go-modules/modules/user/domain"
	"gorm.io/gorm"
)

// CustomerStore adapts the shared users table to billing.CustomerStore.
type CustomerStore struct {
	users *gormrepo.Repo
}

func NewCustomerStore(users *gormrepo.Repo) *CustomerStore { return &CustomerStore{users: users} }

func (s *CustomerStore) LoadCustomer(ctx context.Context, userID string) (billingport.Customer, error) {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return billingport.Customer{}, err
	}
	return billingport.Customer{
		UserID:                 user.ID,
		Email:                  user.Email,
		ProviderCustomerID:     user.StripeCustomerID,
		ProviderSubscriptionID: user.StripeSubscriptionID,
	}, nil
}

func (s *CustomerStore) SaveCustomerID(ctx context.Context, userID, provider, customerID string) error {
	if strings.TrimSpace(provider) != "stripe" {
		return nil
	}
	return s.users.UpdateFields(ctx, userID, map[string]any{
		"stripe_customer_id": strings.TrimSpace(customerID),
	})
}

// UserResolver adapts the shared users table to billing.UserResolver.
type UserResolver struct {
	users *gormrepo.Repo
}

func NewUserResolver(users *gormrepo.Repo) *UserResolver { return &UserResolver{users: users} }

func (r *UserResolver) Resolve(ctx context.Context, h billingport.UserHint) (string, error) {
	if userID := strings.TrimSpace(h.UserID); userID != "" {
		return userID, nil
	}
	if customerID := strings.TrimSpace(h.ProviderCustomerID); customerID != "" {
		user, err := r.users.FindByStripeCustomerID(ctx, customerID)
		if err == nil {
			return user.ID, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", err
		}
	}
	if subscriptionID := strings.TrimSpace(h.ProviderSubscriptionID); subscriptionID != "" {
		user, err := r.users.FindByStripeSubscriptionID(ctx, subscriptionID)
		if err == nil {
			return user.ID, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", err
		}
	}
	if email := strings.TrimSpace(h.Email); email != "" {
		user, err := r.users.FindByEmail(ctx, email)
		if err == nil {
			return user.ID, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", err
		}
	}
	return "", gorm.ErrRecordNotFound
}

func ApplyFreePlan(ctx context.Context, users *gormrepo.Repo, userID string) error {
	return users.UpdateFields(ctx, userID, map[string]any{
		"plan":           userdomain.PlanFree,
		"billing_status": "",
	})
}

func ApplySubscriptionSnapshot(ctx context.Context, users *gormrepo.Repo, userID string, snapshot domain.SubscriptionSnapshot) error {
	return users.UpdateFields(ctx, userID, map[string]any{
		"plan":                   string(snapshot.Plan),
		"billing_status":         string(snapshot.Status),
		"stripe_subscription_id": snapshot.ProviderSubscriptionID,
		"stripe_customer_id":     snapshot.ProviderCustomerID,
		"stripe_price_id":        snapshot.ProviderPriceID,
		"stripe_product_id":      snapshot.ProviderProductID,
		"billing_period_start":   snapshot.PeriodStart,
		"billing_period_end":     snapshot.PeriodEnd,
		"cancel_effective_at":    snapshot.CancelEffectiveAt,
	})
}

func ApplySubscriptionCanceling(ctx context.Context, users *gormrepo.Repo, userID string, effectiveAt *time.Time) error {
	return users.UpdateFields(ctx, userID, map[string]any{
		"billing_status":      string(domain.StatusCanceling),
		"cancel_effective_at": effectiveAt,
	})
}

func ApplySubscriptionCanceled(ctx context.Context, users *gormrepo.Repo, userID string) error {
	return users.UpdateFields(ctx, userID, map[string]any{
		"plan":                   userdomain.PlanFree,
		"billing_status":         string(domain.StatusCanceled),
		"stripe_subscription_id": "",
		"stripe_price_id":        "",
		"stripe_product_id":      "",
		"billing_period_start":   nil,
		"billing_period_end":     nil,
		"cancel_effective_at":    nil,
	})
}

func ApplyPaymentFailed(ctx context.Context, users *gormrepo.Repo, userID string) error {
	return users.UpdateFields(ctx, userID, map[string]any{
		"billing_status": string(domain.StatusPaymentFailed),
	})
}

var (
	_ billingport.CustomerStore = (*CustomerStore)(nil)
	_ billingport.UserResolver  = (*UserResolver)(nil)
)

