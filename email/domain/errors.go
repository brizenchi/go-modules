package domain

import "errors"

var (
	ErrInvalidAPIKey     = errors.New("email: invalid api key")
	ErrInvalidSender     = errors.New("email: invalid sender")
	ErrInvalidRecipient  = errors.New("email: invalid recipient")
	ErrEmptyContent      = errors.New("email: subject and body are required (or use TemplateRef)")
	ErrTemplateNotFound  = errors.New("email: template not found")
	ErrSenderUnavailable = errors.New("email: no sender configured for this project")
	ErrSendFailed        = errors.New("email: send failed")
	ErrProviderDisabled  = errors.New("email: provider is disabled")
)
