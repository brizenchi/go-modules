// Package email_glue wires the email module from project config.
//
// Falls back to the log sender (no real email) when no provider is
// configured — that lets you boot the service in environments without
// email credentials.
package email_glue

import (
	"log/slog"

	"github.com/brizenchi/go-modules/email"
	"github.com/brizenchi/go-modules/email/adapter/brevo"
	logsender "github.com/brizenchi/go-modules/email/adapter/log"
	"github.com/brizenchi/go-modules/email/domain"
)

// Config holds the per-environment email settings.
type Config struct {
	BrevoAPIKey      string
	BrevoSenderEmail string
	BrevoSenderName  string

	// VerificationTplRef is the provider-side template id used for
	// passwordless email-code login. Optional — only required if your
	// auth flow uses email codes.
	VerificationTplRef string
}

// VerificationTemplateRef is a package-level shortcut so other glue
// packages can reach the configured template id.
var VerificationTemplateRef string

// Init returns the wired email Module. Stash the result in a package-
// level var if other glue packages need to send mail too.
func Init(cfg Config) *email.Module {
	VerificationTemplateRef = cfg.VerificationTplRef

	if cfg.BrevoAPIKey != "" && cfg.BrevoSenderEmail != "" {
		sender, err := brevo.New(brevo.Config{
			APIKey: cfg.BrevoAPIKey,
			Sender: domain.Address{
				Email: cfg.BrevoSenderEmail,
				Name:  cfg.BrevoSenderName,
			},
		})
		if err != nil {
			slog.Warn("email_glue: brevo init failed, falling back to log sender", "error", err)
			return email.New(logsender.New(nil), nil)
		}
		slog.Info("email_glue: brevo sender registered")
		return email.New(sender, nil)
	}
	slog.Info("email_glue: no provider configured, using log sender")
	return email.New(logsender.New(nil), nil)
}
