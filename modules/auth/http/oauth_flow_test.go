package http

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/brizenchi/go-modules/modules/auth/adapter/memstore"
	"github.com/brizenchi/go-modules/modules/auth/app"
	"github.com/brizenchi/go-modules/modules/auth/domain"
	"github.com/brizenchi/go-modules/modules/auth/port"
	"github.com/gin-gonic/gin"
)

type httpTestProvider struct {
	mu          sync.Mutex
	stateCount  int
	exchangeErr error
}

func (p *httpTestProvider) Name() domain.Provider { return domain.ProviderGoogle }
func (p *httpTestProvider) IssueState() (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stateCount++
	return fmt.Sprintf("signed-state-%d", p.stateCount), nil
}
func (p *httpTestProvider) VerifyState(state string) error {
	if !strings.HasPrefix(state, "signed-state-") {
		return domain.ErrInvalidState
	}
	return nil
}
func (p *httpTestProvider) OAuthStateTTL() time.Duration { return 10 * time.Minute }
func (p *httpTestProvider) AuthorizeURL(state string, _ url.Values) (string, error) {
	return "https://provider.example/authorize?state=" + url.QueryEscape(state), nil
}
func (p *httpTestProvider) Exchange(_ context.Context, _ url.Values) (*domain.OAuthProfile, error) {
	if p.exchangeErr != nil {
		return nil, p.exchangeErr
	}
	return &domain.OAuthProfile{
		Provider: domain.ProviderGoogle, Subject: "subject-1", Email: "user@example.com",
	}, nil
}

type httpTestUsers struct{ identity domain.Identity }

func (u *httpTestUsers) FindByEmail(context.Context, string) (*domain.Identity, error) {
	return nil, domain.ErrUserNotFound
}
func (u *httpTestUsers) FindOrCreateByEmail(context.Context, string) (*domain.Identity, error) {
	copy := u.identity
	return &copy, nil
}
func (u *httpTestUsers) FindOrCreateFromOAuth(context.Context, domain.OAuthProfile) (*domain.Identity, error) {
	copy := u.identity
	return &copy, nil
}
func (u *httpTestUsers) FindByID(context.Context, string) (*domain.Identity, error) {
	copy := u.identity
	return &copy, nil
}
func (*httpTestUsers) MarkLogin(context.Context, string) error { return nil }

type httpTestSigner struct{}

func (httpTestSigner) Issue(id domain.Identity, ttl time.Duration) (*domain.Token, error) {
	return &domain.Token{Value: "jwt-" + id.UserID, ExpiresAt: time.Now().Add(ttl)}, nil
}
func (httpTestSigner) Parse(string) (*domain.Identity, error) { return nil, domain.ErrInvalidToken }

func newOAuthHTTPHandler(provider *httpTestProvider, secure bool) *Handler {
	users := &httpTestUsers{identity: domain.Identity{
		UserID: "user-1", Email: "user@example.com", Provider: domain.ProviderGoogle, IsNew: true,
	}}
	service := app.NewOAuthService(app.OAuthDeps{
		Providers: map[string]port.IdentityProvider{"google": provider},
		Users:     users, Signer: httpTestSigner{},
		ExchangeStore: memstore.NewExchangeStore(), FlowStore: memstore.NewOAuthFlowStore(),
	})
	return NewHandler(Deps{
		OAuth: service, FrontendURL: "https://frontend.example/login", OAuthCookieSecure: secure,
	})
}

func oauthHTTPChallenge() (string, string) {
	verifier := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{7}, 32))
	sum := sha256.Sum256([]byte(verifier))
	return verifier, base64.RawURLEncoding.EncodeToString(sum[:])
}

type authorizeEnvelope struct {
	Data struct {
		RedirectURL string `json:"redirect_url"`
	} `json:"data"`
}

func startOAuthHTTP(t *testing.T, router http.Handler, redirect bool) (*http.Cookie, string, *httptest.ResponseRecorder) {
	t.Helper()
	_, challenge := oauthHTTPChallenge()
	path := "/api/v1/auth/google/authorize?challenge=" + url.QueryEscape(challenge)
	if redirect {
		path += "&redirect=1"
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
	if redirect {
		if response.Code != http.StatusFound {
			t.Fatalf("authorize redirect status=%d body=%s", response.Code, response.Body.String())
		}
	} else if response.Code != http.StatusOK {
		t.Fatalf("authorize status=%d body=%s", response.Code, response.Body.String())
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("authorize cookies=%d headers=%v", len(cookies), response.Header())
	}
	providerURL := response.Header().Get("Location")
	if !redirect {
		var body authorizeEnvelope
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		providerURL = body.Data.RedirectURL
	}
	parsed, err := url.Parse(providerURL)
	if err != nil {
		t.Fatal(err)
	}
	return cookies[0], parsed.Query().Get("state"), response
}

func oauthHTTPRouter(handler *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/auth/:provider/authorize", handler.StartOAuth)
	router.GET("/api/v1/auth/:provider/callback", handler.OAuthCallback)
	router.POST("/api/v1/auth/exchange-token", handler.ExchangeToken)
	return router
}

func TestStartOAuthSetsSecureBrowserFlowCookieAndSupportsRedirectMode(t *testing.T) {
	router := oauthHTTPRouter(newOAuthHTTPHandler(&httpTestProvider{}, true))
	cookie, _, response := startOAuthHTTP(t, router, true)
	if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("unsafe OAuth cookie: %#v", cookie)
	}
	if cookie.Path != "/api/v1/auth/google/callback" || cookie.MaxAge <= 0 || cookie.Value == "" {
		t.Fatalf("unexpected OAuth cookie scope/lifetime: %#v", cookie)
	}
	if strings.Contains(response.Body.String(), cookie.Value) {
		t.Fatal("browser binding leaked in response body")
	}
}

func TestStartOAuthLocalHTTPCookieIsNotSecure(t *testing.T) {
	router := oauthHTTPRouter(newOAuthHTTPHandler(&httpTestProvider{}, false))
	cookie, _, _ := startOAuthHTTP(t, router, false)
	if cookie.Secure || !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("local cookie=%#v", cookie)
	}
}

func TestStartOAuthParallelFlowsUseIndependentCookieNames(t *testing.T) {
	router := oauthHTTPRouter(newOAuthHTTPHandler(&httpTestProvider{}, true))
	first, firstState, _ := startOAuthHTTP(t, router, false)
	second, secondState, _ := startOAuthHTTP(t, router, false)
	if firstState == secondState || first.Name == second.Name || first.Value == second.Value {
		t.Fatalf("parallel flows collided: first=%#v/%q second=%#v/%q", first, firstState, second, secondState)
	}
}

func TestOAuthCallbackRejectsWrongBrowserClearsCookieAndPreservesLegitimateFlow(t *testing.T) {
	router := oauthHTTPRouter(newOAuthHTTPHandler(&httpTestProvider{}, true))
	cookie, state, _ := startOAuthHTTP(t, router, false)
	callbackPath := "/api/v1/auth/google/callback?code=code&state=" + url.QueryEscape(state)

	wrong := *cookie
	wrong.Value = "wrong-browser-binding"
	request := httptest.NewRequest(http.MethodGet, callbackPath, nil)
	request.AddCookie(&wrong)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("wrong browser status=%d body=%s", response.Code, response.Body.String())
	}
	assertOAuthCookieCleared(t, response, cookie.Name)

	request = httptest.NewRequest(http.MethodGet, callbackPath, nil)
	request.AddCookie(cookie)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusFound || !strings.Contains(response.Header().Get("Location"), "?code=") {
		t.Fatalf("legitimate callback status=%d location=%q body=%s", response.Code, response.Header().Get("Location"), response.Body.String())
	}
	assertOAuthCookieCleared(t, response, cookie.Name)

	// Even manually replaying the original request cookie cannot reuse state.
	request = httptest.NewRequest(http.MethodGet, callbackPath, nil)
	request.AddCookie(cookie)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("replay status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestOAuthCallbackProviderErrorStillClearsFlowCookie(t *testing.T) {
	router := oauthHTTPRouter(newOAuthHTTPHandler(&httpTestProvider{exchangeErr: errors.New("provider failed")}, true))
	cookie, state, _ := startOAuthHTTP(t, router, false)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/callback?code=x&state="+url.QueryEscape(state), nil)
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("callback status=%d body=%s", response.Code, response.Body.String())
	}
	assertOAuthCookieCleared(t, response, cookie.Name)
}

func TestOAuthCallbackDenialRedirectsSafelyAndClearsOnlyInitiatingFlow(t *testing.T) {
	providerErr := fmt.Errorf("sensitive provider detail: %w", domain.ErrOAuthDenied)
	router := oauthHTTPRouter(newOAuthHTTPHandler(&httpTestProvider{exchangeErr: providerErr}, true))
	cookie, state, _ := startOAuthHTTP(t, router, false)
	_, challenge := oauthHTTPChallenge()
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/auth/google/callback?error=access_denied&error_description=private&state="+url.QueryEscape(state),
		nil,
	)
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusFound {
		t.Fatalf("callback status=%d body=%s", response.Code, response.Body.String())
	}
	location := response.Header().Get("Location")
	parsed, err := url.Parse(location)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Query().Get("oauth_error") != "access_denied" || parsed.Query().Get("flow") != challenge {
		t.Fatalf("redirect query=%v", parsed.Query())
	}
	if parsed.Query().Has("code") || strings.Contains(location, "private") || strings.Contains(location, "sensitive") {
		t.Fatalf("unsafe denial redirect=%q", location)
	}
	assertOAuthCookieCleared(t, response, cookie.Name)
}

func TestOAuthCallbackFailureUsesStableCodeAndNoFrontendReturnsSafe400(t *testing.T) {
	t.Run("configured frontend", func(t *testing.T) {
		handler := newOAuthHTTPHandler(&httpTestProvider{exchangeErr: fmt.Errorf("secret: %w", domain.ErrOAuthCallback)}, true)
		router := oauthHTTPRouter(handler)
		cookie, state, _ := startOAuthHTTP(t, router, false)
		request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/callback?error=server_error&state="+url.QueryEscape(state), nil)
		request.AddCookie(cookie)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		if response.Code != http.StatusFound {
			t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
		}
		location := response.Header().Get("Location")
		if !strings.Contains(location, "oauth_error=callback_failed") || strings.Contains(location, "secret") {
			t.Fatalf("unsafe callback failure redirect=%q", location)
		}
		assertOAuthCookieCleared(t, response, cookie.Name)
	})

	t.Run("no frontend", func(t *testing.T) {
		handler := newOAuthHTTPHandler(&httpTestProvider{exchangeErr: fmt.Errorf("secret: %w", domain.ErrOAuthDenied)}, true)
		handler.frontendU = ""
		router := oauthHTTPRouter(handler)
		cookie, state, _ := startOAuthHTTP(t, router, false)
		request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/callback?error=access_denied&state="+url.QueryEscape(state), nil)
		request.AddCookie(cookie)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		if response.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
		}
		if strings.Contains(response.Body.String(), "secret") || !strings.Contains(response.Body.String(), domain.ErrOAuthDenied.Error()) {
			t.Fatalf("unsafe callback failure body=%s", response.Body.String())
		}
	})
}

func TestOAuthCallbackCSRFErrorIsNotRedirectedAsProviderDenial(t *testing.T) {
	router := oauthHTTPRouter(newOAuthHTTPHandler(&httpTestProvider{}, true))
	cookie, state, _ := startOAuthHTTP(t, router, false)
	cookie.Value = "wrong-browser"
	request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/callback?error=access_denied&state="+url.QueryEscape(state), nil)
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || response.Header().Get("Location") != "" {
		t.Fatalf("CSRF error confused with denial: status=%d location=%q body=%s", response.Code, response.Header().Get("Location"), response.Body.String())
	}
}

func TestOAuthFrontendRedirectPreservesExistingQueryAndFragment(t *testing.T) {
	got, err := oauthFrontendRedirect(
		"https://frontend.example/login?source=oauth#continue", "exchange code", "flow/id",
	)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Query().Get("source") != "oauth" || parsed.Query().Get("code") != "exchange code" || parsed.Query().Get("flow") != "flow/id" {
		t.Fatalf("redirect query=%v", parsed.Query())
	}
	if parsed.Fragment != "continue" {
		t.Fatalf("fragment=%q, want continue", parsed.Fragment)
	}
}

func TestOAuthFrontendErrorRedirectPreservesExistingQueryAndFragment(t *testing.T) {
	got, err := oauthFrontendErrorRedirect(
		"https://frontend.example/login?source=oauth&code=stale#continue", "access_denied", "flow/id",
	)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Query().Get("source") != "oauth" || parsed.Query().Get("oauth_error") != "access_denied" || parsed.Query().Get("flow") != "flow/id" {
		t.Fatalf("redirect query=%v", parsed.Query())
	}
	if parsed.Query().Has("code") || parsed.Fragment != "continue" {
		t.Fatalf("redirect=%q", got)
	}
}

func assertOAuthCookieCleared(t *testing.T, response *httptest.ResponseRecorder, name string) {
	t.Helper()
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == name && cookie.MaxAge < 0 && cookie.Value == "" {
			return
		}
	}
	t.Fatalf("cookie %q was not cleared: %v", name, response.Header().Values("Set-Cookie"))
}
