package email

import (
	"context"
	"errors"
	"testing"

	"github.com/brizenchi/go-modules/email/adapter/log"
	"github.com/brizenchi/go-modules/email/domain"
)

func TestModule_Send(t *testing.T) {
	m := New(log.New(nil), nil)
	r, err := m.Send(context.Background(), &domain.Message{
		To: []domain.Address{{Email: "a@b"}}, Subject: "s", TextBody: "x",
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if r.Status != domain.StatusSent {
		t.Errorf("status = %s, want sent", r.Status)
	}
}

func TestManager_GetUnknownReturnsError(t *testing.T) {
	m := NewManager()
	_, err := m.Get("missing")
	if !errors.Is(err, domain.ErrSenderUnavailable) {
		t.Errorf("expected ErrSenderUnavailable, got %v", err)
	}
}

func TestManager_RegisterAndSend(t *testing.T) {
	m := NewManager()
	m.Register("default", New(log.New(nil), nil))
	r, err := m.Send(context.Background(), "default", &domain.Message{
		To: []domain.Address{{Email: "a@b"}}, Subject: "s", TextBody: "x",
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if r.Status != domain.StatusSent {
		t.Errorf("status = %s, want sent", r.Status)
	}
}
