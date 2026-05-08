package repo

import (
	"context"
	"strings"

	"gorm.io/gorm"
)

type userSummaryRow struct {
	ID    string `gorm:"column:id"`
	Email string `gorm:"column:email"`
	Plan  string `gorm:"column:plan"`
}

func loadUserSummaryByID(ctx context.Context, db *gorm.DB, userID string) (*userSummaryRow, error) {
	var row userSummaryRow
	if err := db.WithContext(ctx).
		Table("users").
		Select("id", "email", "plan").
		Where("id = ?", strings.TrimSpace(userID)).
		Take(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func loadUserIDByEmail(ctx context.Context, db *gorm.DB, email string) (string, error) {
	var row userSummaryRow
	if err := db.WithContext(ctx).
		Table("users").
		Select("id").
		Where("email = ?", strings.ToLower(strings.TrimSpace(email))).
		Take(&row).Error; err != nil {
		return "", err
	}
	return row.ID, nil
}
