package domain

import (
	"fmt"
	"strings"
)

// Validate checks the message has the minimum required fields.
//
// Rule: must have at least one recipient, and either a TemplateRef OR
// (Subject + at least one body). Sender (From) is allowed to be empty —
// adapters fill in a default sender from configuration.
func (m *Message) Validate() error {
	if len(m.To) == 0 {
		return fmt.Errorf("%w: missing To", ErrInvalidRecipient)
	}
	for i, addr := range m.To {
		if strings.TrimSpace(addr.Email) == "" {
			return fmt.Errorf("%w: To[%d].Email empty", ErrInvalidRecipient, i)
		}
	}
	if m.TemplateRef != "" {
		return nil
	}
	if strings.TrimSpace(m.Subject) == "" {
		return ErrEmptyContent
	}
	if strings.TrimSpace(m.HTMLBody) == "" && strings.TrimSpace(m.TextBody) == "" {
		return ErrEmptyContent
	}
	return nil
}
