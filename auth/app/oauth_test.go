package app

import (
	"context"
	"errors"
	"net/url"
	"testing"

	"github.com/brizenchi/go-modules/auth/domain"
	"github.com/brizenchi/go-modules/auth/event"
	"github.com/brizenchi/go-modules/auth/port"
)

func newOAuth(p *mockProvider, users *mockUserStore, bus *mockBus, store *mockExchange) *OAuthService {
	providers := map[string]port.IdentityProvider{string(p.name): p}
	return NewOAuthService(OAuthDeps{
		Providers:     providers,
		Users:         users,
		Signer:        mockSigner{},
		ExchangeStore: store,
		Bus:           bus,
	})
}

func TestOAuth_StartReturnsAuthorizeURL(t *testing.T) {
	svc := newOAuth(newMockProvider(), newMockUserStore(), &mockBus{}, newMockExchange())
	got, err := svc.StartOAuth(context.Background(), "google")
	if err != nil {
		t.Fatal(err)
	}
	if got == "" || !contains(got, "state-1") {
		t.Errorf("authorize url = %q", got)
	}
}

func TestOAuth_StartUnknownProvider(t *testing.T) {
	svc := newOAuth(newMockProvider(), newMockUserStore(), &mockBus{}, newMockExchange())
	_, err := svc.StartOAuth(context.Background(), "github")
	if !errors.Is(err, domain.ErrProviderUnavailable) {
		t.Errorf("expected ErrProviderUnavailable, got %v", err)
	}
}

func TestOAuth_CallbackStagesExchangeCode(t *testing.T) {
	store := newMockExchange()
	svc := newOAuth(newMockProvider(), newMockUserStore(), &mockBus{}, store)
	res, err := svc.OAuthCallback(context.Background(), "google", url.Values{"code": []string{"x"}, "state": []string{"state-1"}})
	if err != nil {
		t.Fatalf("callback: %v", err)
	}
	if res.ExchangeCode == "" {
		t.Error("expected non-empty exchange code")
	}
	if !res.Identity.IsNew {
		t.Error("expected new user from mock provider's email")
	}
}

func TestOAuth_CallbackPropagatesProviderError(t *testing.T) {
	p := newMockProvider()
	p.exchangeErr = errors.New("boom")
	svc := newOAuth(p, newMockUserStore(), &mockBus{}, newMockExchange())
	_, err := svc.OAuthCallback(context.Background(), "google", nil)
	if err == nil || err.Error() != "boom" {
		t.Errorf("expected boom error, got %v", err)
	}
}

func TestOAuth_ExchangeTokenFlow(t *testing.T) {
	users := newMockUserStore()
	bus := &mockBus{}
	store := newMockExchange()
	svc := newOAuth(newMockProvider(), users, bus, store)
	cb, err := svc.OAuthCallback(context.Background(), "google", url.Values{"code": []string{"x"}, "state": []string{"state-1"}})
	if err != nil {
		t.Fatal(err)
	}
	res, err := svc.ExchangeToken(context.Background(), cb.ExchangeCode)
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	if res.Token.Value == "" {
		t.Error("expected non-empty token")
	}
	if got := len(bus.Got(event.KindUserSignedUp)); got != 1 {
		t.Errorf("UserSignedUp count = %d, want 1", got)
	}
	if got := len(bus.Got(event.KindUserLoggedIn)); got != 1 {
		t.Errorf("UserLoggedIn count = %d, want 1", got)
	}
}

func TestOAuth_ExchangeTokenSingleUse(t *testing.T) {
	store := newMockExchange()
	svc := newOAuth(newMockProvider(), newMockUserStore(), &mockBus{}, store)
	cb, _ := svc.OAuthCallback(context.Background(), "google", url.Values{"code": []string{"x"}, "state": []string{"state-1"}})
	if _, err := svc.ExchangeToken(context.Background(), cb.ExchangeCode); err != nil {
		t.Fatal(err)
	}
	// Second call must fail.
	if _, err := svc.ExchangeToken(context.Background(), cb.ExchangeCode); !errors.Is(err, domain.ErrInvalidExchange) {
		t.Errorf("expected ErrInvalidExchange on reuse, got %v", err)
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
