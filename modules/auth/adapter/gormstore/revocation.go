package gormstore

import (
	"context"
	"time"

	"github.com/brizenchi/go-modules/modules/auth/port"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type tokenRevocationRow struct {
	TokenHash string    `gorm:"primaryKey;type:char(64)"`
	ExpiresAt time.Time `gorm:"not null;index:idx_auth_token_revocations_expires_at"`
}

func (tokenRevocationRow) TableName() string { return "auth_token_revocations" }

func (s *Store) IsRevoked(ctx context.Context, tokenHash string) (bool, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&tokenRevocationRow{}).
		Where("token_hash = ? AND expires_at > ?", tokenHash, time.Now().UTC()).Count(&count).Error
	return count > 0, err
}

func (s *Store) RevokeToken(ctx context.Context, tokenHash string, expiresAt time.Time) error {
	return s.db.WithContext(ctx).Transaction(func(transaction *gorm.DB) error {
		if err := transaction.Where("expires_at <= ?", time.Now().UTC()).Delete(&tokenRevocationRow{}).Error; err != nil {
			return err
		}
		return transaction.Clauses(clause.OnConflict{DoNothing: true}).Create(&tokenRevocationRow{
			TokenHash: tokenHash, ExpiresAt: expiresAt.UTC(),
		}).Error
	})
}

var _ port.TokenRevocationStore = (*Store)(nil)
