package github

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/brizenchi/go-modules/modules/auth/domain"
	jwtv5 "github.com/golang-jwt/jwt/v5"
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

func TestProviderIssuesUniqueExpiringStateWithRandomID(t *testing.T) {
	p, err := New(Config{
		ClientID: "client-id", ClientSecret: "client-secret",
		RedirectURL: "https://api.example.com/auth/github/callback", StateSecret: "state-secret",
		StateTTL: 35 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	first, err := p.IssueState()
	if err != nil {
		t.Fatal(err)
	}
	second, err := p.IssueState()
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("two OAuth starts shared a state")
	}
	claims := &jwtv5.RegisteredClaims{}
	if _, _, err := jwtv5.NewParser().ParseUnverified(first, claims); err != nil {
		t.Fatal(err)
	}
	if claims.ID == "" || claims.IssuedAt == nil || claims.ExpiresAt == nil {
		t.Fatalf("incomplete claims: %#v", claims)
	}
	if got := claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time); got != 35*time.Minute {
		t.Fatalf("state TTL=%v, want 35m", got)
	}
	if p.OAuthStateTTL() != 35*time.Minute {
		t.Fatalf("OAuthStateTTL=%v, want 35m", p.OAuthStateTTL())
	}
}

func TestProviderRejectsExpiredState(t *testing.T) {
	p, err := New(Config{
		ClientID: "client-id", ClientSecret: "client-secret",
		RedirectURL: "https://api.example.com/auth/github/callback", StateSecret: "state-secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	state, err := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, jwtv5.RegisteredClaims{
		ID: "expired", IssuedAt: jwtv5.NewNumericDate(now.Add(-time.Hour)),
		ExpiresAt: jwtv5.NewNumericDate(now.Add(-time.Minute)),
	}).SignedString([]byte(p.cfg.StateSecret))
	if err != nil {
		t.Fatal(err)
	}
	if err := p.VerifyState(state); !errors.Is(err, domain.ErrInvalidState) {
		t.Fatalf("VerifyState error=%v, want ErrInvalidState", err)
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

func TestProviderNormalizesCallbackErrors(t *testing.T) {
	p, err := New(Config{
		ClientID: "client-id", ClientSecret: "client-secret",
		RedirectURL: "https://api.example.com/auth/github/callback", StateSecret: "state-secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	state, err := p.IssueState()
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name     string
		oauthErr string
		want     error
	}{
		{name: "user denied", oauthErr: "access_denied", want: domain.ErrOAuthDenied},
		{name: "provider callback failure", oauthErr: "temporarily_unavailable", want: domain.ErrOAuthCallback},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, gotErr := p.Exchange(t.Context(), url.Values{
				"state":             {state},
				"error":             {tt.oauthErr},
				"error_description": {"provider detail must not leak"},
			})
			if !errors.Is(gotErr, tt.want) {
				t.Fatalf("error=%v, want %v", gotErr, tt.want)
			}
			if strings.Contains(gotErr.Error(), "provider detail") || strings.Contains(gotErr.Error(), tt.oauthErr) {
				t.Fatalf("provider callback detail leaked: %v", gotErr)
			}
		})
	}

	_, err = p.Exchange(t.Context(), url.Values{"state": {state}})
	if !errors.Is(err, domain.ErrOAuthCallback) {
		t.Fatalf("missing code error=%v, want ErrOAuthCallback", err)
	}
}
