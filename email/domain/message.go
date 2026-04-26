// Package domain holds the provider-agnostic types for the email module.
package domain

import "time"

// Address is an email address with an optional display name.
type Address struct {
	Name  string
	Email string
}

// Attachment is a single email attachment.
//
// Exactly one of Content or URL should be set: Content for inline bytes
// (the adapter is responsible for base64-encoding when needed), URL for
// remote attachments (provider-supported only).
type Attachment struct {
	Name        string
	ContentType string // optional, e.g. "application/pdf"
	Content     []byte
	URL         string
}

// Message is a provider-agnostic outgoing email.
//
// Two send modes:
//   - Inline content: set Subject + at least one of HTMLBody/TextBody.
//   - Provider template: set TemplateRef + Variables; Subject/Body
//     are derived by the provider (e.g. Brevo template).
type Message struct {
	From    Address
	To      []Address
	Cc      []Address
	Bcc     []Address
	ReplyTo *Address

	Subject  string
	HTMLBody string
	TextBody string

	// TemplateRef references a provider-side template id (Brevo) or a
	// local template name (when paired with a Renderer). Empty disables
	// templating.
	TemplateRef string
	Variables   map[string]any

	Headers []Header
	Tags    []string
	Attachments []Attachment

	// ScheduledAt, if set, requests delayed delivery (provider-supported only).
	ScheduledAt *time.Time
}

// Header is a single custom email header.
type Header struct {
	Name  string
	Value string
}

// Receipt is the result of a successful send.
type Receipt struct {
	MessageID string // provider-issued id, may be empty
	Status    Status
}

// Status enumerates send outcomes.
type Status string

const (
	StatusSent      Status = "sent"
	StatusQueued    Status = "queued"
	StatusScheduled Status = "scheduled"
)
