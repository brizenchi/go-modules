package github

import (
	"net/http"
	"testing"
	"time"
)

func TestNewUsesInjectedHTTPClientTransport(t *testing.T) {
	transport := &http.Transport{}
	base := &http.Client{Transport: transport, Timeout: time.Minute}
	p, err := New(Config{
		ClientID:     "id",
		ClientSecret: "secret",
		RedirectURL:  "https://app.example.com/callback",
		StateSecret:  "state",
		HTTPTimeout:  3 * time.Second,
		HTTPClient:   base,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if p.client.Transport != transport {
		t.Fatal("injected transport not used")
	}
	if p.client.Timeout != 3*time.Second {
		t.Fatalf("timeout = %v, want HTTPTimeout", p.client.Timeout)
	}
	if base.Timeout != time.Minute {
		t.Fatal("caller's client was mutated")
	}
}
