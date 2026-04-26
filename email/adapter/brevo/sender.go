package brevo

import (
	"context"
	"errors"
	"fmt"

	"github.com/brizenchi/go-modules/email/domain"
	"github.com/brizenchi/go-modules/email/port"
	brevoSDK "github.com/getbrevo/brevo-go/lib"
)

// Sender implements port.Sender against Brevo's transactional API.
type Sender struct {
	cfg    Config
	client *brevoSDK.APIClient
}

// New constructs a Brevo Sender. Returns an error if cfg is invalid.
func New(cfg Config) (*Sender, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	apiCfg := brevoSDK.NewConfiguration()
	apiCfg.AddDefaultHeader("api-key", cfg.APIKey)
	if cfg.PartnerKey != "" {
		apiCfg.AddDefaultHeader("partner-key", cfg.PartnerKey)
	}
	if cfg.BaseURL != "" {
		apiCfg.BasePath = cfg.BaseURL
	} else {
		apiCfg.BasePath = DefaultBaseURL
	}
	return &Sender{cfg: cfg, client: brevoSDK.NewAPIClient(apiCfg)}, nil
}

func (s *Sender) Name() string { return "brevo" }

// Send delivers a Message via Brevo. The Brevo TemplateRef is parsed as
// an integer template id; non-numeric refs are rejected.
func (s *Sender) Send(ctx context.Context, msg *domain.Message) (*domain.Receipt, error) {
	if err := msg.Validate(); err != nil {
		return nil, err
	}
	body := s.buildSMTPEmail(msg)
	result, resp, err := s.client.TransactionalEmailsApi.SendTransacEmail(ctx, body)
	if err != nil {
		statusCode := 0
		if resp != nil {
			statusCode = resp.StatusCode
		}
		var sdkErr brevoSDK.GenericSwaggerError
		if errors.As(err, &sdkErr) {
			return nil, fmt.Errorf("%w: status=%d, body=%s", domain.ErrSendFailed, statusCode, string(sdkErr.Body()))
		}
		return nil, fmt.Errorf("%w: status=%d, err=%v", domain.ErrSendFailed, statusCode, err)
	}

	status := domain.StatusQueued
	if resp != nil {
		switch resp.StatusCode {
		case 201:
			status = domain.StatusSent
		case 202:
			status = domain.StatusScheduled
		}
	}
	return &domain.Receipt{
		MessageID: result.MessageId,
		Status:    status,
	}, nil
}

var _ port.Sender = (*Sender)(nil)
