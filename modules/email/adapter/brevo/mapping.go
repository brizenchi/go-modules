package brevo

import (
	"encoding/base64"
	"strconv"
	"strings"

	"github.com/brizenchi/go-modules/modules/email/domain"
	brevoSDK "github.com/getbrevo/brevo-go/lib"
)

// buildSMTPEmail translates domain.Message into Brevo SDK request body.
func (s *Sender) buildSMTPEmail(msg *domain.Message) brevoSDK.SendSmtpEmail {
	body := brevoSDK.SendSmtpEmail{Subject: msg.Subject}

	from := msg.From
	if from.Email == "" {
		from = s.cfg.Sender
	}
	if from.Email != "" {
		body.Sender = &brevoSDK.SendSmtpEmailSender{Name: from.Name, Email: from.Email}
	}

	if len(msg.To) > 0 {
		to := make([]brevoSDK.SendSmtpEmailTo, len(msg.To))
		for i, a := range msg.To {
			to[i] = brevoSDK.SendSmtpEmailTo{Name: a.Name, Email: a.Email}
		}
		body.To = to
	}
	if len(msg.Cc) > 0 {
		cc := make([]brevoSDK.SendSmtpEmailCc, len(msg.Cc))
		for i, a := range msg.Cc {
			cc[i] = brevoSDK.SendSmtpEmailCc{Name: a.Name, Email: a.Email}
		}
		body.Cc = cc
	}
	if len(msg.Bcc) > 0 {
		bcc := make([]brevoSDK.SendSmtpEmailBcc, len(msg.Bcc))
		for i, a := range msg.Bcc {
			bcc[i] = brevoSDK.SendSmtpEmailBcc{Name: a.Name, Email: a.Email}
		}
		body.Bcc = bcc
	}
	if msg.ReplyTo != nil && msg.ReplyTo.Email != "" {
		body.ReplyTo = &brevoSDK.SendSmtpEmailReplyTo{Name: msg.ReplyTo.Name, Email: msg.ReplyTo.Email}
	}

	if msg.HTMLBody != "" {
		body.HtmlContent = msg.HTMLBody
	}
	if msg.TextBody != "" {
		body.TextContent = msg.TextBody
	}

	if len(msg.Attachments) > 0 {
		atts := make([]brevoSDK.SendSmtpEmailAttachment, 0, len(msg.Attachments))
		for _, a := range msg.Attachments {
			att := brevoSDK.SendSmtpEmailAttachment{Name: a.Name, Url: a.URL}
			if len(a.Content) > 0 {
				att.Content = base64.StdEncoding.EncodeToString(a.Content)
			}
			atts = append(atts, att)
		}
		body.Attachment = atts
	}

	if len(msg.Headers) > 0 {
		headers := make(map[string]any, len(msg.Headers))
		for _, h := range msg.Headers {
			headers[h.Name] = h.Value
		}
		body.Headers = headers
	}

	if msg.TemplateRef != "" {
		if id, err := strconv.ParseInt(strings.TrimSpace(msg.TemplateRef), 10, 64); err == nil {
			body.TemplateId = id
			if len(msg.Variables) > 0 {
				body.Params = msg.Variables
			}
		}
	}

	if len(msg.Tags) > 0 {
		body.Tags = msg.Tags
	}

	if msg.ScheduledAt != nil {
		t := *msg.ScheduledAt
		body.ScheduledAt = &t
	}
	return body
}
