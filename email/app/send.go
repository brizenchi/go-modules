// Package app contains the email module's use cases.
package app

import (
	"context"
	"fmt"

	"github.com/brizenchi/go-modules/email/domain"
	"github.com/brizenchi/go-modules/email/port"
)

// SendService is the primary entry point: send any pre-built Message.
//
// SendTemplate is a higher-level helper that uses a Renderer to build
// Subject + bodies from a local template before delegating to Send.
type SendService struct {
	sender   port.Sender
	renderer port.Renderer // optional
}

// NewSendService wires a Sender (required) and a Renderer (optional).
func NewSendService(s port.Sender, r port.Renderer) *SendService {
	return &SendService{sender: s, renderer: r}
}

// Send delivers a fully-built message.
func (s *SendService) Send(ctx context.Context, msg *domain.Message) (*domain.Receipt, error) {
	if msg == nil {
		return nil, fmt.Errorf("%w: message is nil", domain.ErrInvalidRecipient)
	}
	if err := msg.Validate(); err != nil {
		return nil, err
	}
	return s.sender.Send(ctx, msg)
}

// SendTemplate renders a local template and sends the resulting message.
//
// Use this when the Sender does NOT support provider-side templates, or
// when you want template logic (a/b tests, conditionals) to live in
// your codebase rather than in the provider's UI.
func (s *SendService) SendTemplate(ctx context.Context, in TemplateMessage) (*domain.Receipt, error) {
	if s.renderer == nil {
		return nil, fmt.Errorf("%w: SendService has no renderer configured", domain.ErrTemplateNotFound)
	}
	if in.Template == "" {
		return nil, fmt.Errorf("%w: template name required", domain.ErrTemplateNotFound)
	}
	subject, html, text, err := s.renderer.Render(ctx, in.Template, in.Variables)
	if err != nil {
		return nil, err
	}
	msg := &domain.Message{
		From:        in.From,
		To:          in.To,
		Cc:          in.Cc,
		Bcc:         in.Bcc,
		ReplyTo:     in.ReplyTo,
		Subject:     subject,
		HTMLBody:    html,
		TextBody:    text,
		Headers:     in.Headers,
		Tags:        in.Tags,
		Attachments: in.Attachments,
	}
	return s.Send(ctx, msg)
}

// TemplateMessage is the input to SendTemplate. Subject + bodies are
// derived from the template; everything else is passed through.
type TemplateMessage struct {
	Template    string
	Variables   map[string]any
	From        domain.Address
	To          []domain.Address
	Cc          []domain.Address
	Bcc         []domain.Address
	ReplyTo     *domain.Address
	Headers     []domain.Header
	Tags        []string
	Attachments []domain.Attachment
}

// SendProviderTemplate is a shortcut for using a provider-side template
// (Brevo template_id) without a local Renderer. The Sender must support
// the TemplateRef format the caller passes (e.g. numeric for Brevo).
func (s *SendService) SendProviderTemplate(ctx context.Context, templateRef string, to []domain.Address, vars map[string]any) (*domain.Receipt, error) {
	msg := &domain.Message{
		To:          to,
		TemplateRef: templateRef,
		Variables:   vars,
	}
	return s.Send(ctx, msg)
}
