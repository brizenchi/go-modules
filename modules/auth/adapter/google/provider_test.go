package google

import (
	"errors"
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

func mustContain(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("expected %q in %q", needle, haystack)
	}
}
