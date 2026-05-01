// Package repo contains GORM-backed implementations of modules/billing/port repositories.
//
// These are the default implementations. Hosts may swap them out by
// providing their own port.BillingEventRepository implementation.
package repo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/brizenchi/go-modules/modules/billing/domain"
	"github.com/brizenchi/go-modules/modules/billing/port"
	"gorm.io/gorm"
)

// BillingEventRepo persists domain.BillingEvent in PostgreSQL via GORM.
type BillingEventRepo struct {
	db *gorm.DB
}

func NewBillingEventRepo(db *gorm.DB) *BillingEventRepo {
	return &BillingEventRepo{db: db}
}

// CreateIfAbsent inserts the event, returning the existing row on duplicate.
//
// Concurrency: relies on the unique index on (provider_event_id) — see
// domain.BillingEvent's gorm tags. The unique violation is detected by
// the database, not by a pre-check + insert race.
func (r *BillingEventRepo) CreateIfAbsent(ctx context.Context, e *domain.BillingEvent) (*domain.BillingEvent, bool, error) {
	if e == nil {
		return nil, false, fmt.Errorf("billing: event required")
	}
	e.ProviderEventID = strings.TrimSpace(e.ProviderEventID)
	if e.ProviderEventID == "" {
		return nil, false, fmt.Errorf("billing: provider_event_id required")
	}
	if e.Provider == "" {
		return nil, false, fmt.Errorf("billing: provider required")
	}

	err := r.db.WithContext(ctx).Create(e).Error
	if err == nil {
		return e, true, nil
	}
	if !isUniqueViolation(err) {
		return nil, false, err
	}

	existing, err := r.findByProviderEventID(ctx, e.Provider, e.ProviderEventID)
	if err != nil {
		return nil, false, err
	}
	return existing, false, nil
}

func (r *BillingEventRepo) findByProviderEventID(ctx context.Context, provider, id string) (*domain.BillingEvent, error) {
	var ev domain.BillingEvent
	res := r.db.WithContext(ctx).
		Where("stripe_event_id = ?", id). // column name preserved for backward compat
		Limit(1).
		Find(&ev)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &ev, nil
}

func (r *BillingEventRepo) MarkProcessed(ctx context.Context, provider, providerEventID string) error {
	now := time.Now().UTC()
	res := r.db.WithContext(ctx).
		Model(&domain.BillingEvent{}).
		Where("stripe_event_id = ?", strings.TrimSpace(providerEventID)).
		Updates(map[string]any{
			"processed":    true,
			"processed_at": &now,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("billing: event not found")
	}
	return nil
}

// isUniqueViolation reports whether the error is a duplicate-key violation.
//
// We use string matching to avoid hard-coding a specific driver dependency.
// Postgres returns "23505 unique_violation"; MySQL returns "Error 1062";
// SQLite returns "UNIQUE constraint failed".
func isUniqueViolation(err error) bool {
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "duplicate") ||
		strings.Contains(s, "unique") ||
		strings.Contains(s, "23505")
}

// Compile-time check.
var _ port.BillingEventRepository = (*BillingEventRepo)(nil)
