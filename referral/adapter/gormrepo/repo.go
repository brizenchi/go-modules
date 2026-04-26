package gormrepo

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/brizenchi/go-modules/referral/domain"
	"github.com/brizenchi/go-modules/referral/port"
	"gorm.io/gorm"
)

// CodeRepo is the GORM implementation of port.CodeRepository.
type CodeRepo struct{ db *gorm.DB }

func NewCodeRepo(db *gorm.DB) *CodeRepo { return &CodeRepo{db: db} }

func (r *CodeRepo) FindByUser(ctx context.Context, userID string) (*domain.Code, error) {
	var row codeRow
	res := r.db.WithContext(ctx).Where("user_id = ?", strings.TrimSpace(userID)).Limit(1).Find(&row)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, domain.ErrNotFound
	}
	c := row.toDomain()
	return &c, nil
}

func (r *CodeRepo) FindByValue(ctx context.Context, value string) (*domain.Code, error) {
	var row codeRow
	res := r.db.WithContext(ctx).Where("value = ?", strings.TrimSpace(value)).Limit(1).Find(&row)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, domain.ErrNotFound
	}
	c := row.toDomain()
	return &c, nil
}

func (r *CodeRepo) Create(ctx context.Context, c domain.Code) error {
	row := codeRow{UserID: c.UserID, Value: c.Value}
	err := r.db.WithContext(ctx).Create(&row).Error
	if err != nil && isUniqueViolation(err) {
		return domain.ErrCodeCollision
	}
	return err
}

var _ port.CodeRepository = (*CodeRepo)(nil)

// ReferralRepo is the GORM implementation of port.ReferralRepository.
type ReferralRepo struct{ db *gorm.DB }

func NewReferralRepo(db *gorm.DB) *ReferralRepo { return &ReferralRepo{db: db} }

func (r *ReferralRepo) FindByReferee(ctx context.Context, refereeID string) (*domain.Referral, error) {
	var row referralRow
	res := r.db.WithContext(ctx).Where("referee_id = ?", strings.TrimSpace(refereeID)).Limit(1).Find(&row)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, domain.ErrNotFound
	}
	d := row.toDomain()
	return &d, nil
}

func (r *ReferralRepo) Create(ctx context.Context, ref domain.Referral) (*domain.Referral, error) {
	if ref.Status == "" {
		ref.Status = domain.StatusPending
	}
	row := referralFromDomain(ref)
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, domain.ErrAlreadyAttributed
		}
		return nil, err
	}
	d := row.toDomain()
	return &d, nil
}

func (r *ReferralRepo) Activate(ctx context.Context, refereeID string, rewardCredits int) (*domain.Referral, error) {
	var row referralRow
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Where("referee_id = ?", strings.TrimSpace(refereeID)).Limit(1).Find(&row)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return domain.ErrNotFound
		}
		if row.Status == string(domain.StatusActivated) {
			return domain.ErrAlreadyActivated
		}
		now := time.Now().UTC()
		row.Status = string(domain.StatusActivated)
		row.ActivatedAt = &now
		row.RewardCredits = rewardCredits
		return tx.Save(&row).Error
	})
	if err != nil {
		return nil, err
	}
	d := row.toDomain()
	return &d, nil
}

func (r *ReferralRepo) ListByReferrer(ctx context.Context, referrerID string, page, limit int) ([]domain.Referral, int, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}

	var total int64
	if err := r.db.WithContext(ctx).Model(&referralRow{}).Where("referrer_id = ?", referrerID).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []referralRow
	err := r.db.WithContext(ctx).
		Where("referrer_id = ?", referrerID).
		Order("created_at DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	out := make([]domain.Referral, len(rows))
	for i, row := range rows {
		out[i] = row.toDomain()
	}
	return out, int(total), nil
}

func (r *ReferralRepo) StatsByReferrer(ctx context.Context, referrerID string) (*domain.Stats, error) {
	type aggRow struct {
		Status string
		Count  int64
		Reward int64
	}
	var rows []aggRow
	err := r.db.WithContext(ctx).
		Model(&referralRow{}).
		Select("status, COUNT(*) AS count, COALESCE(SUM(reward_credits),0) AS reward").
		Where("referrer_id = ?", referrerID).
		Group("status").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	stats := &domain.Stats{}
	for _, row := range rows {
		stats.TotalReferred += int(row.Count)
		switch domain.Status(row.Status) {
		case domain.StatusActivated:
			stats.Activated += int(row.Count)
		case domain.StatusPending:
			stats.Pending += int(row.Count)
		}
		stats.TotalRewardCredits += int(row.Reward)
	}
	return stats, nil
}

var _ port.ReferralRepository = (*ReferralRepo)(nil)

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "duplicate") || strings.Contains(s, "unique") || strings.Contains(s, "23505")
}
