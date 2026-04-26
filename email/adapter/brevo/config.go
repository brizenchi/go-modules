// Package brevo is a Sender adapter for the Brevo (formerly Sendinblue)
// transactional email API.
package brevo

import (
	"github.com/brizenchi/go-modules/email/domain"
)

// DefaultBaseURL is the public Brevo API base URL.
const DefaultBaseURL = "https://api.brevo.com/v3"

// Config holds the static configuration for a Brevo sender.
//
// Populate from your config layer (viper, env, ...) — the adapter does
// not read any config source itself.
type Config struct {
	APIKey     string
	PartnerKey string         // optional
	BaseURL    string         // optional, defaults to DefaultBaseURL
	Sender     domain.Address // default From address
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
