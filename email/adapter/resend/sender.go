package resend

import (
	"context"
	"fmt"
	"net/url"

	"github.com/brizenchi/go-modules/email/domain"
	"github.com/brizenchi/go-modules/email/port"
	resendSDK "github.com/resend/resend-go/v2"
)

// Sender implements port.Sender against Resend's transactional API.
type Sender struct {
	cfg    Config
	client *resendSDK.Client
}

// New constructs a Resend Sender. Returns an error if cfg is invalid.
func New(cfg Config) (*Sender, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	client := resendSDK.NewClient(cfg.APIKey)
	if cfg.BaseURL != "" {
		u, err := url.Parse(cfg.BaseURL)
		if err != nil {
			return nil, fmt.Errorf("resend: invalid base url: %w", err)
		}
		client.BaseURL = u
	}
	return &Sender{cfg: cfg, client: client}, nil
}

func (s *Sender) Name() string { return "resend" }

// Send delivers a Message via Resend.
//
// Resend does not have a server-side template engine; if msg.TemplateRef
// is set, the call fails with ErrTemplateNotFound so the misconfiguration
// surfaces immediately.
func (s *Sender) Send(ctx context.Context, msg *domain.Message) (*domain.Receipt, error) {
	if err := msg.Validate(); err != nil {
		return nil, err
	}
	if msg.TemplateRef != "" {
		return nil, fmt.Errorf("%w: resend has no server-side templates; render locally before sending", domain.ErrTemplateNotFound)
	}

	req := s.buildSendRequest(msg)
	resp, err := s.client.Emails.SendWithContext(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrSendFailed, err)
	}

	status := domain.StatusSent
	if msg.ScheduledAt != nil {
		status = domain.StatusScheduled
	}
	return &domain.Receipt{
		MessageID: resp.Id,
		Status:    status,
	}, nil
}

var _ port.Sender = (*Sender)(nil)
