package domain

import "errors"

var (
	ErrInvalidEmail       = errors.New("auth: invalid email")
	ErrInvalidCode        = errors.New("auth: invalid or expired verification code")
	ErrCodeRateLimited    = errors.New("auth: too many code requests")
	ErrCodeMaxAttempts    = errors.New("auth: too many failed attempts, request a new code")
	ErrInvalidExchange    = errors.New("auth: invalid or expired exchange code")
	ErrInvalidToken       = errors.New("auth: invalid or expired token")
	ErrInvalidWSTicket    = errors.New("auth: invalid or expired websocket ticket")
	ErrUserNotFound       = errors.New("auth: user not found")
	ErrProviderUnavailable = errors.New("auth: identity provider not configured")
	ErrInvalidState       = errors.New("auth: invalid oauth state")
	ErrUnauthorized       = errors.New("auth: unauthorized")
)
