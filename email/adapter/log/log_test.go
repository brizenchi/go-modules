package log

import (
	"context"
	"errors"
	"testing"

	"github.com/brizenchi/go-modules/email/domain"
)

func TestLogSender_SendOK(t *testing.T) {
	s := New(nil)
	r, err := s.Send(context.Background(), &domain.Message{
		To:       []domain.Address{{Email: "a@b"}},
		Subject:  "s",
		TextBody: "hi",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Status != domain.StatusSent {
		t.Errorf("status = %s, want sent", r.Status)
	}
	if r.MessageID == "" {
		t.Error("expected non-empty MessageID")
	}
	if s.Name() != "log" {
		t.Errorf("name = %q, want log", s.Name())
	}
}

func TestLogSender_RejectsInvalidMessage(t *testing.T) {
	s := New(nil)
	_, err := s.Send(context.Background(), &domain.Message{}) // no recipient
	if !errors.Is(err, domain.ErrInvalidRecipient) {
		t.Errorf("expected ErrInvalidRecipient, got %v", err)
	}
}
