package google

import (
	"errors"
	"strings"
	"testing"

	"github.com/brizenchi/go-modules/auth/domain"
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
	// Pick a char guaranteed different from the last sig char — base64url
	// alphabet means just toggling case isn't enough (charset includes both).
	last := state[len(state)-1]
	swap := byte('A')
	if last == swap {
		swap = 'B'
	}
	tampered := state[:len(state)-1] + string(swap)
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
