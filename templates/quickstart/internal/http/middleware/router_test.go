package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	authdomain "github.com/brizenchi/go-modules/modules/auth/domain"
	"github.com/brizenchi/quickstart-template/internal/feature/note"
	"github.com/brizenchi/quickstart-template/internal/hostapi"
	apphttp "github.com/brizenchi/quickstart-template/internal/http"
	"github.com/brizenchi/quickstart-template/internal/platform"
	"github.com/brizenchi/quickstart-template/internal/user"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func boolPtr(value bool) *bool { return &value }

func TestOperatorMutationPreflightAllowsIdempotencyHeader(t *testing.T) {
	engine := BuildRouter(RouterConfig{ServiceName: "operator-cors", AllowedOrigins: []string{"https://app.example.com"}}, nil)
	request := httptest.NewRequest(http.MethodOptions, "/api/v1/admin/settings", nil)
	request.Header.Set("Origin", "https://app.example.com")
	request.Header.Set("Access-Control-Request-Method", "PATCH")
	request.Header.Set("Access-Control-Request-Headers", "authorization,content-type,idempotency-key")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || response.Header().Get("Access-Control-Allow-Origin") != "https://app.example.com" {
		t.Fatalf("preflight rejected: status=%d headers=%v", response.Code, response.Header())
	}
	allowed := strings.ToLower(response.Header().Get("Access-Control-Allow-Headers"))
	for _, header := range []string{"authorization", "content-type", "idempotency-key"} {
		if !strings.Contains(allowed, header) {
			t.Fatalf("missing allowed header %s", header)
		}
	}
	if !strings.Contains(response.Header().Get("Access-Control-Allow-Methods"), "PATCH") {
		t.Fatal("PATCH preflight is not allowed")
	}
}

func TestRouterAllowsCredentialedRequestsOnlyFromConfiguredOrigin(t *testing.T) {
	engine := BuildRouter(RouterConfig{
		ServiceName: "cors-test", AllowedOrigins: []string{"https://app.example.com"},
	}, nil)
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Header.Set("Origin", "https://app.example.com")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Header().Get("Access-Control-Allow-Origin") != "https://app.example.com" ||
		response.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatalf("credentialed CORS headers=%v", response.Header())
	}

	request = httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Header.Set("Origin", "https://evil.example")
	response = httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("untrusted origin was allowed: %v", response.Header())
	}
}

func TestRealRouterProtectsDisabledBillingFallback(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	cfg := platform.Config{
		ServiceName: "router-test",
		Email:       platform.EmailConfig{Provider: "log"},
		Auth: platform.AuthConfig{
			UserJWTSecret: "router-test-jwt-secret",
			Google:        platform.GoogleConfig{Enabled: boolPtr(false)},
			GitHub:        platform.GitHubConfig{Enabled: boolPtr(false)},
		},
		Billing: platform.BillingConfig{
			Enabled:  boolPtr(false),
			Provider: "stripe",
			Stripe: platform.StripeConfig{
				SecretKey:     "sk_test_must_not_leak",
				WebhookSecret: "whsec_must_not_leak",
				Prices: platform.StripePricesConfig{
					ProMonthly: "price_must_not_leak",
				},
			},
		},
		Referral: platform.ReferralConfig{Enabled: boolPtr(false)},
	}
	if err := platform.Migrate(db, cfg); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if err := db.AutoMigrate(&note.Note{}); err != nil {
		t.Fatalf("migrate note: %v", err)
	}
	modules, err := platform.New(db, cfg)
	if err != nil {
		t.Fatalf("platform.New: %v", err)
	}
	router := apphttp.NewRouter(modules, hostapi.Deps{
		DB:      db,
		Modules: modules,
		Users:   modules.Users,
	})
	engine := BuildRouter(RouterConfig{ServiceName: "router-test", AllowedOrigins: []string{"https://app.example.com"}}, router)
	account := &user.User{Email: "user@example.com", EmailVerified: true, Username: "Router User"}
	if err := modules.Users.Create(t.Context(), account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	if err := db.Create(&note.Note{UserID: account.ID, Title: "private admin count sentinel"}).Error; err != nil {
		t.Fatalf("create note: %v", err)
	}

	response := httptest.NewRecorder()
	engine.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/stripe/subscription", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous status=%d body=%s", response.Code, response.Body.String())
	}

	token, err := modules.Auth.Deps.TokenSigner.Issue(authdomain.Identity{
		UserID: account.ID,
		Email:  account.Email,
		Role:   authdomain.RoleUser,
	}, time.Hour)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/stripe/subscription", nil)
	request.Header.Set("Authorization", "Bearer "+token.Value)
	response = httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable || !strings.Contains(response.Body.String(), "billing is not configured") {
		t.Fatalf("authenticated status=%d body=%s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/api/v1/account/profile", nil)
	request.Header.Set("Authorization", "Bearer "+token.Value)
	response = httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), account.Email) {
		t.Fatalf("profile status=%d body=%s", response.Code, response.Body.String())
	}

	// The admin role check must run before the real handler. A previous nested
	// middleware implementation called c.Next from RequireUser, exposing this
	// count to ordinary users and appending it to anonymous error responses.
	request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/notes/count", nil)
	request.Header.Set("Authorization", "Bearer "+token.Value)
	response = httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || strings.Contains(response.Body.String(), "total") {
		t.Fatalf("non-admin status=%d body=%s", response.Code, response.Body.String())
	}

	response = httptest.NewRecorder()
	engine.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/admin/notes/count", nil))
	if response.Code != http.StatusUnauthorized || strings.Contains(response.Body.String(), "total") {
		t.Fatalf("anonymous admin status=%d body=%s", response.Code, response.Body.String())
	}

	response = httptest.NewRecorder()
	engine.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/capabilities", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("capabilities status=%d body=%s", response.Code, response.Body.String())
	}
	for _, secret := range []string{"sk_test_must_not_leak", "whsec_must_not_leak", "price_must_not_leak"} {
		if strings.Contains(response.Body.String(), secret) {
			t.Fatalf("capabilities leaked %q: %s", secret, response.Body.String())
		}
	}
}

func TestRealRouterStopsProtectedHandlersWhenAuthDisabled(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	cfg := platform.Config{
		ServiceName: "auth-disabled-router-test",
		Email:       platform.EmailConfig{Provider: "none"},
		Auth:        platform.AuthConfig{Enabled: boolPtr(false)},
		Billing:     platform.BillingConfig{Enabled: boolPtr(false)},
		Referral:    platform.ReferralConfig{Enabled: boolPtr(false)},
	}
	if err := platform.Migrate(db, cfg); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if err := db.AutoMigrate(&note.Note{}); err != nil {
		t.Fatalf("migrate note: %v", err)
	}
	if err := db.Create(&note.Note{UserID: "sentinel-user", Title: "must stay private"}).Error; err != nil {
		t.Fatalf("create note: %v", err)
	}
	modules, err := platform.New(db, cfg)
	if err != nil {
		t.Fatalf("platform.New: %v", err)
	}
	router := apphttp.NewRouter(modules, hostapi.Deps{DB: db, Modules: modules, Users: modules.Users})
	assertProtectedHandlersUnavailable(t, BuildRouter(RouterConfig{ServiceName: "auth-disabled-router-test"}, router))
}

func TestRealRouterFailsClosedWithoutPlatform(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&note.Note{}); err != nil {
		t.Fatalf("migrate note: %v", err)
	}
	if err := db.Create(&note.Note{UserID: "sentinel-user", Title: "must stay private"}).Error; err != nil {
		t.Fatalf("create note: %v", err)
	}
	router := apphttp.NewRouter(nil, hostapi.Deps{DB: db})
	assertProtectedHandlersUnavailable(t, BuildRouter(RouterConfig{ServiceName: "nil-platform-router-test"}, router))
}

func assertProtectedHandlersUnavailable(t *testing.T, engine http.Handler) {
	t.Helper()
	for _, path := range []string{"/api/v1/notes", "/api/v1/admin/notes/count"} {
		response := httptest.NewRecorder()
		engine.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusServiceUnavailable {
			t.Fatalf("path=%s status=%d body=%s", path, response.Code, response.Body.String())
		}
		if strings.Contains(response.Body.String(), "total") || strings.Contains(response.Body.String(), "must stay private") {
			t.Fatalf("path=%s leaked protected handler response: %s", path, response.Body.String())
		}
	}
}
