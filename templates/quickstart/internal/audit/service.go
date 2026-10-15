// Package audit records administrator operations independently of feature handlers.
// Mutable retry reservations and immutable outcome events have separate tables.
package audit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrConflict = errors.New("audit: idempotency key reused")

// Operation retains the existing table and API identifiers during the transition.
type Operation struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ActorID        string    `gorm:"type:varchar(36);not null;index;uniqueIndex:uniq_operations_actor_key" json:"actor_id"`
	IdempotencyKey string    `gorm:"type:varchar(128);not null;uniqueIndex:uniq_operations_actor_key" json:"idempotency_key"`
	Action         string    `gorm:"type:varchar(64);not null;index" json:"action"`
	TargetID       string    `gorm:"type:varchar(255);not null" json:"target_id"`
	Reason         string    `gorm:"type:varchar(500);not null" json:"reason"`
	RequestHash    string    `gorm:"type:varchar(64);not null" json:"-"`
	Status         string    `gorm:"type:varchar(32);not null" json:"status"`
	Details        string    `gorm:"type:text" json:"details"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (Operation) TableName() string { return "operations_audit_events" }

type Event struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	OperationID uint64    `gorm:"not null;uniqueIndex:uniq_audit_outcome" json:"operation_id"`
	Outcome     string    `gorm:"type:varchar(32);not null;uniqueIndex:uniq_audit_outcome" json:"outcome"`
	ActorID     string    `gorm:"type:varchar(36);not null;index" json:"actor_id"`
	Action      string    `gorm:"type:varchar(64);not null;index" json:"action"`
	TargetID    string    `gorm:"type:varchar(255);not null;index" json:"target_id"`
	Reason      string    `gorm:"type:varchar(500);not null" json:"reason"`
	Details     string    `gorm:"type:text" json:"details"`
	CreatedAt   time.Time `json:"created_at"`
}

func (Event) TableName() string { return "audit_outcome_events" }
func Models() []any             { return []any{&Operation{}, &Event{}} }

type Service struct{}

func New() *Service { return &Service{} }
func Hash(value any) string {
	raw, _ := json.Marshal(value)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

var credential = regexp.MustCompile(`\b(?:[sr]k_(?:test|live)_|whsec_|re_)[A-Za-z0-9_\-]+`)

func SafeReason(value string) string {
	return credential.ReplaceAllString(strings.TrimSpace(value), "[redacted]")
}

// Claim must run in the same transaction as the local mutation.
func (*Service) Claim(ctx context.Context, tx *gorm.DB, row *Operation) (bool, error) {
	tx = tx.WithContext(ctx)
	row.Reason = SafeReason(row.Reason)
	result := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "actor_id"}, {Name: "idempotency_key"}}, DoNothing: true}).Create(row)
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected > 0 {
		return true, nil
	}
	var previous Operation
	if err := tx.Where("actor_id = ? AND idempotency_key = ?", row.ActorID, row.IdempotencyKey).Take(&previous).Error; err != nil {
		return false, err
	}
	if previous.RequestHash != row.RequestHash || previous.Action != row.Action || previous.TargetID != row.TargetID {
		return false, ErrConflict
	}
	*row = previous
	return false, nil
}

// Finish appends the outcome and updates its retry reservation atomically. Details
// must be an allowlisted feature DTO, never a request body or provider payload.
func (*Service) Finish(ctx context.Context, tx *gorm.DB, row *Operation, outcome, details string) error {
	if outcome != "succeeded" && outcome != "failed" {
		return errors.New("audit: invalid outcome")
	}
	tx = tx.WithContext(ctx)
	return tx.Transaction(func(db *gorm.DB) error {
		q := db.Model(&Operation{}).Where("id = ?", row.ID)
		if outcome == "failed" {
			q = q.Where("status <> ?", "succeeded")
		}
		result := q.Updates(map[string]any{"status": outcome, "details": details, "updated_at": time.Now().UTC()})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		event := Event{OperationID: row.ID, Outcome: outcome, ActorID: row.ActorID, Action: row.Action, TargetID: row.TargetID, Reason: SafeReason(row.Reason), Details: details, CreatedAt: time.Now().UTC()}
		return db.Clauses(clause.OnConflict{DoNothing: true}).Create(&event).Error
	})
}

// Record is for a transactionally idempotent business action, such as a credit
// ledger write. A stable business key keeps retries a single successful operation.
func (s *Service) Record(ctx context.Context, tx *gorm.DB, row Operation, details any) error {
	raw, err := json.Marshal(details)
	if err != nil {
		return err
	}
	row.Status = "pending"
	created, err := s.Claim(ctx, tx, &row)
	if err != nil {
		return err
	}
	if !created {
		return nil
	}
	return s.Finish(ctx, tx, &row, "succeeded", string(raw))
}
