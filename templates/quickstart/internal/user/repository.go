package user

import (
	"context"
	"strings"
	"time"

	authdomain "github.com/brizenchi/go-modules/modules/auth/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func Models() []any {
	return []any{&User{}, &Identity{}, &CreditGrant{}, &CreditTransaction{}, &CreditLot{}, &CreditAllocation{}}
}

func AutoMigrate(db *gorm.DB) error { return db.AutoMigrate(Models()...) }

type Repository struct {
	db          *gorm.DB
	creditClock func() time.Time
}

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) FindByID(ctx context.Context, userID string) (*User, error) {
	var user User
	err := r.db.WithContext(ctx).Where("id = ?", strings.TrimSpace(userID)).Take(&user).Error
	if err == nil && user.CreditsVersion == 1 {
		var summary CreditSummary
		summary, err = r.GetCreditSummary(ctx, user.ID)
		user.Credits = summary.Balance
	}
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
	if user.Credits < 0 {
		return ErrInvalidCreditOperation
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		user.CreditsVersion = 1
		if err := tx.Create(user).Error; err != nil {
			return err
		}
		if user.Credits == 0 {
			return nil
		}
		entry := CreditTransaction{UserID: user.ID, Kind: CreditKindOpening, Amount: user.Credits, BalanceAfter: user.Credits, Source: "opening", SourceID: user.ID, Reason: "Initial account balance"}
		if err := tx.Create(&entry).Error; err != nil {
			return err
		}
		return tx.Create(&CreditLot{UserID: user.ID, TransactionID: entry.ID, Amount: user.Credits, Remaining: user.Credits}).Error
	})
}

func (r *Repository) Save(ctx context.Context, user *User) error {
	// Login/profile updates must never overwrite a concurrent credit operation.
	return r.db.WithContext(ctx).Omit("credits", "credits_version").Save(user).Error
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

func (r *Repository) DB() *gorm.DB { return r.db }
