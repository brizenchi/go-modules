// Package port defines the interfaces the email module depends on.
package port

import (
	"context"

	"github.com/brizenchi/go-modules/email/domain"
)

// Sender is the abstraction over a transactional email backend
// (Brevo, SendGrid, SMTP server, ...).
//
// Implementations must be safe for concurrent use.
type Sender interface {
	// Name returns a short identifier ("brevo", "sendgrid", "smtp", "log").
	Name() string

	// Send delivers a message. The returned Receipt's MessageID and
	// Status fields are best-effort and depend on the provider.
	Send(ctx context.Context, msg *domain.Message) (*domain.Receipt, error)
}

// Renderer fills in Subject + bodies from a local template.
//
// This is OPTIONAL — projects using provider-side templates (e.g.
// Brevo template_id) don't need a Renderer.
type Renderer interface {
	// Render returns subject, HTML body, and plain-text body. Either body
	// may be empty (e.g. an HTML-only template); the caller decides
	// whether that's acceptable.
	Render(ctx context.Context, templateName string, vars map[string]any) (subject, html, text string, err error)
}
