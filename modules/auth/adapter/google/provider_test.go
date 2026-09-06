package google

import (
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

func newTestProvider(t *testing.T) *Provider {
	t.Helper()
	p, err := New(Config{
		ClientID:     "cid",
		ClientSecret: "csec",
		RedirectURL:  "https://app/cb",
		StateSecret:  "state-secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestProvider_Name(t *testing.T) {
	p := newTestProvider(t)
	if p.Name() != domain.ProviderGoogle {
		t.Errorf("name = %s", p.Name())
	}
}

func TestProvider_StateRoundTrip(t *testing.T) {
	p := newTestProvider(t)
	state, err := p.IssueState()
	if err != nil {
		t.Fatal(err)
	}
	if err := p.VerifyState(state); err != nil {
		t.Errorf("VerifyState: %v", err)
	}
}

func TestProvider_IssueStateAlwaysUsesFreshRandomID(t *testing.T) {
	p := newTestProvider(t)
	first, err := p.IssueState()
	if err != nil {
		t.Fatal(err)
	}
	second, err := p.IssueState()
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("two OAuth flows issued in the same instant shared a state")
	}
	claims := &jwtv5.RegisteredClaims{}
	if _, _, err := jwtv5.NewParser().ParseUnverified(first, claims); err != nil {
		t.Fatal(err)
	}
	if claims.ID == "" {
		t.Fatal("state JWT omitted random jti")
	}
}

func TestProvider_VerifyStateRejectsEmpty(t *testing.T) {
	p := newTestProvider(t)
	if err := p.VerifyState(""); !errors.Is(err, domain.ErrInvalidState) {
		t.Errorf("expected ErrInvalidState, got %v", err)
	}
}

func TestProvider_VerifyStateRejectsTampered(t *testing.T) {
	p := newTestProvider(t)
	state, _ := p.IssueState()
	// Mutate a char in the payload segment of the JWT (between the two
	// dots). Mutating the last sig char is unreliable: base64url's
	// trailing char encodes only 4 bits, so swapping to a value sharing
	// the high 2 bits leaves the decoded signature unchanged → HMAC verify
	// still passes.
	parts := strings.SplitN(state, ".", 3)
	if len(parts) != 3 {
		t.Fatalf("malformed jwt state: %q", state)
	}
	mid := []byte(parts[1])
	idx := len(mid) / 2
	if mid[idx] == 'A' {
		mid[idx] = 'B'
	} else {
		mid[idx] = 'A'
	}
	tampered := parts[0] + "." + string(mid) + "." + parts[2]
	if err := p.VerifyState(tampered); !errors.Is(err, domain.ErrInvalidState) {
		t.Errorf("expected ErrInvalidState, got %v", err)
	}
}

func TestProvider_VerifyStateRejectsExpired(t *testing.T) {
	p := newTestProvider(t)
	now := time.Now().UTC()
	claims := jwtv5.RegisteredClaims{
		IssuedAt:  jwtv5.NewNumericDate(now.Add(-2 * time.Minute)),
		ExpiresAt: jwtv5.NewNumericDate(now.Add(-time.Minute)),
		ID:        "expired-state-id",
	}
	state, err := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims).SignedString([]byte(p.cfg.StateSecret))
	if err != nil {
		t.Fatal(err)
	}
	if err := p.VerifyState(state); !errors.Is(err, domain.ErrInvalidState) {
		t.Fatalf("expected ErrInvalidState for expired state, got %v", err)
	}
}

func TestProvider_DefaultStateTTLIsTwentyMinutes(t *testing.T) {
	p := newTestProvider(t)
	if p.cfg.StateTTL != 20*time.Minute {
		t.Fatalf("StateTTL = %v, want 20m", p.cfg.StateTTL)
	}
}

func TestProvider_ExposesOAuthStateTTL(t *testing.T) {
	p := newTestProvider(t)
	if got := p.OAuthStateTTL(); got != 20*time.Minute {
		t.Fatalf("OAuthStateTTL=%v, want 20m", got)
	}
}

func TestProvider_CustomStateTTLIsApplied(t *testing.T) {
	p, err := New(Config{
		ClientID:     "cid",
		ClientSecret: "csec",
		RedirectURL:  "https://app/cb",
		StateSecret:  "state-secret",
		StateTTL:     45 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	state, err := p.IssueState()
	if err != nil {
		t.Fatal(err)
	}
	claims := &jwtv5.RegisteredClaims{}
	if _, _, err := jwtv5.NewParser().ParseUnverified(state, claims); err != nil {
		t.Fatalf("ParseUnverified: %v", err)
	}
	if claims.IssuedAt == nil || claims.ExpiresAt == nil {
		t.Fatalf("expected iat and exp claims, got %+v", claims)
	}
	if got := claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time); got != 45*time.Minute {
		t.Fatalf("state ttl = %v, want 45m", got)
	}
}

func TestProvider_AuthorizeURL(t *testing.T) {
	p := newTestProvider(t)
	state, _ := p.IssueState()
	url, err := p.AuthorizeURL(state, nil)
	if err != nil {
		t.Fatal(err)
	}
	mustContain(t, url, "client_id=cid")
	mustContain(t, url, "redirect_uri=")
	mustContain(t, url, "state="+state[:20]) // partial match — query encoded
}

func TestProvider_ExchangeNormalizesCallbackErrors(t *testing.T) {
	p := newTestProvider(t)
	state, err := p.IssueState()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name     string
		oauthErr string
		want     error
	}{
		{name: "user denied", oauthErr: "access_denied", want: domain.ErrOAuthDenied},
		{name: "provider callback failure", oauthErr: "server_error", want: domain.ErrOAuthCallback},
	}
	for _, tt := range tests {
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

func TestProvider_FetchUserInfoRequiresVerifiedEmail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sub":"google-1","email":"owner@example.com","email_verified":false}`))
	}))
	defer server.Close()

	p := newTestProvider(t)
	p.cfg.UserInfoURL = server.URL
	if _, err := p.fetchUserInfo(t.Context(), "token"); err == nil || !strings.Contains(err.Error(), "verified") {
		t.Fatalf("expected unverified email to be rejected, got %v", err)
	}
}

func TestProvider_FetchUserInfoAcceptsVerifiedEmail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sub":"google-1","email":"owner@example.com","email_verified":true}`))
	}))
	defer server.Close()

	p := newTestProvider(t)
	p.cfg.UserInfoURL = server.URL
	info, err := p.fetchUserInfo(t.Context(), "token")
	if err != nil {
		t.Fatalf("fetchUserInfo: %v", err)
	}
	if info.Email != "owner@example.com" {
		t.Fatalf("email=%q", info.Email)
	}
}

func mustContain(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("expected %q in %q", needle, haystack)
	}
}
