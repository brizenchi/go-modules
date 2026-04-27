// Package smtp is a Sender adapter that talks to a plain SMTP server
// (Postfix, mailhog, AWS SES SMTP, ...).
//
// It uses the stdlib net/smtp client. For more advanced features
// (DKIM signing, OAuth2, connection pooling) use a provider adapter
// instead.
package smtp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"net/smtp"
	"strings"

	"github.com/brizenchi/go-modules/email/domain"
	"github.com/brizenchi/go-modules/email/port"
)

// Config holds the SMTP server settings.
type Config struct {
	Host     string // e.g. "smtp.sendgrid.net"
	Port     int    // e.g. 587
	Username string // optional (no AUTH if empty)
	Password string
	Sender   domain.Address // default From
}

func (c Config) Validate() error {
	if c.Host == "" || c.Port == 0 {
		return fmt.Errorf("%w: host and port required", domain.ErrInvalidAPIKey)
	}
	if c.Sender.Email == "" {
		return domain.ErrInvalidSender
	}
	return nil
}

// Sender implements port.Sender via SMTP.
type Sender struct {
	cfg Config
}

func New(cfg Config) (*Sender, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &Sender{cfg: cfg}, nil
}

func (s *Sender) Name() string { return "smtp" }

// Send delivers a message via SMTP. Templates (TemplateRef) are not
// supported by this adapter — callers should pre-render the template
// and set Subject + HTMLBody/TextBody directly.
func (s *Sender) Send(ctx context.Context, msg *domain.Message) (*domain.Receipt, error) {
	if err := msg.Validate(); err != nil {
		return nil, err
	}
	if msg.TemplateRef != "" {
		return nil, fmt.Errorf("%w: smtp adapter does not support TemplateRef; render locally first", domain.ErrSendFailed)
	}

	from := msg.From
	if from.Email == "" {
		from = s.cfg.Sender
	}

	to := make([]string, 0, len(msg.To)+len(msg.Cc)+len(msg.Bcc))
	for _, a := range msg.To {
		to = append(to, a.Email)
	}
	for _, a := range msg.Cc {
		to = append(to, a.Email)
	}
	for _, a := range msg.Bcc {
		to = append(to, a.Email)
	}

	body, err := buildMIME(from, msg)
	if err != nil {
		return nil, err
	}

	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	var auth smtp.Auth
	if s.cfg.Username != "" {
		auth = smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	}
	if err := smtp.SendMail(addr, auth, from.Email, to, body); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrSendFailed, err)
	}
	return &domain.Receipt{Status: domain.StatusSent}, nil
}

// buildMIME builds a minimal multipart/alternative MIME message.
// Attachments are not yet supported by this adapter.
func buildMIME(from domain.Address, msg *domain.Message) ([]byte, error) {
	if len(msg.Attachments) > 0 {
		return nil, errors.New("smtp: attachments not yet supported, use a provider adapter")
	}
	var buf bytes.Buffer
	writeHeader(&buf, "From", formatAddr(from))
	writeHeader(&buf, "To", joinAddrs(msg.To))
	if len(msg.Cc) > 0 {
		writeHeader(&buf, "Cc", joinAddrs(msg.Cc))
	}
	if msg.ReplyTo != nil && msg.ReplyTo.Email != "" {
		writeHeader(&buf, "Reply-To", formatAddr(*msg.ReplyTo))
	}
	writeHeader(&buf, "Subject", msg.Subject)
	writeHeader(&buf, "MIME-Version", "1.0")
	for _, h := range msg.Headers {
		writeHeader(&buf, h.Name, h.Value)
	}

	if msg.HTMLBody != "" && msg.TextBody != "" {
		mw := multipart.NewWriter(&buf)
		writeHeader(&buf, "Content-Type", `multipart/alternative; boundary="`+mw.Boundary()+`"`)
		buf.WriteString("\r\n")
		writePart(mw, "text/plain; charset=UTF-8", msg.TextBody)
		writePart(mw, "text/html; charset=UTF-8", msg.HTMLBody)
		mw.Close()
	} else if msg.HTMLBody != "" {
		writeHeader(&buf, "Content-Type", "text/html; charset=UTF-8")
		buf.WriteString("\r\n")
		buf.WriteString(msg.HTMLBody)
	} else {
		writeHeader(&buf, "Content-Type", "text/plain; charset=UTF-8")
		buf.WriteString("\r\n")
		buf.WriteString(msg.TextBody)
	}
	return buf.Bytes(), nil
}

func writeHeader(buf *bytes.Buffer, name, value string) {
	buf.WriteString(name)
	buf.WriteString(": ")
	buf.WriteString(value)
	buf.WriteString("\r\n")
}

func writePart(mw *multipart.Writer, contentType, body string) {
	w, _ := mw.CreatePart(map[string][]string{"Content-Type": {contentType}})
	_, _ = w.Write([]byte(body))
}

func formatAddr(a domain.Address) string {
	if a.Name == "" {
		return a.Email
	}
	return fmt.Sprintf("%q <%s>", a.Name, a.Email)
}

func joinAddrs(addrs []domain.Address) string {
	out := make([]string, len(addrs))
	for i, a := range addrs {
		out[i] = formatAddr(a)
	}
	return strings.Join(out, ", ")
}

var _ port.Sender = (*Sender)(nil)
