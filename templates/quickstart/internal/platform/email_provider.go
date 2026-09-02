package platform

import (
	"context"
	"fmt"
	"html"
	"strings"

	"github.com/brizenchi/go-modules/modules/auth/adapter/emailcode"
	"github.com/brizenchi/go-modules/modules/email"
	"github.com/brizenchi/go-modules/modules/email/adapter/brevo"
	logsender "github.com/brizenchi/go-modules/modules/email/adapter/log"
	"github.com/brizenchi/go-modules/modules/email/adapter/resend"
	emaildomain "github.com/brizenchi/go-modules/modules/email/domain"
)

func buildEmail(cfg EmailConfig) (*email.Module, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case "none", "disabled", "off":
		return nil, nil
	case "", "log":
		return email.New(logsender.New(nil), nil), nil
	case "brevo":
		sender, err := brevo.New(brevo.Config{
			APIKey: cfg.Brevo.APIKey,
			Sender: emaildomain.Address{Email: cfg.Brevo.SenderEmail, Name: cfg.Brevo.SenderName},
		})
		if err != nil {
			return nil, fmt.Errorf("platform: init brevo: %w", err)
		}
		return email.New(sender, nil), nil
	case "resend":
		sender, err := resend.New(resend.Config{
			APIKey: cfg.Resend.APIKey,
			Sender: emaildomain.Address{Email: cfg.Resend.SenderEmail, Name: cfg.Resend.SenderName},
		})
		if err != nil {
			return nil, fmt.Errorf("platform: init resend: %w", err)
		}
		return email.New(sender, nil), nil
	default:
		return nil, fmt.Errorf("platform: unsupported email provider %q", cfg.Provider)
	}
}

type verificationMailer struct {
	email       *email.Module
	serviceName string
}

func (m verificationMailer) SendProviderTemplate(ctx context.Context, _ string, to []emailcode.EmailAddress, vars map[string]any) error {
	if m.email == nil {
		return fmt.Errorf("platform: email module required for email login")
	}
	addresses := make([]emaildomain.Address, len(to))
	for i, address := range to {
		addresses[i] = emaildomain.Address{Name: address.Name, Email: address.Email}
	}
	code := strings.TrimSpace(fmt.Sprint(vars["code"]))
	brand := strings.TrimSpace(m.serviceName)
	if brand == "" {
		brand = "SaaS"
	}
	_, err := m.email.Send(ctx, &emaildomain.Message{
		To:       addresses,
		Subject:  brand + " 登录验证码",
		TextBody: fmt.Sprintf("你的验证码是 %s，10 分钟内有效。请勿转发给其他人。", code),
		HTMLBody: fmt.Sprintf("<p>你的验证码是：</p><p style=\"font-size:32px;font-weight:700;letter-spacing:6px\">%s</p><p>10 分钟内有效，请勿转发给其他人。</p>", html.EscapeString(code)),
	})
	return err
}
