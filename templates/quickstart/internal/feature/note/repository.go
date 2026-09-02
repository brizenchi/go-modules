package note

import (
	"context"

	"gorm.io/gorm"
)

// repository is the only layer allowed to touch *gorm.DB.
type repository struct {
	db *gorm.DB
}

func newRepository(db *gorm.DB) *repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, n *Note) error {
	return r.db.WithContext(ctx).Create(n).Error
}

func (r *repository) ListByUser(ctx context.Context, userID string, limit int) ([]Note, error) {
	var rows []Note
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("id DESC").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

func (r *repository) Count(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&Note{}).Count(&n).Error
	return n, err
}
