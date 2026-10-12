// Package billingrecovery repairs missed deliveries and retries failed listeners.
package billingrecovery

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/brizenchi/go-modules/modules/billing"
	"github.com/brizenchi/go-modules/modules/billing/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// State is both a cross-replica lease and a durable provider pagination cursor.
type State struct {
	ID             string `gorm:"primaryKey;size:64"`
	LeaseUntil     time.Time
	From           time.Time
	Until          time.Time
	Cursor         string
	RetryCursor    uint
	SnapshotCursor uint
}

func (State) TableName() string { return "billing_recovery_states" }

type feed interface {
	ListRecoveryEvents(context.Context, time.Time, time.Time, string) ([]string, bool, error)
}

func Run(ctx context.Context, db *gorm.DB, m *billing.Module) error {
	if db == nil || m == nil || !m.Provider.Enabled() {
		return nil
	}
	source, ok := m.Provider.(feed)
	if !ok {
		return nil
	}
	// Lease exceeds the bounded execution deadline; a crashed worker is recoverable.
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	now := time.Now().UTC()
	key := m.Provider.Name()
	seed := State{ID: key, LeaseUntil: now.Add(-time.Hour), From: now.Add(-24 * time.Hour)}
	if err := db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&seed).Error; err != nil {
		return err
	}
	lease := now.Add(2 * time.Minute)
	lock := db.WithContext(ctx).Model(&State{}).Where("id = ? AND lease_until < ?", key, now).Update("lease_until", lease)
	if lock.Error != nil {
		return lock.Error
	}
	if lock.RowsAffected == 0 {
		return nil
	}
	var state State
	if err := db.WithContext(ctx).Where("id = ?", key).Take(&state).Error; err != nil {
		return err
	}
	owned := func() *gorm.DB {
		return db.WithContext(ctx).Model(&State{}).Where("id = ? AND lease_until = ?", key, lease)
	}
	var failures []error
	// Cycle over pending rows; a permanently failing old event cannot starve new ones.
	var rows []domain.BillingEvent
	if err := db.WithContext(ctx).Where("provider = ? AND processed = ? AND id > ?", key, false, state.RetryCursor).Order("id ASC").Limit(20).Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if _, err := m.Webhook.Replay(ctx, row.ProviderEventID); err != nil {
			failures = append(failures, fmt.Errorf("retry event %s: %w", row.ProviderEventID, err))
		}
		state.RetryCursor = row.ID
	}
	if len(rows) < 20 {
		state.RetryCursor = 0
	}
	if state.Until.IsZero() {
		state.Until = now.Add(-time.Minute)
	}
	// Refresh canonical dates/status even if an original callback was incomplete.
	var subscriptions []domain.BillingSubscription
	if err := db.WithContext(ctx).Where("provider = ? AND id > ? AND provider_subscription_id <> ?", key, state.SnapshotCursor, "").Order("id ASC").Limit(10).Find(&subscriptions).Error; err != nil {
		return err
	}
	for _, row := range subscriptions {
		observedAt := time.Now().UTC().Truncate(time.Second)
		snapshot, err := m.Provider.GetSubscription(ctx, row.ProviderSubscriptionID)
		if err == nil && snapshot != nil && m.Subscriptions != nil {
			_, err = m.Subscriptions.UpsertSnapshot(ctx, row.UserID, key, *snapshot, observedAt, "reconcile:"+row.ProviderSubscriptionID)
		}
		if err != nil {
			failures = append(failures, fmt.Errorf("refresh subscription %s: %w", row.ProviderSubscriptionID, err))
		}
		state.SnapshotCursor = row.ID
	}
	if len(subscriptions) < 10 {
		state.SnapshotCursor = 0
	}
	if err := owned().Updates(map[string]any{"until": state.Until, "retry_cursor": state.RetryCursor, "snapshot_cursor": state.SnapshotCursor}).Error; err != nil {
		return err
	}
	ids, more, err := source.ListRecoveryEvents(ctx, state.From, state.Until, state.Cursor)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if _, err := m.Webhook.Replay(ctx, id); err != nil {
			var saved domain.BillingEvent
			if db.WithContext(ctx).Where("provider = ? AND stripe_event_id = ?", key, id).Take(&saved).Error != nil {
				return fmt.Errorf("recover event %s: %w", id, err)
			}
			failures = append(failures, fmt.Errorf("recover event %s: %w", id, err))
		}
		// Checkpoint each event so a deadline cannot repeatedly restart a busy page.
		if err := owned().Update("cursor", id).Error; err != nil {
			return err
		}
	}
	if more && len(ids) > 0 {
		state.Cursor = ids[len(ids)-1]
	} else {
		state.From = state.Until.Add(-time.Minute)
		state.Until = time.Time{}
		state.Cursor = ""
	}
	err = owned().Updates(map[string]any{"from": state.From, "until": state.Until, "cursor": state.Cursor, "retry_cursor": state.RetryCursor, "snapshot_cursor": state.SnapshotCursor, "lease_until": now}).Error
	return errors.Join(append(failures, err)...)
}
