// Package gormrepo is a GORM-backed implementation of the referral repositories.
package gormrepo

import (
	"time"

	"github.com/brizenchi/go-modules/referral/domain"
)

// codeRow is the GORM row for referral_codes.
type codeRow struct {
	UserID    string    `gorm:"primaryKey;type:varchar(36)"`
	Value     string    `gorm:"uniqueIndex;type:varchar(64);not null"`
	CreatedAt time.Time
}

func (codeRow) TableName() string { return "referral_codes" }

func (r codeRow) toDomain() domain.Code {
	return domain.Code{UserID: r.UserID, Value: r.Value, CreatedAt: r.CreatedAt}
}

// referralRow is the GORM row for referrals.
type referralRow struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement"`
	Code          string    `gorm:"type:varchar(64);index;not null"`
	ReferrerID    string    `gorm:"type:varchar(36);index;not null"`
	RefereeID     string    `gorm:"type:varchar(36);uniqueIndex;not null"` // a referee has at most one referrer
	Status        string    `gorm:"type:varchar(16);index;not null;default:'pending'"`
	ActivatedAt   *time.Time
	ExpiresAt     *time.Time
	RewardCredits int `gorm:"not null;default:0"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (referralRow) TableName() string { return "referrals" }

func (r referralRow) toDomain() domain.Referral {
	return domain.Referral{
		ID:            r.ID,
		Code:          r.Code,
		ReferrerID:    r.ReferrerID,
		RefereeID:     r.RefereeID,
		Status:        domain.Status(r.Status),
		ActivatedAt:   r.ActivatedAt,
		ExpiresAt:     r.ExpiresAt,
		RewardCredits: r.RewardCredits,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
	}
}

func referralFromDomain(r domain.Referral) referralRow {
	return referralRow{
		ID:            r.ID,
		Code:          r.Code,
		ReferrerID:    r.ReferrerID,
		RefereeID:     r.RefereeID,
		Status:        string(r.Status),
		ActivatedAt:   r.ActivatedAt,
		ExpiresAt:     r.ExpiresAt,
		RewardCredits: r.RewardCredits,
	}
}

// AutoMigrateModels lists the GORM models for migration.
func AutoMigrateModels() []any {
	return []any{&codeRow{}, &referralRow{}}
}
