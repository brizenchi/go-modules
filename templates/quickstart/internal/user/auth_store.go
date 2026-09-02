package user

import (
	"context"
	"errors"
	"strings"
	"time"

	authdomain "github.com/brizenchi/go-modules/modules/auth/domain"
	authport "github.com/brizenchi/go-modules/modules/auth/port"
	"gorm.io/gorm"
)

// AuthStore adapts this SaaS's user schema to the shared auth module.
// Changes to User stay here; auth only receives its minimal Identity view.
type AuthStore struct {
	users *Repository
}

func NewAuthStore(users *Repository) *AuthStore { return &AuthStore{users: users} }

func (s *AuthStore) FindByEmail(ctx context.Context, email string) (*authdomain.Identity, error) {
	user, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, authdomain.ErrUserNotFound
		}
		return nil, err
	}
	return toAuthIdentity(user, false, "", ""), nil
}

func (s *AuthStore) FindOrCreateByEmail(ctx context.Context, email string) (*authdomain.Identity, error) {
	email = normalizeEmail(email)
	user, err := s.users.FindByEmail(ctx, email)
	switch {
	case err == nil:
		now := time.Now().UTC()
		user.EmailVerified = true
		user.EmailVerifiedAt = &now
		if err := s.users.Save(ctx, user); err != nil {
			return nil, err
		}
		return toAuthIdentity(user, false, authdomain.ProviderEmail, email), nil
	case !errors.Is(err, gorm.ErrRecordNotFound):
		return nil, err
	}

	now := time.Now().UTC()
	user = &User{Email: email, EmailVerified: true, EmailVerifiedAt: &now, LastLoginAt: &now}
	if err := s.users.Create(ctx, user); err != nil {
		// Another request may have created the same normalized email first.
		existing, findErr := s.users.FindByEmail(ctx, email)
		if findErr != nil {
			return nil, err
		}
		return toAuthIdentity(existing, false, authdomain.ProviderEmail, email), nil
	}
	return toAuthIdentity(user, true, authdomain.ProviderEmail, email), nil
}

func (s *AuthStore) FindOrCreateFromOAuth(ctx context.Context, profile authdomain.OAuthProfile) (*authdomain.Identity, error) {
	provider := profile.Provider
	subject := strings.TrimSpace(profile.Subject)
	email := normalizeEmail(profile.Email)

	if subject != "" {
		link, err := s.users.FindIdentity(ctx, provider, subject)
		switch {
		case err == nil:
			user, err := s.users.FindByID(ctx, link.UserID)
			if err != nil {
				return nil, err
			}
			updateFromOAuth(user, profile)
			if err := s.users.Save(ctx, user); err != nil {
				return nil, err
			}
			return toAuthIdentity(user, false, provider, subject), nil
		case !errors.Is(err, gorm.ErrRecordNotFound):
			return nil, err
		}
	}

	user, err := s.users.FindByEmail(ctx, email)
	isNew := false
	switch {
	case err == nil:
	case !errors.Is(err, gorm.ErrRecordNotFound):
		return nil, err
	default:
		user = &User{Email: email}
		isNew = true
	}
	updateFromOAuth(user, profile)
	if isNew {
		if err := s.users.Create(ctx, user); err != nil {
			return nil, err
		}
	} else if err := s.users.Save(ctx, user); err != nil {
		return nil, err
	}
	if subject != "" {
		if err := s.users.LinkIdentity(ctx, user.ID, provider, subject); err != nil {
			return nil, err
		}
	}
	return toAuthIdentity(user, isNew, provider, subject), nil
}

func (s *AuthStore) FindByID(ctx context.Context, userID string) (*authdomain.Identity, error) {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, authdomain.ErrUserNotFound
		}
		return nil, err
	}
	return toAuthIdentity(user, false, "", ""), nil
}

func (s *AuthStore) MarkLogin(ctx context.Context, userID string) error {
	return s.users.MarkLogin(ctx, userID, time.Now().UTC())
}

func updateFromOAuth(user *User, profile authdomain.OAuthProfile) {
	now := time.Now().UTC()
	user.Email = normalizeEmail(profile.Email)
	user.EmailVerified = true
	user.EmailVerifiedAt = &now
	user.LastLoginAt = &now
	if username := strings.TrimSpace(profile.Username); username != "" {
		user.Username = username
	}
	if avatar := strings.TrimSpace(profile.AvatarURL); avatar != "" {
		user.AvatarURL = avatar
	}
}

func toAuthIdentity(user *User, isNew bool, provider authdomain.Provider, subject string) *authdomain.Identity {
	return &authdomain.Identity{
		UserID:    user.ID,
		Email:     user.Email,
		Username:  user.Username,
		AvatarURL: user.AvatarURL,
		Provider:  provider,
		Subject:   subject,
		Role:      authdomain.Role(normalizeRole(user.Role)),
		IsNew:     isNew,
	}
}

var _ authport.UserStore = (*AuthStore)(nil)
