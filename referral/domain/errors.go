package domain

import "errors"

var (
	ErrInvalidUser       = errors.New("referral: invalid user_id")
	ErrInvalidCode       = errors.New("referral: invalid or unknown code")
	ErrSelfReferral      = errors.New("referral: cannot refer yourself")
	ErrAlreadyAttributed = errors.New("referral: this user already has a referrer")
	ErrCodeCollision     = errors.New("referral: code collision after retries")
	ErrNotFound          = errors.New("referral: not found")
	ErrAlreadyActivated  = errors.New("referral: already activated")
)
