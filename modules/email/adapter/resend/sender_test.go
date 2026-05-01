package resend

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/brizenchi/go-modules/modules/email/domain"
)

func newServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *Sender) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	s, err := New(Config{
		APIKey:  "test-key",
		BaseURL: srv.URL,
		Sender:  domain.Address{Name: "Acme", Email: "no-reply@acme.test"},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return srv, s
}

func TestConfigValidate(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
		want error
	}{
		{"missing api key", Config{Sender: domain.Address{Email: "a@b"}}, domain.ErrInvalidAPIKey},
		{"missing sender", Config{APIKey: "k"}, domain.ErrInvalidSender},
		{"ok", Config{APIKey: "k", Sender: domain.Address{Email: "a@b"}}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.cfg.Validate()
			if !errors.Is(got, tc.want) {
				t.Fatalf("want %v, got %v", tc.want, got)
			}
		})
	}
}

func TestSend_Success(t *testing.T) {
	var captured map[string]any
	_, s := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("auth header = %q", got)
		}
		_ = json.NewDecoder(r.Body).Decode(&captured)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"abc-123"}`))
	})

	scheduled := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	rec, err := s.Send(context.Background(), &domain.Message{
		To:          []domain.Address{{Name: "Bob", Email: "bob@x.test"}},
		Subject:     "hi",
		HTMLBody:    "<p>hi</p>",
		Tags:        []string{"welcome"},
		ScheduledAt: &scheduled,
		Headers:     []domain.Header{{Name: "X-Test", Value: "1"}},
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if rec.MessageID != "abc-123" {
		t.Errorf("MessageID = %q", rec.MessageID)
	}
	if rec.Status != domain.StatusScheduled {
		t.Errorf("Status = %q, want scheduled", rec.Status)
	}
	if got, _ := captured["from"].(string); !strings.Contains(got, "no-reply@acme.test") {
		t.Errorf("from default not applied: %v", captured["from"])
	}
	if got, _ := captured["scheduled_at"].(string); got != "2026-01-01T12:00:00Z" {
		t.Errorf("scheduled_at = %q", got)
	}
}

func TestSend_TemplateRefRejected(t *testing.T) {
	_, s := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request")
	})
	_, err := s.Send(context.Background(), &domain.Message{
		To:          []domain.Address{{Email: "a@b.test"}},
		Subject:     "x",
		HTMLBody:    "<p/>",
		TemplateRef: "welcome",
	})
	if !errors.Is(err, domain.ErrTemplateNotFound) {
		t.Fatalf("want ErrTemplateNotFound, got %v", err)
	}
}

func TestSend_ProviderError(t *testing.T) {
	_, s := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"bad request"}`))
	})
	_, err := s.Send(context.Background(), &domain.Message{
		To:       []domain.Address{{Email: "a@b.test"}},
		Subject:  "x",
		HTMLBody: "<p/>",
	})
	if !errors.Is(err, domain.ErrSendFailed) {
		t.Fatalf("want ErrSendFailed, got %v", err)
	}
}

func TestFormatAddress(t *testing.T) {
	if got := formatAddress(domain.Address{Email: "a@b"}); got != "a@b" {
		t.Errorf("plain: %q", got)
	}
	if got := formatAddress(domain.Address{Name: "Bob", Email: "b@x"}); got != "Bob <b@x>" {
		t.Errorf("named: %q", got)
	}
}
