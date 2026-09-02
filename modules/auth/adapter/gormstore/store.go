// Package gormstore persists auth-owned short-lived state with GORM.
//
// It deliberately contains no user table. The host supplies its own
// auth/port.UserStore while this adapter owns only verification codes,
// per-day counters and single-use OAuth exchange codes.
package gormstore

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/brizenchi/go-modules/modules/auth/domain"
	"github.com/brizenchi/go-modules/modules/auth/port"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type emailCodeRow struct {
	Email        string    `gorm:"primaryKey;type:varchar(255)"`
	Code         string    `gorm:"type:varchar(32);not null"`
	ExpiresAt    time.Time `gorm:"not null"`
	LastSentAt   time.Time `gorm:"not null"`
	AttemptCount int       `gorm:"not null;default:0"`
	UpdatedAt    time.Time
	CreatedAt    time.Time
}

func (emailCodeRow) TableName() string { return "auth_email_codes" }

type exchangeCodeRow struct {
	Code      string    `gorm:"primaryKey;type:varchar(255)"`
	UserID    string    `gorm:"type:varchar(36);not null;index"`
	Provider  string    `gorm:"type:varchar(32);not null"`
	IsNew     bool      `gorm:"not null;default:false"`
	ExpiresAt time.Time `gorm:"not null;index"`
	UpdatedAt time.Time
	CreatedAt time.Time
}

func (exchangeCodeRow) TableName() string { return "auth_exchange_codes" }

type dailyCountRow struct {
	Email     string `gorm:"primaryKey;type:varchar(255)"`
	DayBucket string `gorm:"primaryKey;type:char(10)"`
	Count     int    `gorm:"not null;default:0"`
	UpdatedAt time.Time
	CreatedAt time.Time
}

func (dailyCountRow) TableName() string { return "auth_email_daily_counts" }

// Models returns the persistence models owned by the auth module.
func Models() []any {
	return []any{&emailCodeRow{}, &exchangeCodeRow{}, &dailyCountRow{}}
}

// AutoMigrate creates or updates the auth-owned tables.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(Models()...)
}

// Store implements CodeRateLimitStore and ExchangeCodeStore.
type Store struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Store { return &Store{db: db} }

func (s *Store) SaveCode(ctx context.Context, email, code string, expiresAt, lastSentAt time.Time) error {
	row := emailCodeRow{
		Email:        normalizeEmail(email),
		Code:         strings.TrimSpace(code),
		ExpiresAt:    expiresAt.UTC(),
		LastSentAt:   lastSentAt.UTC(),
		AttemptCount: 0,
	}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "email"}},
		DoUpdates: clause.AssignmentColumns([]string{"code", "expires_at", "last_sent_at", "attempt_count", "updated_at"}),
	}).Create(&row).Error
}

func (s *Store) LoadCode(ctx context.Context, email string) (string, time.Time, error) {
	var row emailCodeRow
	err := s.db.WithContext(ctx).Where("email = ?", normalizeEmail(email)).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", time.Time{}, nil
		}
		return "", time.Time{}, err
	}
	if time.Now().UTC().After(row.ExpiresAt) {
		_ = s.db.WithContext(ctx).Delete(&emailCodeRow{}, "email = ?", row.Email).Error
		return "", time.Time{}, nil
	}
	return row.Code, row.LastSentAt, nil
}

func (s *Store) DeleteCode(ctx context.Context, email string) error {
	return s.db.WithContext(ctx).Delete(&emailCodeRow{}, "email = ?", normalizeEmail(email)).Error
}

func (s *Store) IncrAttempts(ctx context.Context, email string) (int, error) {
	email = normalizeEmail(email)
	if err := s.db.WithContext(ctx).
		Model(&emailCodeRow{}).
		Where("email = ?", email).
		UpdateColumn("attempt_count", gorm.Expr("attempt_count + 1")).Error; err != nil {
		return 0, err
	}
	var row emailCodeRow
	if err := s.db.WithContext(ctx).Where("email = ?", email).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return row.AttemptCount, nil
}

func (s *Store) IncrDailyCount(ctx context.Context, email, dayBucket string) (int, error) {
	email = normalizeEmail(email)
	dayBucket = strings.TrimSpace(dayBucket)
	if _, err := time.Parse("2006-01-02", dayBucket); err != nil {
		dayBucket = time.Now().UTC().Format("2006-01-02")
	}

	row := dailyCountRow{Email: email, DayBucket: dayBucket, Count: 1}
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "email"}, {Name: "day_bucket"}},
		DoUpdates: clause.Assignments(map[string]any{
			"count":      gorm.Expr("auth_email_daily_counts.count + 1"),
			"updated_at": gorm.Expr("CURRENT_TIMESTAMP"),
		}),
	}).Create(&row).Error; err != nil {
		return 0, err
	}

	var current dailyCountRow
	if err := s.db.WithContext(ctx).
		Where("email = ? AND day_bucket = ?", email, dayBucket).
		First(&current).Error; err != nil {
		return 0, err
	}
	return current.Count, nil
}

func (s *Store) Save(ctx context.Context, code domain.ExchangeCode) error {
	row := exchangeCodeRow{
		Code:      strings.TrimSpace(code.Code),
		UserID:    strings.TrimSpace(code.UserID),
		Provider:  string(code.Provider),
		IsNew:     code.IsNew,
		ExpiresAt: code.ExpiresAt.UTC(),
	}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "code"}},
		DoUpdates: clause.AssignmentColumns([]string{"user_id", "provider", "is_new", "expires_at", "updated_at"}),
	}).Create(&row).Error
}

func (s *Store) Consume(ctx context.Context, code string) (*domain.ExchangeCode, error) {
	code = strings.TrimSpace(code)
	var row exchangeCodeRow
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("code = ?", code).
			First(&row).Error; err != nil {
			return err
		}
		return tx.Delete(&exchangeCodeRow{}, "code = ?", code).Error
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrInvalidExchange
		}
		return nil, err
	}
	if time.Now().UTC().After(row.ExpiresAt) {
		return nil, domain.ErrInvalidExchange
	}
	return &domain.ExchangeCode{
		Code:      row.Code,
		UserID:    row.UserID,
		Provider:  domain.Provider(row.Provider),
		IsNew:     row.IsNew,
		ExpiresAt: row.ExpiresAt,
	}, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

var _ port.CodeRateLimitStore = (*Store)(nil)
var _ port.ExchangeCodeStore = (*Store)(nil)
