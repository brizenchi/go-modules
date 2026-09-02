// Package user owns this SaaS's user schema.
//
// This package is intentionally inside the copied template, not go-modules.
// Add product-specific user fields here and migrate them in this SaaS only.
package user

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

// User is the host application's profile and authorization record.
// Billing provider IDs, subscriptions and referrals live in their modules'
// tables rather than being copied into this struct.
type User struct {
	ID              string     `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Email           string     `json:"email" gorm:"type:varchar(255);uniqueIndex;not null"`
	EmailVerified   bool       `json:"email_verified" gorm:"not null;default:false"`
	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty"`
	Username        string     `json:"username,omitempty" gorm:"type:varchar(100)"`
	AvatarURL       string     `json:"avatar_url,omitempty" gorm:"type:varchar(512)"`
	Role            string     `json:"role" gorm:"type:varchar(20);not null;default:'user';index"`
	Credits         int64      `json:"credits" gorm:"not null;default:0"`
	LastLoginAt     *time.Time `json:"last_login_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`

	// Credits 是 quickstart 的宿主业务示例；不需要积分的 SaaS 可以连同监听器一起删除。
	// 在这里添加当前 SaaS 独有的字段，例如 WorkspaceID、Locale、OnboardingStep。
}

func (User) TableName() string { return "users" }

func (u *User) BeforeCreate(_ *gorm.DB) error {
	if strings.TrimSpace(u.ID) == "" {
		u.ID = uuid.NewString()
	}
	u.normalize()
	return nil
}

func (u *User) BeforeSave(_ *gorm.DB) error {
	u.normalize()
	return nil
}

func (u *User) normalize() {
	u.Email = normalizeEmail(u.Email)
	u.Username = strings.TrimSpace(u.Username)
	u.AvatarURL = strings.TrimSpace(u.AvatarURL)
	u.Role = normalizeRole(u.Role)
}

// Identity links a host user to one external identity. Keeping it separate
// lets the same user connect Google and GitHub without adding provider columns
// to User or overwriting an existing login.
type Identity struct {
	ID        uint   `gorm:"primaryKey;autoIncrement"`
	UserID    string `gorm:"type:varchar(36);not null;uniqueIndex:uniq_user_identity_provider;index"`
	Provider  string `gorm:"type:varchar(32);not null;uniqueIndex:uniq_user_identity_provider;uniqueIndex:uniq_provider_subject"`
	Subject   string `gorm:"type:varchar(255);not null;uniqueIndex:uniq_provider_subject"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Identity) TableName() string { return "user_identities" }

// CreditGrant is an idempotency ledger for externally triggered credit
// changes. A Stripe webhook retry with the same provider event ID therefore
// cannot add credits twice.
type CreditGrant struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	UserID    string    `gorm:"type:varchar(36);not null;index"`
	Source    string    `gorm:"type:varchar(32);not null;uniqueIndex:uniq_credit_grant_source"`
	SourceID  string    `gorm:"type:varchar(255);not null;uniqueIndex:uniq_credit_grant_source"`
	Amount    int64     `gorm:"not null"`
	CreatedAt time.Time `gorm:"not null"`
}

func (CreditGrant) TableName() string { return "user_credit_grants" }

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func normalizeRole(role string) string {
	if strings.EqualFold(strings.TrimSpace(role), RoleAdmin) {
		return RoleAdmin
	}
	return RoleUser
}
