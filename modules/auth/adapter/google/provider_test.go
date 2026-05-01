package google

import (
	"errors"
	"strings"
	"testing"

	"github.com/brizenchi/go-modules/modules/auth/domain"
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
