package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/brizenchi/go-modules/modules/auth/domain"
	"github.com/brizenchi/go-modules/modules/auth/event"
	"github.com/brizenchi/go-modules/modules/auth/port"
)

func newOAuth(p *mockProvider, users *mockUserStore, bus *mockBus, store *mockExchange) *OAuthService {
	providers := map[string]port.IdentityProvider{string(p.name): p}
	return NewOAuthService(OAuthDeps{
		Providers:     providers,
		Users:         users,
		Signer:        mockSigner{},
		ExchangeStore: store,
		FlowStore:     newMockOAuthFlowStore(),
		Bus:           bus,
	})
}

func testOAuthVerifier(seed byte) (string, string) {
	verifier := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{seed}, 32))
	challenge, err := oauthVerifierChallenge(verifier)
	if err != nil {
		panic(err)
	}
	return verifier, challenge
}

func beginOAuth(t *testing.T, svc *OAuthService, provider string) (*OAuthStartResult, string, string) {
	t.Helper()
	verifier, challenge := testOAuthVerifier(1)
	result, err := svc.StartOAuth(t.Context(), provider, challenge)
	if err != nil {
		t.Fatalf("StartOAuth: %v", err)
	}
	parsed, err := url.Parse(result.AuthorizeURL)
	if err != nil {
		t.Fatalf("parse authorize URL: %v", err)
	}
	state := parsed.Query().Get("state")
	if state == "" {
		t.Fatal("authorize URL omitted state")
	}
	return result, state, verifier
}

func finishOAuth(t *testing.T, svc *OAuthService, provider string) (*OAuthCallbackResult, string) {
	t.Helper()
	start, state, verifier := beginOAuth(t, svc, provider)
	result, err := svc.OAuthCallback(
		t.Context(), provider, url.Values{"code": {"x"}, "state": {state}}, start.BrowserBinding,
	)
	if err != nil {
		t.Fatalf("OAuthCallback: %v", err)
	}
	return result, verifier
}

func TestOAuth_StartReturnsAuthorizeURLAndBrowserBinding(t *testing.T) {
	p := newMockProvider()
	svc := newOAuth(p, newMockUserStore(), &mockBus{}, newMockExchange())
	result, state, _ := beginOAuth(t, svc, "google")
	if result.AuthorizeURL == "" || state != "state-1" || result.BrowserBinding == "" {
		t.Fatalf("unexpected start result: %#v state=%q", result, state)
	}
	if got := time.Until(result.ExpiresAt); got < 4*time.Minute || got > 6*time.Minute {
		t.Fatalf("flow ttl=%v, want provider ttl near 5m", got)
	}
}

func TestOAuth_StartUnknownProvider(t *testing.T) {
	svc := newOAuth(newMockProvider(), newMockUserStore(), &mockBus{}, newMockExchange())
	_, challenge := testOAuthVerifier(1)
	_, err := svc.StartOAuth(context.Background(), "github", challenge)
	if !errors.Is(err, domain.ErrProviderUnavailable) {
		t.Errorf("expected ErrProviderUnavailable, got %v", err)
	}
}

func TestOAuth_StartRejectsMissingOrMalformedVerifierChallenge(t *testing.T) {
	svc := newOAuth(newMockProvider(), newMockUserStore(), &mockBus{}, newMockExchange())
	for _, challenge := range []string{"", "not-a-sha256-challenge", "abc="} {
		if _, err := svc.StartOAuth(t.Context(), "google", challenge); !errors.Is(err, domain.ErrInvalidState) {
			t.Fatalf("challenge=%q error=%v, want ErrInvalidState", challenge, err)
		}
	}
}

func TestOAuth_CallbackStagesBrowserBoundExchangeCode(t *testing.T) {
	store := newMockExchange()
	svc := newOAuth(newMockProvider(), newMockUserStore(), &mockBus{}, store)
	result, verifier := finishOAuth(t, svc, "google")
	if result.ExchangeCode == "" || !result.Identity.IsNew {
		t.Fatalf("unexpected callback result: %#v", result)
	}
	challenge, _ := oauthVerifierChallenge(verifier)
	stored := store.m[result.ExchangeCode]
	if stored.BindingHash != challenge {
		t.Fatalf("exchange binding=%q, want verifier challenge %q", stored.BindingHash, challenge)
	}
}

func TestOAuth_CallbackConsumesStateBeforeProviderError(t *testing.T) {
	p := newMockProvider()
	p.exchangeErr = errors.New("boom")
	svc := newOAuth(p, newMockUserStore(), &mockBus{}, newMockExchange())
	start, state, _ := beginOAuth(t, svc, "google")
	query := url.Values{"code": {"x"}, "state": {state}}
	if _, err := svc.OAuthCallback(t.Context(), "google", query, start.BrowserBinding); err == nil || err.Error() != "boom" {
		t.Fatalf("expected boom error, got %v", err)
	}
	if _, err := svc.OAuthCallback(t.Context(), "google", query, start.BrowserBinding); !errors.Is(err, domain.ErrInvalidState) {
		t.Fatalf("replayed callback error=%v, want ErrInvalidState", err)
	}
}

func TestOAuth_CallbackRejectsWrongBrowserWithoutConsumingLegitimateFlow(t *testing.T) {
	p := newMockProvider()
	svc := newOAuth(p, newMockUserStore(), &mockBus{}, newMockExchange())
	start, state, _ := beginOAuth(t, svc, "google")
	query := url.Values{"code": {"x"}, "state": {state}}
	if _, err := svc.OAuthCallback(t.Context(), "google", query, "another-browser"); !errors.Is(err, domain.ErrInvalidState) {
		t.Fatalf("wrong-browser error=%v, want ErrInvalidState", err)
	}
	if p.exchangeCount() != 0 {
		t.Fatal("provider exchange ran for a mismatched browser")
	}
	if _, err := svc.OAuthCallback(t.Context(), "google", query, start.BrowserBinding); err != nil {
		t.Fatalf("legitimate browser callback: %v", err)
	}
}

func TestOAuth_CallbackStateIsSingleUseUnderConcurrency(t *testing.T) {
	p := newMockProvider()
	svc := newOAuth(p, newMockUserStore(), &mockBus{}, newMockExchange())
	start, state, _ := beginOAuth(t, svc, "google")
	query := url.Values{"code": {"x"}, "state": {state}}
	var successes atomic.Int32
	var invalid atomic.Int32
	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.OAuthCallback(context.Background(), "google", query, start.BrowserBinding)
			switch {
			case err == nil:
				successes.Add(1)
			case errors.Is(err, domain.ErrInvalidState):
				invalid.Add(1)
			default:
				t.Errorf("unexpected callback error: %v", err)
			}
		}()
	}
	wg.Wait()
	if successes.Load() != 1 || invalid.Load() != 15 || p.exchangeCount() != 1 {
		t.Fatalf("success=%d invalid=%d provider exchanges=%d", successes.Load(), invalid.Load(), p.exchangeCount())
	}
}

func TestOAuth_ParallelFlowsHaveDistinctStateAndRemainUsable(t *testing.T) {
	svc := newOAuth(newMockProvider(), newMockUserStore(), &mockBus{}, newMockExchange())
	first, firstState, _ := beginOAuth(t, svc, "google")
	second, secondState, _ := beginOAuth(t, svc, "google")
	if firstState == secondState || first.FlowID == second.FlowID {
		t.Fatal("parallel OAuth starts reused state")
	}
	for _, flow := range []struct {
		start *OAuthStartResult
		state string
	}{{first, firstState}, {second, secondState}} {
		if _, err := svc.OAuthCallback(t.Context(), "google", url.Values{"code": {"x"}, "state": {flow.state}}, flow.start.BrowserBinding); err != nil {
			t.Fatalf("parallel callback: %v", err)
		}
	}
}

func TestOAuth_ExpiredServerFlowIsRejected(t *testing.T) {
	p := newMockProvider()
	flowStore := newMockOAuthFlowStore()
	svc := NewOAuthService(OAuthDeps{
		Providers: map[string]port.IdentityProvider{"google": p}, Users: newMockUserStore(),
		Signer: mockSigner{}, ExchangeStore: newMockExchange(), FlowStore: flowStore,
	})
	start, state, _ := beginOAuth(t, svc, "google")
	flowStore.mu.Lock()
	flow := flowStore.flows[start.FlowID]
	flow.ExpiresAt = time.Now().Add(-time.Second)
	flowStore.flows[start.FlowID] = flow
	flowStore.mu.Unlock()
	if _, err := svc.OAuthCallback(t.Context(), "google", url.Values{"code": {"x"}, "state": {state}}, start.BrowserBinding); !errors.Is(err, domain.ErrInvalidState) {
		t.Fatalf("expired flow error=%v, want ErrInvalidState", err)
	}
}

func TestOAuth_ExchangeTokenRequiresOriginatingBrowserVerifierWithoutConsumingOnMismatch(t *testing.T) {
	users := newMockUserStore()
	bus := &mockBus{}
	store := newMockExchange()
	svc := newOAuth(newMockProvider(), users, bus, store)
	callback, verifier := finishOAuth(t, svc, "google")
	wrongVerifier, _ := testOAuthVerifier(2)

	if _, err := svc.ExchangeToken(t.Context(), callback.ExchangeCode, wrongVerifier); !errors.Is(err, domain.ErrInvalidExchange) {
		t.Fatalf("wrong verifier error=%v, want ErrInvalidExchange", err)
	}
	result, err := svc.ExchangeToken(t.Context(), callback.ExchangeCode, verifier)
	if err != nil {
		t.Fatalf("correct verifier exchange: %v", err)
	}
	if result.Token.Value == "" {
		t.Fatal("expected non-empty token")
	}
	if got := len(bus.Got(event.KindUserSignedUp)); got != 1 {
		t.Errorf("UserSignedUp count=%d, want 1", got)
	}
	if got := len(bus.Got(event.KindUserLoggedIn)); got != 1 {
		t.Errorf("UserLoggedIn count=%d, want 1", got)
	}
	if _, err := svc.ExchangeToken(t.Context(), callback.ExchangeCode, verifier); !errors.Is(err, domain.ErrInvalidExchange) {
		t.Fatalf("reused exchange error=%v, want ErrInvalidExchange", err)
	}
}

func TestOAuth_ExchangeCodeIsSingleUseUnderConcurrency(t *testing.T) {
	svc := newOAuth(newMockProvider(), newMockUserStore(), &mockBus{}, newMockExchange())
	callback, verifier := finishOAuth(t, svc, "google")
	var successes atomic.Int32
	var invalid atomic.Int32
	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.ExchangeToken(context.Background(), callback.ExchangeCode, verifier)
			switch {
			case err == nil:
				successes.Add(1)
			case errors.Is(err, domain.ErrInvalidExchange):
				invalid.Add(1)
			default:
				t.Errorf("unexpected exchange error: %v", err)
			}
		}()
	}
	wg.Wait()
	if successes.Load() != 1 || invalid.Load() != 15 {
		t.Fatalf("success=%d invalid=%d, want 1/15", successes.Load(), invalid.Load())
	}
}

func TestOAuth_FailsClosedWithoutFlowStore(t *testing.T) {
	p := newMockProvider()
	_, challenge := testOAuthVerifier(1)
	svc := NewOAuthService(OAuthDeps{Providers: map[string]port.IdentityProvider{"google": p}})
	if _, err := svc.StartOAuth(t.Context(), "google", challenge); !errors.Is(err, domain.ErrOAuthFlowUnavailable) {
		t.Fatalf("StartOAuth error=%v, want ErrOAuthFlowUnavailable", err)
	}
}
