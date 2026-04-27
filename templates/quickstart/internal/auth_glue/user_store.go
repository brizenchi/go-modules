package auth_glue

import (
	"context"
	"errors"
	"strings"
	"time"

	authdomain "github.com/brizenchi/go-modules/auth/domain"
	"github.com/brizenchi/go-modules/auth/port"
	"gorm.io/gorm"
)

// User is the project's User model — replace with whatever schema you
// already have. Only `ID` and `Email` are needed by the auth module;
// other fields are project-specific and the auth module never reads them.
type User struct {
	ID          string `gorm:"primaryKey;type:varchar(36)"`
	Email       string `gorm:"uniqueIndex;type:varchar(255);not null"`
	Username    string `gorm:"type:varchar(64)"`
	AvatarURL   string `gorm:"type:varchar(512)"`
	Provider    string `gorm:"type:varchar(32)"`
	Subject     string `gorm:"type:varchar(255);index"`
	Role        string `gorm:"type:varchar(16);default:user"`
	LastLoginAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// userStore implements auth.port.UserStore. Adapt the bodies if your
// User model has different field names.
type userStore struct{ db *gorm.DB }

func (s *userStore) FindByEmail(ctx context.Context, email string) (*authdomain.Identity, error) {
	var u User
	res := s.db.WithContext(ctx).Where("email = ?", strings.ToLower(strings.TrimSpace(email))).Limit(1).Find(&u)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, authdomain.ErrUserNotFound
	}
	return toIdentity(&u, false), nil
}

func (s *userStore) FindOrCreateByEmail(ctx context.Context, email string) (*authdomain.Identity, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	var u User
	res := s.db.WithContext(ctx).Where("email = ?", email).Limit(1).Find(&u)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected > 0 {
		return toIdentity(&u, false), nil
	}
	now := time.Now().UTC()
	u = User{
		ID:          newID(),
		Email:       email,
		Provider:    "email",
		Subject:     email,
		LastLoginAt: &now,
	}
	if err := s.db.WithContext(ctx).Create(&u).Error; err != nil {
		return nil, err
	}
	return toIdentity(&u, true), nil
}

func (s *userStore) FindOrCreateFromOAuth(ctx context.Context, p authdomain.OAuthProfile) (*authdomain.Identity, error) {
	provider := string(p.Provider)
	email := strings.ToLower(strings.TrimSpace(p.Email))
	subject := strings.TrimSpace(p.Subject)

	var u User
	if subject != "" {
		s.db.WithContext(ctx).Where("provider = ? AND subject = ?", provider, subject).Limit(1).Find(&u)
	}
	if u.ID == "" && email != "" {
		s.db.WithContext(ctx).Where("email = ?", email).Limit(1).Find(&u)
	}

	now := time.Now().UTC()
	if u.ID == "" {
		u = User{
			ID:          newID(),
			Email:       email,
			Username:    p.Username,
			AvatarURL:   p.AvatarURL,
			Provider:    provider,
			Subject:     subject,
			LastLoginAt: &now,
		}
		if err := s.db.WithContext(ctx).Create(&u).Error; err != nil {
			return nil, err
		}
		return toIdentity(&u, true), nil
	}
	updates := map[string]any{"last_login_at": &now}
	if p.Username != "" {
		updates["username"] = p.Username
	}
	if p.AvatarURL != "" {
		updates["avatar_url"] = p.AvatarURL
	}
	if u.Provider == "" {
		updates["provider"] = provider
		updates["subject"] = subject
	}
	if err := s.db.WithContext(ctx).Model(&u).Updates(updates).Error; err != nil {
		return nil, err
	}
	return toIdentity(&u, false), nil
}

func (s *userStore) FindByID(ctx context.Context, id string) (*authdomain.Identity, error) {
	var u User
	res := s.db.WithContext(ctx).Where("id = ?", strings.TrimSpace(id)).Limit(1).Find(&u)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, authdomain.ErrUserNotFound
	}
	return toIdentity(&u, false), nil
}

func (s *userStore) MarkLogin(ctx context.Context, id string) error {
	now := time.Now().UTC()
	return s.db.WithContext(ctx).Model(&User{}).
		Where("id = ?", id).
		Update("last_login_at", &now).Error
}

func toIdentity(u *User, isNew bool) *authdomain.Identity {
	role := authdomain.RoleUser
	if u.Role == string(authdomain.RoleAdmin) {
		role = authdomain.RoleAdmin
	}
	return &authdomain.Identity{
		UserID:    u.ID,
		Email:     u.Email,
		Username:  u.Username,
		AvatarURL: u.AvatarURL,
		Provider:  authdomain.Provider(u.Provider),
		Subject:   u.Subject,
		Role:      role,
		IsNew:     isNew,
	}
}

// roleResolver assigns admin role based on `auth.admin_emails` config.
type roleResolver struct{}

func (r *roleResolver) Resolve(_ context.Context, id authdomain.Identity) (authdomain.Role, error) {
	if isAdminEmail(id.Email) {
		return authdomain.RoleAdmin, nil
	}
	if id.Role != "" {
		return id.Role, nil
	}
	return authdomain.RoleUser, nil
}

func isAdminEmail(email string) bool {
	// Replace with viper.GetStringSlice("auth.admin_emails") in your project.
	_ = email
	return false
}

// newID returns a fresh user id. Replace with your project's id strategy
// (UUIDv4, KSUID, ULID, monotonic ints, ...).
func newID() string {
	// Placeholder: time-based — DO NOT use in production. Use uuid.NewString()
	// or an id package. Imported here to avoid forcing a uuid dep on the
	// quickstart for a stub.
	return "u-" + time.Now().UTC().Format("20060102150405.000000000")
}

var _ port.UserStore = (*userStore)(nil)
var _ port.RoleResolver = (*roleResolver)(nil)
var _ = errors.New // keep import if list of helpers shrinks
