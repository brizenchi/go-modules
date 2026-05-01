package brevo

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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

func TestSend_Success_201Sent(t *testing.T) {
	var captured map[string]any
	_, s := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("api-key"); got != "test-key" {
			t.Errorf("api-key header = %q", got)
		}
		_ = json.NewDecoder(r.Body).Decode(&captured)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"messageId":"<msg-1@brevo>"}`))
	})

	rec, err := s.Send(context.Background(), &domain.Message{
		To:       []domain.Address{{Name: "Bob", Email: "bob@x.test"}},
		Subject:  "hi",
		HTMLBody: "<p>hi</p>",
		Tags:     []string{"welcome"},
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if rec.MessageID != "<msg-1@brevo>" {
		t.Errorf("MessageID = %q", rec.MessageID)
	}
	if rec.Status != domain.StatusSent {
		t.Errorf("Status = %q, want sent", rec.Status)
	}
	sender, _ := captured["sender"].(map[string]any)
	if email, _ := sender["email"].(string); email != "no-reply@acme.test" {
		t.Errorf("default sender not applied: %v", sender)
	}
}

func TestSend_Success_202Scheduled(t *testing.T) {
	_, s := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"messageId":"<msg-2@brevo>"}`))
	})
	scheduled := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	rec, err := s.Send(context.Background(), &domain.Message{
		To:          []domain.Address{{Email: "a@b.test"}},
		Subject:     "hi",
		HTMLBody:    "<p/>",
		ScheduledAt: &scheduled,
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if rec.Status != domain.StatusScheduled {
		t.Errorf("Status = %q, want scheduled", rec.Status)
	}
}

func TestSend_TemplateRefNumeric(t *testing.T) {
	var captured map[string]any
	_, s := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&captured)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"messageId":"<id>"}`))
	})
	_, err := s.Send(context.Background(), &domain.Message{
		To:          []domain.Address{{Email: "a@b.test"}},
		Subject:     "x",
		HTMLBody:    "<p/>",
		TemplateRef: "42",
		Variables:   map[string]any{"name": "Bob"},
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if id, _ := captured["templateId"].(float64); id != 42 {
		t.Errorf("templateId = %v", captured["templateId"])
	}
}

func TestSend_ProviderError(t *testing.T) {
	_, s := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":"bad_request","message":"nope"}`))
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
