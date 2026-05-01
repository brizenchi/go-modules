// Package resend is a Sender adapter for the Resend transactional email API.
//
// Resend has no server-side template engine; pair this adapter with a
// port.Renderer (e.g. email/adapter/gotemplate) when you need templating.
// A non-empty domain.Message.TemplateRef is rejected to fail loudly
// rather than silently dropping the template.
package resend

import (
	"github.com/brizenchi/go-modules/modules/email/domain"
)

// Config holds the static configuration for a Resend sender.
//
// Populate from your config layer (viper, env, ...) — the adapter does
// not read any config source itself.
type Config struct {
	APIKey  string
	BaseURL string         // optional; SDK default is used when empty
	Sender  domain.Address // default From address
}

// Validate enforces the minimum invariants.
func (c Config) Validate() error {
	if c.APIKey == "" {
		return domain.ErrInvalidAPIKey
	}
	if c.Sender.Email == "" {
		return domain.ErrInvalidSender
	}
	return nil
}
