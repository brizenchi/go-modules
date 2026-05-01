// Package log is a Sender that does NOT actually send mail; it logs the
// message and returns a fake Receipt.
//
// Use it in development, in tests where a real network call is undesired,
// or as a circuit-breaker fallback.
package log

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/brizenchi/go-modules/modules/email/domain"
	"github.com/brizenchi/go-modules/modules/email/port"
)

// Sender is a no-op email sender that logs every send call.
type Sender struct {
	logger *slog.Logger
}

// New returns a Sender that writes to the given slog.Logger
// (uses slog.Default() when nil).
func New(logger *slog.Logger) *Sender {
	if logger == nil {
		logger = slog.Default()
	}
	return &Sender{logger: logger}
}

func (s *Sender) Name() string { return "log" }

func (s *Sender) Send(ctx context.Context, msg *domain.Message) (*domain.Receipt, error) {
	if err := msg.Validate(); err != nil {
		return nil, err
	}
	to := make([]string, len(msg.To))
	for i, addr := range msg.To {
		to[i] = addr.Email
	}
	s.logger.Info("email.log: pretend send",
		"to", to,
		"subject", msg.Subject,
		"template_ref", msg.TemplateRef,
		"tags", msg.Tags,
	)
	return &domain.Receipt{
		MessageID: fmt.Sprintf("log-%d", len(to)),
		Status:    domain.StatusSent,
	}, nil
}

var _ port.Sender = (*Sender)(nil)
