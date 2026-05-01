package authstore

import (
	"context"
	"errors"
	"strings"
	"time"

	authdomain "github.com/brizenchi/go-modules/modules/auth/domain"
	authport "github.com/brizenchi/go-modules/modules/auth/port"
	"github.com/brizenchi/go-modules/modules/user/adapter/gormrepo"
	userdomain "github.com/brizenchi/go-modules/modules/user/domain"
	"gorm.io/gorm"
)

// Store adapts the shared user repo to auth.port.UserStore.
type Store struct {
	users *gormrepo.Repo
}

func New(users *gormrepo.Repo) *Store { return &Store{users: users} }

func (s *Store) FindByEmail(ctx context.Context, email string) (*authdomain.Identity, error) {
	user, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, authdomain.ErrUserNotFound
		}
		return nil, err
	}
	return toIdentity(user, false), nil
}

func (s *Store) FindOrCreateByEmail(ctx context.Context, email string) (*authdomain.Identity, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	user, err := s.users.FindByEmail(ctx, email)
	if err == nil {
		now := time.Now().UTC()
		user.EmailVerified = true
		user.EmailVerifiedAt = &now
		user.LastLoginAt = &now
		if strings.TrimSpace(user.Provider) == "" {
			user.Provider = "email"
			user.ProviderSubject = email
		}
		if err := s.users.Save(ctx, user); err != nil {
			return nil, err
		}
		return toIdentity(user, false), nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	now := time.Now().UTC()
	user = &userdomain.User{
		Email:           email,
		EmailVerified:   true,
		EmailVerifiedAt: &now,
		Provider:        "email",
		ProviderSubject: email,
		LastLoginAt:     &now,
		Plan:            userdomain.PlanFree,
	}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}
	return toIdentity(user, true), nil
}

func (s *Store) FindOrCreateFromOAuth(ctx context.Context, p authdomain.OAuthProfile) (*authdomain.Identity, error) {
	provider := string(p.Provider)
	email := strings.ToLower(strings.TrimSpace(p.Email))
	subject := strings.TrimSpace(p.Subject)

	var (
		user *userdomain.User
		err  error
	)
	if subject != "" {
		user, err = s.users.FindByProviderSubject(ctx, provider, subject)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	if user == nil && email != "" {
		user, err = s.users.FindByEmail(ctx, email)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	now := time.Now().UTC()
	if user == nil {
		user = &userdomain.User{
			Email:           email,
			EmailVerified:   true,
			EmailVerifiedAt: &now,
			Username:        p.Username,
			AvatarURL:       p.AvatarURL,
			Provider:        provider,
			ProviderSubject: subject,
			LastLoginAt:     &now,
			Plan:            userdomain.PlanFree,
		}
		if err := s.users.Create(ctx, user); err != nil {
			return nil, err
		}
		return toIdentity(user, true), nil
	}

	updates := map[string]any{
		"last_login_at":     &now,
		"email_verified":    true,
		"email_verified_at": &now,
	}
	if p.Username != "" {
		updates["username"] = p.Username
	}
	if p.AvatarURL != "" {
		updates["avatar_url"] = p.AvatarURL
	}
	if user.Provider == "" {
		updates["provider"] = provider
		updates["provider_subject"] = subject
	}
	if err := s.users.UpdateFields(ctx, user.ID, updates); err != nil {
		return nil, err
	}
	user, err = s.users.FindByID(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	return toIdentity(user, false), nil
}

func (s *Store) FindByID(ctx context.Context, id string) (*authdomain.Identity, error) {
	user, err := s.users.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, authdomain.ErrUserNotFound
		}
		return nil, err
	}
	return toIdentity(user, false), nil
}

func (s *Store) MarkLogin(ctx context.Context, id string) error {
	now := time.Now().UTC()
	return s.users.UpdateFields(ctx, id, map[string]any{"last_login_at": &now})
}

func toIdentity(u *userdomain.User, isNew bool) *authdomain.Identity {
	role := authdomain.RoleUser
	if u.Role == userdomain.RoleAdmin {
		role = authdomain.RoleAdmin
	}
	return &authdomain.Identity{
		UserID:    u.ID,
		Email:     u.Email,
		Username:  u.Username,
		AvatarURL: u.AvatarURL,
		Provider:  authdomain.Provider(u.Provider),
		Subject:   u.ProviderSubject,
		Role:      role,
		IsNew:     isNew,
	}
}

var _ authport.UserStore = (*Store)(nil)

