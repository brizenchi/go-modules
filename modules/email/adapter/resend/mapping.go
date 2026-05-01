package resend

import (
	"fmt"

	"github.com/brizenchi/go-modules/modules/email/domain"
	resendSDK "github.com/resend/resend-go/v2"
)

// formatAddress renders an Address as RFC 5322 ("Name <email>") when a
// display name is present, or the raw email otherwise.
func formatAddress(a domain.Address) string {
	if a.Name == "" {
		return a.Email
	}
	return fmt.Sprintf("%s <%s>", a.Name, a.Email)
}

func formatAddresses(addrs []domain.Address) []string {
	if len(addrs) == 0 {
		return nil
	}
	out := make([]string, len(addrs))
	for i, a := range addrs {
		out[i] = formatAddress(a)
	}
	return out
}

// buildSendRequest translates domain.Message into a Resend SDK request.
func (s *Sender) buildSendRequest(msg *domain.Message) *resendSDK.SendEmailRequest {
	from := msg.From
	if from.Email == "" {
		from = s.cfg.Sender
	}

	req := &resendSDK.SendEmailRequest{
		From:    formatAddress(from),
		To:      formatAddresses(msg.To),
		Subject: msg.Subject,
		Html:    msg.HTMLBody,
		Text:    msg.TextBody,
		Cc:      formatAddresses(msg.Cc),
		Bcc:     formatAddresses(msg.Bcc),
	}

	if msg.ReplyTo != nil && msg.ReplyTo.Email != "" {
		req.ReplyTo = formatAddress(*msg.ReplyTo)
	}

	if len(msg.Headers) > 0 {
		headers := make(map[string]string, len(msg.Headers))
		for _, h := range msg.Headers {
			headers[h.Name] = h.Value
		}
		req.Headers = headers
	}

	if len(msg.Attachments) > 0 {
		atts := make([]*resendSDK.Attachment, 0, len(msg.Attachments))
		for _, a := range msg.Attachments {
			atts = append(atts, &resendSDK.Attachment{
				Filename:    a.Name,
				ContentType: a.ContentType,
				Path:        a.URL,
				Content:     a.Content,
			})
		}
		req.Attachments = atts
	}

	if len(msg.Tags) > 0 {
		tags := make([]resendSDK.Tag, len(msg.Tags))
		for i, t := range msg.Tags {
			tags[i] = resendSDK.Tag{Name: t}
		}
		req.Tags = tags
	}

	if msg.ScheduledAt != nil {
		req.ScheduledAt = msg.ScheduledAt.UTC().Format("2006-01-02T15:04:05Z")
	}

	return req
}
