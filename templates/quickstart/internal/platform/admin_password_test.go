package platform

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	authevent "github.com/brizenchi/go-modules/modules/auth/event"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

const testAdminPassword = "Isolated-admin-pass-42!"

func adminPasswordConfig() Config {
	return Config{
		ServiceName: "admin-password-test",
		Auth: AuthConfig{
			UserJWTSecret: "isolated-admin-password-jwt-key",
			AdminEmail:    " Owner@Example.test ", AdminPassword: testAdminPassword,
			AdminEmails: []string{"other-admin@example.test"},
			Email:       AuthEmailConfig{Enabled: boolPointer(false)},
			Google:      GoogleConfig{Enabled: boolPointer(false)},
			GitHub:      GitHubConfig{Enabled: boolPointer(false)},
		},
		Email:    EmailConfig{Provider: "none"},
		Billing:  BillingConfig{Enabled: boolPointer(false)},
		Referral: ReferralConfig{Enabled: boolPointer(false)},
	}
}

func newAdminPasswordTestServer(t *testing.T, cfg Config) (*Modules, *gin.Engine) {
	t.Helper()
	db := openPlatformTestDB(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := Migrate(db, cfg); err != nil {
		t.Fatal(err)
	}
	modules, err := New(db, cfg)
	if err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	modules.Mount(router.Group("/api/v1"), router.Group("/api/v1", modules.RequireUser()))
	router.GET("/api/v1/admin/protected", modules.RequireAdmin(), func(c *gin.Context) { c.Status(http.StatusOK) })
	return modules, router
}

func adminPasswordRequest(router http.Handler, body, peer string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/admin/login", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.RemoteAddr = peer + ":12345"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	return w
}

func TestAdminPasswordCreatesRealAdminAndRefreshPreservesRole(t *testing.T) {
	modules, router := newAdminPasswordTestServer(t, adminPasswordConfig())
	var signups, logins int
	modules.Auth.Subscribe(authevent.KindUserSignedUp, func(_ context.Context, event authevent.Envelope) error {
		signups++
		return nil
	})
	modules.Auth.Subscribe(authevent.KindUserLoggedIn, func(_ context.Context, event authevent.Envelope) error {
		logins++
		return nil
	})
	body := `{"email":" OWNER@example.test ","password":"` + testAdminPassword + `"}`
	var id string
	for index := 0; index < 2; index++ {
		w := adminPasswordRequest(router, body, "192.0.2.1")
		if w.Code != http.StatusOK || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("login failed: status=%d body=%s", w.Code, w.Body)
		}
		var result struct {
			Data struct {
				Token string `json:"token"`
				User  struct {
					ID    string `json:"id"`
					Email string `json:"email"`
					Role  string `json:"role"`
					IsNew bool   `json:"is_new"`
				} `json:"user"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if result.Data.User.Role != "admin" || result.Data.User.Email != "owner@example.test" || result.Data.User.IsNew != (index == 0) {
			t.Fatalf("unexpected user: %+v", result.Data.User)
		}
		if index == 1 && result.Data.User.ID != id {
			t.Fatal("second login created a different account")
		}
		id = result.Data.User.ID
		parsed, err := modules.Auth.Session.VerifyToken(result.Data.Token)
		if err != nil || string(parsed.Role) != "admin" || parsed.UserID != id {
			t.Fatalf("invalid admin JWT: %+v %v", parsed, err)
		}
		for _, route := range []struct{ method, path string }{{http.MethodGet, "/admin/protected"}, {http.MethodPost, "/auth/refresh"}} {
			r := httptest.NewRequest(route.method, "/api/v1"+route.path, nil)
			r.Header.Set("Authorization", "Bearer "+result.Data.Token)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			if w.Code != http.StatusOK {
				t.Fatalf("%s failed: %d %s", route.path, w.Code, w.Body)
			}
			if route.path == "/auth/refresh" && !strings.Contains(w.Body.String(), `"role":"admin"`) {
				t.Fatalf("refresh lost admin role: %s", w.Body)
			}
		}
		if strings.Contains(w.Body.String(), testAdminPassword) {
			t.Fatal("password leaked in response")
		}
	}
	if signups != 1 || logins != 2 {
		t.Fatalf("hooks: signups=%d logins=%d", signups, logins)
	}
	account, err := modules.Users.FindByID(t.Context(), id)
	if err != nil || account.LastLoginAt == nil || !account.EmailVerified {
		t.Fatalf("login bookkeeping not saved: %+v %v", account, err)
	}
	if modules.Config.Auth.AdminPassword != "" {
		t.Fatal("raw password retained in Modules.Config")
	}
}

func TestAdminPasswordOnlyAuthenticatesConfiguredEmailAndNeverCreatesFailedUsers(t *testing.T) {
	modules, router := newAdminPasswordTestServer(t, adminPasswordConfig())
	var previous string
	for index, pair := range []struct{ email, password string }{
		{"owner@example.test", "incorrect-pass-42!"},
		{"other-admin@example.test", testAdminPassword},
		{"member@example.test", testAdminPassword},
		{"not-an-email", testAdminPassword},
		{"owner@example.test", ""},
		{"owner@example.test", strings.Repeat("a", 73)},
	} {
		body, _ := json.Marshal(pairToBody(pair.email, pair.password))
		w := adminPasswordRequest(router, string(body), fmt.Sprintf("192.0.2.%d", index+1))
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("case%d status=%d body=%s", index, w.Code, w.Body)
		}
		if previous != "" && w.Body.String() != previous {
			t.Fatal("wrong account and wrong password errors differ")
		}
		previous = w.Body.String()
	}
	var count int64
	if err := modules.DB.Table("users").Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("failed login created users: %d %v", count, err)
	}
}

func pairToBody(email, password string) map[string]string {
	return map[string]string{"email": email, "password": password}
}

func TestAdminPasswordCapabilitiesAndDisabledEndpoint(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprint(enabled), func(t *testing.T) {
			cfg := adminPasswordConfig()
			if !enabled {
				cfg.Auth.AdminEmail, cfg.Auth.AdminPassword = "", ""
			}
			_, router := newAdminPasswordTestServer(t, cfg)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/capabilities", nil))
			if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), fmt.Sprintf(`"admin_password_enabled":%t`, enabled)) {
				t.Fatalf("capabilities: %s", w.Body)
			}
			if strings.Contains(w.Body.String(), "owner@example") || strings.Contains(w.Body.String(), testAdminPassword) {
				t.Fatal("capabilities exposed credentials")
			}
			if !enabled {
				w = adminPasswordRequest(router, `{}`, "192.0.2.1")
				if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), "admin password login is not configured") {
					t.Fatalf("disabled login: %d %s", w.Code, w.Body)
				}
			}
		})
	}
}

func TestAdminPasswordHTTPRateLimitIgnoresForwardedHeadersAndLimitsBody(t *testing.T) {
	_, router := newAdminPasswordTestServer(t, adminPasswordConfig())
	for i := 0; i < adminLoginIPLimit+1; i++ {
		r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/admin/login", strings.NewReader(`{"password":"`+strings.Repeat("x", 40<<10)+`"}`))
		r.RemoteAddr = "192.0.2.10:12345"
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-Forwarded-For", fmt.Sprintf("198.51.100.%d", i+1))
		r.Header.Set("X-Real-IP", fmt.Sprintf("198.51.100.%d", i+1))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		want := http.StatusRequestEntityTooLarge
		if i == adminLoginIPLimit {
			want = http.StatusTooManyRequests
		}
		if w.Code != want {
			t.Fatalf("attempt%d status=%d want=%d body=%s", i, w.Code, want, w.Body)
		}
		if want == http.StatusTooManyRequests && w.Header().Get("Retry-After") == "" {
			t.Fatal("retry-after missing")
		}
	}
}

func TestAdminPasswordLimiterBoundsDistributedAndConcurrentAttempts(t *testing.T) {
	var limiter adminLoginLimiter
	started := time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC)
	var accepted atomic.Int64
	var group sync.WaitGroup
	for i := 0; i < 200; i++ {
		group.Add(1)
		go func(i int) {
			defer group.Done()
			if limiter.reserve(started, fmt.Sprint(i)) == 0 {
				accepted.Add(1)
			}
		}(i)
	}
	group.Wait()
	if accepted.Load() != adminLoginGlobalLimit || len(limiter.peers) > adminLoginGlobalLimit {
		t.Fatalf("accepted=%d peers=%d", accepted.Load(), len(limiter.peers))
	}
	if limiter.reserve(started.Add(time.Second), "new-peer") <= 0 {
		t.Fatal("distributed attempts bypassed shared limit")
	}
	if limiter.reserve(started.Add(adminLoginWindow), "new-peer") != 0 || len(limiter.peers) != 1 {
		t.Fatal("expired limiter did not reset")
	}
}

func TestAdminPasswordVerifierDoesNotTruncateOrTrimPassword(t *testing.T) {
	password := " " + strings.Repeat("x", 70) + " "
	hash, err := bcrypt.GenerateFromPassword([]byte(password), adminPasswordCost)
	if err != nil {
		t.Fatal(err)
	}
	verifier := &adminPasswordVerifier{emailHash: sha256.Sum256([]byte("admin@example.test")), hash: hash}
	if err := verifier.Verify(t.Context(), "ADMIN@example.test", password); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{password + "x", strings.TrimSpace(password)} {
		if err := verifier.Verify(t.Context(), "admin@example.test", bad); !errors.Is(err, errInvalidAdminCredentials) {
			t.Fatalf("altered password accepted: %v", err)
		}
	}
	cost, err := bcrypt.Cost(hash)
	if err != nil || cost < 12 {
		t.Fatalf("bcrypt cost=%d err=%v", cost, err)
	}
}

func TestValidateAdminPasswordConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name, email, password string
		disabled, valid       bool
	}{
		{name: "disabled", valid: true},
		{name: "valid", email: " Owner@example.test ", password: testAdminPassword, valid: true},
		{name: "missing email", password: testAdminPassword},
		{name: "missing password", email: "admin@example.test"},
		{name: "invalid email", email: "not-an-email", password: testAdminPassword},
		{name: "display name", email: "Admin <admin@example.test>", password: testAdminPassword},
		{name: "short password", email: "admin@example.test", password: "short"},
		{name: "blank password", email: "admin@example.test", password: strings.Repeat(" ", 12)},
		{name: "overlong password", email: "admin@example.test", password: strings.Repeat("a", 73)},
		{name: "auth disabled", email: "admin@example.test", password: testAdminPassword, disabled: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Config{Auth: AuthConfig{AdminEmail: tc.email, AdminPassword: tc.password}}
			if tc.disabled {
				cfg.Auth.Enabled = boolPointer(false)
			}
			err := cfg.ValidateAdminPassword()
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%t err=%v", tc.valid, err)
			}
			if err != nil && tc.password != "" && strings.Contains(err.Error(), tc.password) {
				t.Fatal("config error leaked password")
			}
		})
	}
}
