package gormrepo

import (
	"strings"
	"time"

	"github.com/brizenchi/go-modules/modules/user/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserRow is the GORM model for the shared users table.
type UserRow struct {
	ID                   string `gorm:"primaryKey;type:varchar(36)"`
	Email                string `gorm:"type:varchar(255);uniqueIndex;not null"`
	EmailVerified        bool   `gorm:"default:false;not null"`
	EmailVerifiedAt      *time.Time
	Username             string `gorm:"type:varchar(100)"`
	AvatarURL            string `gorm:"type:varchar(512)"`
	Provider             string `gorm:"type:varchar(50)"`
	ProviderSubject      string `gorm:"type:varchar(255);index"`
	Role                 string `gorm:"type:varchar(20);default:'user';not null;index"`
	Plan                 string `gorm:"type:varchar(20);default:'free';not null"`
	Credits              int    `gorm:"default:0;not null"`
	StripeCustomerID     string `gorm:"type:varchar(255);index"`
	StripeSubscriptionID string `gorm:"type:varchar(255);index"`
	StripePriceID        string `gorm:"type:varchar(255)"`
	StripeProductID      string `gorm:"type:varchar(255)"`
	BillingStatus        string `gorm:"type:varchar(64)"`
	BillingPeriodStart   *time.Time
	BillingPeriodEnd     *time.Time
	CancelEffectiveAt    *time.Time
	LastLoginAt          *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

func (UserRow) TableName() string { return "users" }

func (u *UserRow) BeforeCreate(_ *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.NewString()
	}
	normalizeRow(u)
	return nil
}

func (u *UserRow) BeforeSave(_ *gorm.DB) error {
	normalizeRow(u)
	return nil
}

func (u UserRow) toDomain() *domain.User {
	return &domain.User{
		ID:                   u.ID,
		Email:                u.Email,
		EmailVerified:        u.EmailVerified,
		EmailVerifiedAt:      u.EmailVerifiedAt,
		Username:             u.Username,
		AvatarURL:            u.AvatarURL,
		Provider:             u.Provider,
		ProviderSubject:      u.ProviderSubject,
		Role:                 u.Role,
		Plan:                 u.Plan,
		Credits:              u.Credits,
		StripeCustomerID:     u.StripeCustomerID,
		StripeSubscriptionID: u.StripeSubscriptionID,
		StripePriceID:        u.StripePriceID,
		StripeProductID:      u.StripeProductID,
		BillingStatus:        u.BillingStatus,
		BillingPeriodStart:   u.BillingPeriodStart,
		BillingPeriodEnd:     u.BillingPeriodEnd,
		CancelEffectiveAt:    u.CancelEffectiveAt,
		LastLoginAt:          u.LastLoginAt,
		CreatedAt:            u.CreatedAt,
		UpdatedAt:            u.UpdatedAt,
	}
}

func rowFromDomain(u *domain.User) *UserRow {
	if u == nil {
		return nil
	}
	row := &UserRow{
		ID:                   u.ID,
		Email:                u.Email,
		EmailVerified:        u.EmailVerified,
		EmailVerifiedAt:      u.EmailVerifiedAt,
		Username:             u.Username,
		AvatarURL:            u.AvatarURL,
		Provider:             u.Provider,
		ProviderSubject:      u.ProviderSubject,
		Role:                 u.Role,
		Plan:                 u.Plan,
		Credits:              u.Credits,
		StripeCustomerID:     u.StripeCustomerID,
		StripeSubscriptionID: u.StripeSubscriptionID,
		StripePriceID:        u.StripePriceID,
		StripeProductID:      u.StripeProductID,
		BillingStatus:        u.BillingStatus,
		BillingPeriodStart:   u.BillingPeriodStart,
		BillingPeriodEnd:     u.BillingPeriodEnd,
		CancelEffectiveAt:    u.CancelEffectiveAt,
		LastLoginAt:          u.LastLoginAt,
		CreatedAt:            u.CreatedAt,
		UpdatedAt:            u.UpdatedAt,
	}
	normalizeRow(row)
	return row
}

func NormalizeUserRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case domain.RoleAdmin:
		return domain.RoleAdmin
	default:
		return domain.RoleUser
	}
}

func NormalizePlan(plan string) string {
	switch strings.ToLower(strings.TrimSpace(plan)) {
	case domain.PlanStarter:
		return domain.PlanStarter
	case domain.PlanPro:
		return domain.PlanPro
	case domain.PlanPremium:
		return domain.PlanPremium
	case domain.PlanLifetime:
		return domain.PlanLifetime
	default:
		return domain.PlanFree
	}
}

func normalizeRow(user *UserRow) {
	if user == nil {
		return
	}
	user.Email = strings.ToLower(strings.TrimSpace(user.Email))
	user.Provider = strings.TrimSpace(user.Provider)
	user.ProviderSubject = strings.TrimSpace(user.ProviderSubject)
	user.Role = NormalizeUserRole(user.Role)
	user.Plan = NormalizePlan(user.Plan)
	if user.Provider == "email" && user.ProviderSubject == "" {
		user.ProviderSubject = user.Email
	}
}
