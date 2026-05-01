package saascore

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	authmodule "github.com/brizenchi/go-modules/modules/auth"
	authjwt "github.com/brizenchi/go-modules/modules/auth/adapter/jwt"
	authapp "github.com/brizenchi/go-modules/modules/auth/app"
	authdomain "github.com/brizenchi/go-modules/modules/auth/domain"
	authevent "github.com/brizenchi/go-modules/modules/auth/event"
	authport "github.com/brizenchi/go-modules/modules/auth/port"
	"github.com/gin-gonic/gin"
)

type routeTestVerifier struct {
	wantCode string
	lastCode string
	lastCtx  context.Context
	err      error
}

func (v *routeTestVerifier) Verify(ctx context.Context, _ string, code string) error {
	v.lastCtx = ctx
	v.lastCode = code
	if v.err != nil {
		return v.err
	}
	if code != v.wantCode {
		return authdomain.ErrInvalidCode
	}
	return nil
}

type routeTestUsers struct {
	byEmail map[string]*authdomain.Identity
	byID    map[string]*authdomain.Identity
}

func newRouteTestUsers() *routeTestUsers {
	return &routeTestUsers{
		byEmail: map[string]*authdomain.Identity{},
		byID:    map[string]*authdomain.Identity{},
	}
}

func (u *routeTestUsers) seed(id authdomain.Identity) {
	cp := id
	u.byEmail[id.Email] = &cp
	u.byID[id.UserID] = &cp
}

func (u *routeTestUsers) FindByEmail(_ context.Context, email string) (*authdomain.Identity, error) {
	if id, ok := u.byEmail[email]; ok {
		cp := *id
		cp.IsNew = false
		return &cp, nil
	}
	return nil, authdomain.ErrUserNotFound
}

func (u *routeTestUsers) FindOrCreateByEmail(_ context.Context, email string) (*authdomain.Identity, error) {
	if id, ok := u.byEmail[email]; ok {
		cp := *id
		cp.IsNew = false
		return &cp, nil
	}
	id := &authdomain.Identity{
		UserID: "u-" + email,
		Email:  email,
		Role:   authdomain.RoleUser,
		IsNew:  true,
	}
	u.byEmail[email] = id
	u.byID[id.UserID] = id
	return id, nil
}

func (u *routeTestUsers) FindOrCreateFromOAuth(_ context.Context, profile authdomain.OAuthProfile) (*authdomain.Identity, error) {
	if id, ok := u.byEmail[profile.Email]; ok {
		cp := *id
		cp.IsNew = false
		return &cp, nil
	}
	id := &authdomain.Identity{
		UserID:   "u-" + profile.Email,
		Email:    profile.Email,
		Provider: profile.Provider,
		Subject:  profile.Subject,
		Role:     authdomain.RoleUser,
		IsNew:    true,
	}
	u.byEmail[profile.Email] = id
	u.byID[id.UserID] = id
	return id, nil
}

func (u *routeTestUsers) FindByID(_ context.Context, userID string) (*authdomain.Identity, error) {
	if id, ok := u.byID[userID]; ok {
		cp := *id
		cp.IsNew = false
		return &cp, nil
	}
	return nil, authdomain.ErrUserNotFound
}

func (u *routeTestUsers) MarkLogin(_ context.Context, _ string) error {
	return nil
}

type routeTestExchangeStore struct {
	record  *authdomain.ExchangeCode
	lastCtx context.Context
}

func (s *routeTestExchangeStore) Save(ctx context.Context, code authdomain.ExchangeCode) error {
	s.lastCtx = ctx
	cp := code
	s.record = &cp
	return nil
}

func (s *routeTestExchangeStore) Consume(ctx context.Context, code string) (*authdomain.ExchangeCode, error) {
	s.lastCtx = ctx
	if s.record == nil || s.record.Code != code || time.Now().UTC().After(s.record.ExpiresAt) {
		return nil, authdomain.ErrInvalidExchange
	}
	cp := *s.record
	s.record = nil
	return &cp, nil
}

type routeTestBus struct{}

func (routeTestBus) Subscribe(kind authevent.Kind, fn authport.Listener) {}
func (routeTestBus) Publish(ctx context.Context, env authevent.Envelope) {}

type routeTestProvider struct {
	profile *authdomain.OAuthProfile
	lastCtx context.Context
}

func (p *routeTestProvider) Name() authdomain.Provider { return authdomain.ProviderGoogle }
func (p *routeTestProvider) AuthorizeURL(state string, extra url.Values) (string, error) {
	return "https://example.com/oauth?state=" + state, nil
}
func (p *routeTestProvider) Exchange(ctx context.Context, _ url.Values) (*authdomain.OAuthProfile, error) {
	p.lastCtx = ctx
	return p.profile, nil
}
func (p *routeTestProvider) VerifyState(state string) error { return nil }
func (p *routeTestProvider) IssueState() (string, error)    { return "state-1", nil }

type routeHarness struct {
	stack         *Stack
	verifier      *routeTestVerifier
	exchangeStore *routeTestExchangeStore
	signer        *authjwt.Signer
}

func newRouteHarness(t *testing.T) *routeHarness {
	t.Helper()

	users := newRouteTestUsers()
	users.seed(authdomain.Identity{
		UserID: "u-existing",
		Email:  "existing@example.com",
		Role:   authdomain.RoleUser,
	})

	signer, err := authjwt.NewSigner(authjwt.Config{
		Secret: "route-secret",
		Issuer: "route-test",
	})
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	ticketSigner, err := authjwt.NewTicketSigner(authjwt.Config{
		Secret: "route-secret",
		Issuer: "route-test-ws",
	})
	if err != nil {
		t.Fatalf("NewTicketSigner: %v", err)
	}

	verifier := &routeTestVerifier{wantCode: "123456"}
	exchangeStore := &routeTestExchangeStore{
		record: &authdomain.ExchangeCode{
			Code:      "exch-1",
			UserID:    "u-existing",
			Provider:  authdomain.ProviderGoogle,
			IsNew:     false,
			ExpiresAt: time.Now().UTC().Add(time.Minute),
		},
	}
	provider := &routeTestProvider{
		profile: &authdomain.OAuthProfile{
			Provider: authdomain.ProviderGoogle,
			Subject:  "sub-1",
			Email:    "existing@example.com",
		},
	}

	login := authapp.NewLoginService(authapp.LoginDeps{
		Verifier: verifier,
		Users:    users,
		Signer:   signer,
		Bus:      routeTestBus{},
	})
	oauth := authapp.NewOAuthService(authapp.OAuthDeps{
		Providers: map[string]authport.IdentityProvider{
			string(authdomain.ProviderGoogle): provider,
		},
		Users:         users,
		Signer:        signer,
		ExchangeStore: exchangeStore,
		Bus:           routeTestBus{},
	})
	session := authapp.NewSessionService(authapp.SessionDeps{
		Users:        users,
		Signer:       signer,
		TicketSigner: ticketSigner,
	})

	return &routeHarness{
		stack: &Stack{
			Auth: &authmodule.Module{
				Login:   login,
				OAuth:   oauth,
				Session: session,
			},
		},
		verifier:      verifier,
		exchangeStore: exchangeStore,
		signer:        signer,
	}
}

func postJSON(t *testing.T, r http.Handler, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestWithReferralCodeRoundTrip(t *testing.T) {
	ctx := withReferralCode(context.Background(), "  REF-123  ")
	if got := referralCode(ctx); got != "REF-123" {
		t.Fatalf("referralCode = %q, want REF-123", got)
	}
}

func TestWithReferralCodeEmptyKeepsBlank(t *testing.T) {
	if got := referralCode(withReferralCode(context.Background(), "   ")); got != "" {
		t.Fatalf("referralCode = %q, want empty", got)
	}
}

func TestVerifyCodePassesReferralCodeIntoContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newRouteHarness(t)

	router := gin.New()
	router.POST("/auth/verify-code", h.stack.verifyCode())

	rec := postJSON(t, router, "/auth/verify-code", map[string]any{
		"email":         "new@example.com",
		"code":          "123456",
		"referral_code": "INV-888",
	}, nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := referralCode(h.verifier.lastCtx); got != "INV-888" {
		t.Fatalf("referralCode = %q, want INV-888", got)
	}
}

func TestExchangeTokenPassesReferralCodeIntoContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newRouteHarness(t)

	router := gin.New()
	router.POST("/auth/exchange-token", h.stack.exchangeToken())

	rec := postJSON(t, router, "/auth/exchange-token", map[string]any{
		"code":          "exch-1",
		"referral_code": "INV-999",
	}, nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := referralCode(h.exchangeStore.lastCtx); got != "INV-999" {
		t.Fatalf("referralCode = %q, want INV-999", got)
	}
}

func TestRequireUserRejectsMissingBearerToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newRouteHarness(t)

	router := gin.New()
	protected := router.Group("/")
	protected.Use(h.stack.RequireUser())
	protected.GET("/me", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestRequireUserAcceptsValidBearerToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newRouteHarness(t)

	jwtToken, err := h.signer.Issue(authdomain.Identity{
		UserID: "u-existing",
		Email:  "existing@example.com",
		Role:   authdomain.RoleUser,
	}, time.Hour)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	router := gin.New()
	protected := router.Group("/")
	protected.Use(h.stack.RequireUser())
	protected.GET("/me", func(c *gin.Context) {
		userID, ok := h.stack.userIDFromGin(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false})
			return
		}
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+jwtToken.Value)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"user_id":"u-existing"`)) {
		t.Fatalf("body = %s, want user_id", rec.Body.String())
	}
}
