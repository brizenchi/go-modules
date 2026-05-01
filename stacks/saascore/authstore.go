package saascore

import (
	"context"
	"strings"
	"time"

	authdomain "github.com/brizenchi/go-modules/modules/auth/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type authCodeRow struct {
	Email        string    `gorm:"primaryKey;type:varchar(255)"`
	Code         string    `gorm:"type:varchar(32);not null"`
	ExpiresAt    time.Time `gorm:"not null"`
	LastSentAt   time.Time `gorm:"not null"`
	AttemptCount int       `gorm:"not null;default:0"`
	UpdatedAt    time.Time
	CreatedAt    time.Time
}

func (authCodeRow) TableName() string { return "auth_email_codes" }

type authExchangeRow struct {
	Code      string    `gorm:"primaryKey;type:varchar(255)"`
	UserID    string    `gorm:"type:varchar(36);not null;index"`
	Provider  string    `gorm:"type:varchar(32);not null"`
	IsNew     bool      `gorm:"not null;default:false"`
	ExpiresAt time.Time `gorm:"not null;index"`
	UpdatedAt time.Time
	CreatedAt time.Time
}

func (authExchangeRow) TableName() string { return "auth_exchange_codes" }

type authDailyCountRow struct {
	Email     string `gorm:"primaryKey;type:varchar(255)"`
	DayBucket string `gorm:"primaryKey;type:char(10)"`
	Count     int    `gorm:"not null;default:0"`
	UpdatedAt time.Time
	CreatedAt time.Time
}

func (authDailyCountRow) TableName() string { return "auth_email_daily_counts" }

type authStore struct {
	db *gorm.DB
}

func newAuthStore(db *gorm.DB) *authStore {
	return &authStore{db: db}
}

func autoMigrateAuthStore(db *gorm.DB) error {
	return db.AutoMigrate(&authCodeRow{}, &authExchangeRow{}, &authDailyCountRow{})
}

func (s *authStore) SaveCode(ctx context.Context, email, code string, expiresAt, lastSentAt time.Time) error {
	row := authCodeRow{
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

func (s *authStore) LoadCode(ctx context.Context, email string) (string, time.Time, error) {
	var row authCodeRow
	err := s.db.WithContext(ctx).Where("email = ?", normalizeEmail(email)).First(&row).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", time.Time{}, nil
		}
		return "", time.Time{}, err
	}
	if time.Now().UTC().After(row.ExpiresAt) {
		_ = s.db.WithContext(ctx).Delete(&authCodeRow{}, "email = ?", row.Email).Error
		return "", time.Time{}, nil
	}
	return row.Code, row.LastSentAt, nil
}

func (s *authStore) DeleteCode(ctx context.Context, email string) error {
	return s.db.WithContext(ctx).Delete(&authCodeRow{}, "email = ?", normalizeEmail(email)).Error
}

func (s *authStore) IncrAttempts(ctx context.Context, email string) (int, error) {
	email = normalizeEmail(email)
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Model(&authCodeRow{}).
			Where("email = ?", email).
			UpdateColumn("attempt_count", gorm.Expr("attempt_count + 1")).Error
	})
	if err != nil {
		return 0, err
	}
	var row authCodeRow
	if err := s.db.WithContext(ctx).Where("email = ?", email).First(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
		return 0, err
	}
	return row.AttemptCount, nil
}

func (s *authStore) IncrDailyCount(ctx context.Context, email, dayBucket string) (int, error) {
	email = normalizeEmail(email)
	dayBucket = strings.TrimSpace(dayBucket)
	if _, err := time.Parse("2006-01-02", dayBucket); err != nil {
		dayBucket = time.Now().UTC().Format("2006-01-02")
	}

	row := authDailyCountRow{
		Email:     email,
		DayBucket: dayBucket,
		Count:     1,
	}
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "email"}, {Name: "day_bucket"}},
		DoUpdates: clause.Assignments(map[string]any{
			"count":      gorm.Expr("\"auth_email_daily_counts\".\"count\" + 1"),
			"updated_at": gorm.Expr("CURRENT_TIMESTAMP"),
		}),
	}).Create(&row).Error; err != nil {
		return 0, err
	}

	var current authDailyCountRow
	if err := s.db.WithContext(ctx).
		Where("email = ? AND day_bucket = ?", email, dayBucket).
		First(&current).Error; err != nil {
		return 0, err
	}
	return current.Count, nil
}

func (s *authStore) Save(ctx context.Context, code authdomain.ExchangeCode) error {
	row := authExchangeRow{
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

func (s *authStore) Consume(ctx context.Context, code string) (*authdomain.ExchangeCode, error) {
	code = strings.TrimSpace(code)
	var row authExchangeRow
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("code = ?", code).
			First(&row).Error; err != nil {
			return err
		}
		return tx.Delete(&authExchangeRow{}, "code = ?", code).Error
	})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, authdomain.ErrInvalidExchange
		}
		return nil, err
	}
	if time.Now().UTC().After(row.ExpiresAt) {
		return nil, authdomain.ErrInvalidExchange
	}
	return &authdomain.ExchangeCode{
		Code:      row.Code,
		UserID:    row.UserID,
		Provider:  authdomain.Provider(row.Provider),
		IsNew:     row.IsNew,
		ExpiresAt: row.ExpiresAt,
	}, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
