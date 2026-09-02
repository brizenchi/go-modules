package github

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/brizenchi/go-modules/modules/auth/domain"
)

func TestProviderAuthorizeURLAndState(t *testing.T) {
	p, err := New(Config{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURL:  "https://api.example.com/auth/github/callback",
		StateSecret:  "state-secret",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	state, err := p.IssueState()
	if err != nil {
		t.Fatalf("IssueState: %v", err)
	}
	if err := p.VerifyState(state); err != nil {
		t.Fatalf("VerifyState: %v", err)
	}
	authorizeURL, err := p.AuthorizeURL(state, url.Values{"allow_signup": {"false"}})
	if err != nil {
		t.Fatalf("AuthorizeURL: %v", err)
	}
	parsed, err := url.Parse(authorizeURL)
	if err != nil {
		t.Fatalf("parse authorize url: %v", err)
	}
	if parsed.Host != "github.com" || parsed.Query().Get("client_id") != "client-id" {
		t.Fatalf("unexpected authorize url: %s", authorizeURL)
	}
	if parsed.Query().Get("state") != state || parsed.Query().Get("allow_signup") != "false" {
		t.Fatalf("missing state/extra query: %s", authorizeURL)
	}
}

func TestProviderExchangeUsesVerifiedPrimaryEmail(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse token form: %v", err)
		}
		if r.Form.Get("code") != "oauth-code" || r.Form.Get("client_id") != "client-id" {
			t.Fatalf("unexpected token form: %v", r.Form)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "access-token"})
	})
	mux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer access-token" {
			t.Fatalf("authorization = %q", r.Header.Get("Authorization"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":         12345,
			"login":      "octocat",
			"name":       "The Octocat",
			"avatar_url": "https://example.com/avatar.png",
			"email":      "",
		})
	})
	mux.HandleFunc("/emails", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"email": "other@example.com", "primary": false, "verified": true},
			{"email": " OCTOCAT@EXAMPLE.COM ", "primary": true, "verified": true},
		})
	})

	p, err := New(Config{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURL:  "https://api.example.com/auth/github/callback",
		StateSecret:  "state-secret",
		TokenURL:     server.URL + "/token",
		UserURL:      server.URL + "/user",
		EmailsURL:    server.URL + "/emails",
		HTTPTimeout:  time.Second,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	state, _ := p.IssueState()
	profile, err := p.Exchange(t.Context(), url.Values{"code": {"oauth-code"}, "state": {state}})
	if err != nil {
		t.Fatalf("Exchange: %v", err)
	}
	if profile.Provider != domain.ProviderGitHub || profile.Subject != "12345" {
		t.Fatalf("unexpected identity: %#v", profile)
	}
	if profile.Email != "octocat@example.com" || profile.Username != "The Octocat" {
		t.Fatalf("unexpected profile: %#v", profile)
	}
}

func TestProviderRejectsInvalidState(t *testing.T) {
	p, err := New(Config{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURL:  "https://api.example.com/auth/github/callback",
		StateSecret:  "state-secret",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = p.Exchange(t.Context(), url.Values{"code": {"oauth-code"}, "state": {"invalid"}})
	if err == nil || !strings.Contains(err.Error(), domain.ErrInvalidState.Error()) {
		t.Fatalf("Exchange error=%v, want invalid state", err)
	}
}
