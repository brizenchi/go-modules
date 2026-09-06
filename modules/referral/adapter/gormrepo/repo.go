package gormrepo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/brizenchi/go-modules/modules/referral/domain"
	"github.com/brizenchi/go-modules/modules/referral/port"
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
	var activated bool
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		refereeID = strings.TrimSpace(refereeID)
		// Claim the pending row with a conditional write. Concurrent webhooks
		// must not overwrite the first reward or resurrect an expired invite.
		res := tx.Model(&referralRow{}).
			Where("referee_id = ? AND status = ?", refereeID, string(domain.StatusPending)).
			Where("expires_at IS NULL OR expires_at > ?", now).
			Updates(map[string]any{
				"status": string(domain.StatusActivated), "activated_at": now, "reward_credits": rewardCredits,
			})
		if res.Error != nil {
			return res.Error
		}
		activated = res.RowsAffected > 0
		if !activated {
			if err := tx.Model(&referralRow{}).
				Where("referee_id = ? AND status = ? AND expires_at IS NOT NULL AND expires_at <= ?", refereeID, string(domain.StatusPending), now).
				Updates(map[string]any{"status": string(domain.StatusExpired), "activated_at": nil, "reward_credits": 0}).Error; err != nil {
				return err
			}
		}
		res = tx.Where("referee_id = ?", refereeID).Limit(1).Find(&row)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	d := row.toDomain()
	if activated {
		return &d, nil
	}
	if d.Status == domain.StatusExpired {
		return &d, domain.ErrExpired
	}
	if d.Status == domain.StatusActivated {
		return nil, domain.ErrAlreadyActivated
	}
	return nil, fmt.Errorf("referral: unexpected activation state %q", d.Status)
}

func (r *ReferralRepo) ListByReferrer(ctx context.Context, referrerID string, page, limit int) ([]domain.Referral, int, error) {
	if err := r.expirePending(ctx, referrerID); err != nil {
		return nil, 0, err
	}
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
		Order("created_at DESC, id DESC").
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
	if err := r.expirePending(ctx, referrerID); err != nil {
		return nil, err
	}
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

// expirePending keeps dashboard reads honest even when no qualifying billing
// event arrives after an invitation's deadline. Activation independently
// enforces the same deadline inside its transaction.
func (r *ReferralRepo) expirePending(ctx context.Context, referrerID string) error {
	return r.db.WithContext(ctx).
		Model(&referralRow{}).
		Where(
			"referrer_id = ? AND status = ? AND expires_at IS NOT NULL AND expires_at <= ?",
			strings.TrimSpace(referrerID),
			string(domain.StatusPending),
			time.Now().UTC(),
		).
		Updates(map[string]any{
			"status":         string(domain.StatusExpired),
			"activated_at":   nil,
			"reward_credits": 0,
		}).Error
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
