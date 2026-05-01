package gormrepo

import (
	"context"
	"strings"

	"github.com/brizenchi/go-modules/modules/user/domain"
	"gorm.io/gorm"
)

// Repo is the GORM-backed repository for the standard users table.
type Repo struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Repo { return &Repo{db: db} }

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&UserRow{})
}

func (r *Repo) DB() *gorm.DB { return r.db }

func (r *Repo) FindByID(ctx context.Context, userID string) (*domain.User, error) {
	var row UserRow
	if err := r.db.WithContext(ctx).Where("id = ?", strings.TrimSpace(userID)).First(&row).Error; err != nil {
		return nil, err
	}
	return row.toDomain(), nil
}

func (r *Repo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var row UserRow
	if err := r.db.WithContext(ctx).Where("email = ?", strings.ToLower(strings.TrimSpace(email))).First(&row).Error; err != nil {
		return nil, err
	}
	return row.toDomain(), nil
}

func (r *Repo) FindByProviderSubject(ctx context.Context, provider, subject string) (*domain.User, error) {
	var row UserRow
	if err := r.db.WithContext(ctx).
		Where("provider = ? AND provider_subject = ?", strings.TrimSpace(provider), strings.TrimSpace(subject)).
		First(&row).Error; err != nil {
		return nil, err
	}
	return row.toDomain(), nil
}

func (r *Repo) FindByStripeCustomerID(ctx context.Context, customerID string) (*domain.User, error) {
	var row UserRow
	if err := r.db.WithContext(ctx).
		Where("stripe_customer_id = ?", strings.TrimSpace(customerID)).
		First(&row).Error; err != nil {
		return nil, err
	}
	return row.toDomain(), nil
}

func (r *Repo) FindByStripeSubscriptionID(ctx context.Context, subscriptionID string) (*domain.User, error) {
	var row UserRow
	if err := r.db.WithContext(ctx).
		Where("stripe_subscription_id = ?", strings.TrimSpace(subscriptionID)).
		First(&row).Error; err != nil {
		return nil, err
	}
	return row.toDomain(), nil
}

func (r *Repo) Create(ctx context.Context, user *domain.User) error {
	row := rowFromDomain(user)
	if err := r.db.WithContext(ctx).Create(row).Error; err != nil {
		return err
	}
	*user = *row.toDomain()
	return nil
}

func (r *Repo) Save(ctx context.Context, user *domain.User) error {
	row := rowFromDomain(user)
	if err := r.db.WithContext(ctx).Save(row).Error; err != nil {
		return err
	}
	*user = *row.toDomain()
	return nil
}

func (r *Repo) UpdateFields(ctx context.Context, userID string, updates map[string]any) error {
	if len(updates) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Model(&UserRow{}).
		Where("id = ?", strings.TrimSpace(userID)).
		Updates(updates).Error
}

func (r *Repo) AddCredits(ctx context.Context, userID string, delta int) error {
	if delta == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Model(&UserRow{}).
		Where("id = ?", strings.TrimSpace(userID)).
		UpdateColumn("credits", gorm.Expr("credits + ?", delta)).
		Error
}
