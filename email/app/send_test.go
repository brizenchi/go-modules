package app

import (
	"context"
	"errors"
	"testing"

	"github.com/brizenchi/go-modules/email/domain"
)

type mockSender struct {
	calls   int
	lastMsg *domain.Message
	err     error
	receipt *domain.Receipt
}

func (m *mockSender) Name() string { return "mock" }

func (m *mockSender) Send(ctx context.Context, msg *domain.Message) (*domain.Receipt, error) {
	m.calls++
	m.lastMsg = msg
	if m.err != nil {
		return nil, m.err
	}
	if m.receipt != nil {
		return m.receipt, nil
	}
	return &domain.Receipt{Status: domain.StatusSent, MessageID: "mock-1"}, nil
}

type mockRenderer struct {
	subject, html, text string
	err                 error
}

func (m *mockRenderer) Render(ctx context.Context, name string, vars map[string]any) (string, string, string, error) {
	if m.err != nil {
		return "", "", "", m.err
	}
	return m.subject, m.html, m.text, nil
}

func TestSendService_SendCallsSender(t *testing.T) {
	s := &mockSender{}
	svc := NewSendService(s, nil)
	r, err := svc.Send(context.Background(), &domain.Message{
		To: []domain.Address{{Email: "a@b"}}, Subject: "s", TextBody: "x",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.calls != 1 || r.MessageID != "mock-1" {
		t.Errorf("unexpected: calls=%d r=%v", s.calls, r)
	}
}

func TestSendService_RejectsInvalidMessage(t *testing.T) {
	s := &mockSender{}
	svc := NewSendService(s, nil)
	_, err := svc.Send(context.Background(), &domain.Message{}) // no To
	if !errors.Is(err, domain.ErrInvalidRecipient) {
		t.Errorf("expected ErrInvalidRecipient, got %v", err)
	}
	if s.calls != 0 {
		t.Errorf("sender should not be called on invalid msg, calls=%d", s.calls)
	}
}

func TestSendService_NilMessage(t *testing.T) {
	svc := NewSendService(&mockSender{}, nil)
	_, err := svc.Send(context.Background(), nil)
	if err == nil {
		t.Error("expected error on nil message")
	}
}

func TestSendService_SendTemplate_RequiresRenderer(t *testing.T) {
	svc := NewSendService(&mockSender{}, nil)
	_, err := svc.SendTemplate(context.Background(), TemplateMessage{
		Template: "x",
		To:       []domain.Address{{Email: "a@b"}},
	})
	if !errors.Is(err, domain.ErrTemplateNotFound) {
		t.Errorf("expected ErrTemplateNotFound, got %v", err)
	}
}

func TestSendService_SendTemplate_RendersAndSends(t *testing.T) {
	r := &mockRenderer{
		subject: "Hello",
		html:    "<p>hi</p>",
		text:    "hi",
	}
	s := &mockSender{}
	svc := NewSendService(s, r)
	_, err := svc.SendTemplate(context.Background(), TemplateMessage{
		Template:  "welcome",
		Variables: map[string]any{"Name": "A"},
		To:        []domain.Address{{Email: "a@b"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.lastMsg.Subject != "Hello" {
		t.Errorf("subject = %q, want Hello", s.lastMsg.Subject)
	}
	if s.lastMsg.HTMLBody != "<p>hi</p>" {
		t.Errorf("html = %q", s.lastMsg.HTMLBody)
	}
}

func TestSendService_SendTemplate_PropagatesRenderError(t *testing.T) {
	r := &mockRenderer{err: errors.New("boom")}
	svc := NewSendService(&mockSender{}, r)
	_, err := svc.SendTemplate(context.Background(), TemplateMessage{
		Template: "x", To: []domain.Address{{Email: "a@b"}},
	})
	if err == nil || err.Error() != "boom" {
		t.Errorf("expected boom error, got %v", err)
	}
}

func TestSendService_SendProviderTemplate(t *testing.T) {
	s := &mockSender{}
	svc := NewSendService(s, nil)
	_, err := svc.SendProviderTemplate(context.Background(), "3",
		[]domain.Address{{Email: "a@b"}}, map[string]any{"code": "12345"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.lastMsg.TemplateRef != "3" {
		t.Errorf("TemplateRef = %q, want 3", s.lastMsg.TemplateRef)
	}
	if got := s.lastMsg.Variables["code"]; got != "12345" {
		t.Errorf("vars[code] = %v", got)
	}
}
