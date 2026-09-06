package user

import (
	"context"
	"fmt"
	"strings"
	"time"

	authdomain "github.com/brizenchi/go-modules/modules/auth/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func Models() []any { return []any{&User{}, &Identity{}, &CreditGrant{}} }

func AutoMigrate(db *gorm.DB) error { return db.AutoMigrate(Models()...) }

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) FindByID(ctx context.Context, userID string) (*User, error) {
	var user User
	err := r.db.WithContext(ctx).Where("id = ?", strings.TrimSpace(userID)).Take(&user).Error
	return &user, err
}

func (r *Repository) FindByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	err := r.db.WithContext(ctx).Where("email = ?", normalizeEmail(email)).Take(&user).Error
	return &user, err
}

func (r *Repository) FindIdentity(ctx context.Context, provider authdomain.Provider, subject string) (*Identity, error) {
	var identity Identity
	err := r.db.WithContext(ctx).
		Where("provider = ? AND subject = ?", strings.TrimSpace(string(provider)), strings.TrimSpace(subject)).
		Take(&identity).Error
	return &identity, err
}

func (r *Repository) Create(ctx context.Context, user *User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *Repository) Save(ctx context.Context, user *User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

// UpdateProfile changes only the host-owned, user-editable profile fields.
// Email, role, credits, and provider identities are intentionally excluded.
func (r *Repository) UpdateProfile(ctx context.Context, userID string, username, avatarURL *string) (*User, error) {
	updates := make(map[string]any, 3)
	if username != nil {
		updates["username"] = strings.TrimSpace(*username)
	}
	if avatarURL != nil {
		updates["avatar_url"] = strings.TrimSpace(*avatarURL)
	}
	if len(updates) == 0 {
		return r.FindByID(ctx, userID)
	}
	updates["updated_at"] = time.Now().UTC()
	result := r.db.WithContext(ctx).
		Model(&User{}).
		Where("id = ?", strings.TrimSpace(userID)).
		Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected != 1 {
		return nil, gorm.ErrRecordNotFound
	}
	return r.FindByID(ctx, userID)
}

func (r *Repository) LinkIdentity(ctx context.Context, userID string, provider authdomain.Provider, subject string) error {
	identity := &Identity{
		UserID:   strings.TrimSpace(userID),
		Provider: strings.TrimSpace(string(provider)),
		Subject:  strings.TrimSpace(subject),
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "provider"}, {Name: "subject"}},
		DoNothing: true,
	}).Create(identity).Error
}

func (r *Repository) MarkLogin(ctx context.Context, userID string, at time.Time) error {
	return r.db.WithContext(ctx).
		Model(&User{}).
		Where("id = ?", strings.TrimSpace(userID)).
		Update("last_login_at", at.UTC()).Error
}

func (r *Repository) AddCredits(ctx context.Context, userID string, delta int64) error {
	if delta == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Model(&User{}).
		Where("id = ?", strings.TrimSpace(userID)).
		UpdateColumn("credits", gorm.Expr("credits + ?", delta)).Error
}

// GrantCredits applies a credit change exactly once for a stable external
// source key such as ("stripe", "evt_123"). The ledger insert and balance
// update share one database transaction.
func (r *Repository) GrantCredits(ctx context.Context, userID, source, sourceID string, delta int64) error {
	userID = strings.TrimSpace(userID)
	source = strings.ToLower(strings.TrimSpace(source))
	sourceID = strings.TrimSpace(sourceID)
	if delta == 0 {
		return nil
	}
	if userID == "" || source == "" || sourceID == "" {
		return fmt.Errorf("user: user_id, source and source_id required for credit grant")
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		grant := &CreditGrant{UserID: userID, Source: source, SourceID: sourceID, Amount: delta}
		result := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "source"}, {Name: "source_id"}},
			DoNothing: true,
		}).Create(grant)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		result = tx.Model(&User{}).
			Where("id = ?", userID).
			UpdateColumn("credits", gorm.Expr("credits + ?", delta))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func (r *Repository) DB() *gorm.DB { return r.db }
