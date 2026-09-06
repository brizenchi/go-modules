package account

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/brizenchi/quickstart-template/internal/user"
)

var (
	errProfileFieldsRequired = errors.New("at least one of username or avatar_url is required")
	errUsernameTooLong       = errors.New("username must not exceed 100 characters")
	errAvatarURLTooLong      = errors.New("avatar_url must not exceed 512 characters")
	errAvatarURLInvalid      = errors.New("avatar_url must be an absolute http(s) URL")
)

type profilePatch struct {
	Username  *string
	AvatarURL *string
}

type profileStore interface {
	FindByID(ctx context.Context, userID string) (*user.User, error)
	UpdateProfile(ctx context.Context, userID string, username, avatarURL *string) (*user.User, error)
}

type service struct {
	users profileStore
}

func newService(users profileStore) *service {
	return &service{users: users}
}

func (s *service) Get(ctx context.Context, userID string) (*user.User, error) {
	return s.users.FindByID(ctx, strings.TrimSpace(userID))
}

func (s *service) Update(ctx context.Context, userID string, patch profilePatch) (*user.User, error) {
	if patch.Username == nil && patch.AvatarURL == nil {
		return nil, errProfileFieldsRequired
	}
	if patch.Username != nil {
		normalized := strings.TrimSpace(*patch.Username)
		if utf8.RuneCountInString(normalized) > 100 {
			return nil, errUsernameTooLong
		}
		patch.Username = &normalized
	}
	if patch.AvatarURL != nil {
		normalized := strings.TrimSpace(*patch.AvatarURL)
		if utf8.RuneCountInString(normalized) > 512 {
			return nil, errAvatarURLTooLong
		}
		if normalized != "" {
			parsed, err := url.ParseRequestURI(normalized)
			if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil {
				return nil, errAvatarURLInvalid
			}
		}
		patch.AvatarURL = &normalized
	}
	return s.users.UpdateProfile(ctx, strings.TrimSpace(userID), patch.Username, patch.AvatarURL)
}

func isValidationError(err error) bool {
	return errors.Is(err, errProfileFieldsRequired) ||
		errors.Is(err, errUsernameTooLong) ||
		errors.Is(err, errAvatarURLTooLong) ||
		errors.Is(err, errAvatarURLInvalid)
}
