package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	authdomain "github.com/brizenchi/go-modules/modules/auth/domain"
	"github.com/brizenchi/quickstart-template/internal/feature/operations"
	"github.com/brizenchi/quickstart-template/internal/hostapi"
	apphttp "github.com/brizenchi/quickstart-template/internal/http"
	"github.com/brizenchi/quickstart-template/internal/platform"
	"github.com/brizenchi/quickstart-template/internal/user"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestEnvironmentAdminPasswordSessionWithHostRoutes(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	cfg := platform.Config{
		ServiceName: "admin-password-integration",
		Auth: platform.AuthConfig{
			UserJWTSecret: "isolated-admin-session-signing-secret",
			AdminEmail:    "operator@example.test", AdminPassword: "Isolated-admin-route-pass!",
			Email:  platform.AuthEmailConfig{Enabled: boolPtr(false)},
			Google: platform.GoogleConfig{Enabled: boolPtr(false)}, GitHub: platform.GitHubConfig{Enabled: boolPtr(false)},
		},
		Email: platform.EmailConfig{Provider: "none"}, Billing: platform.BillingConfig{Enabled: boolPtr(false)}, Referral: platform.ReferralConfig{Enabled: boolPtr(false)},
	}
	if err := platform.Migrate(db, cfg); err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(operations.Models()...); err != nil {
		t.Fatal(err)
	}
	modules, err := platform.New(db, cfg)
	if err != nil {
		t.Fatal(err)
	}
	engine := BuildRouter(RouterConfig{ServiceName: "admin-password-integration", AllowedOrigins: []string{"http://localhost:3100"}}, apphttp.NewRouter(modules, hostapi.Deps{DB: db, Modules: modules, Users: modules.Users}))
	call := func(method, path, body, token string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "/api/v1"+path, strings.NewReader(body))
		r.RemoteAddr = "192.0.2.1:12345"
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Origin", "http://localhost:3100")
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, r)
		return w
	}
	response := call(http.MethodPost, "/auth/admin/login", `{"email":"operator@example.test","password":"Isolated-admin-route-pass!"}`, "")
	if response.Code != http.StatusOK {
		t.Fatalf("admin login: %d %s", response.Code, response.Body)
	}
	if response.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3100" {
		t.Fatal("admin login CORS did not allow frontend")
	}
	var session struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &session); err != nil {
		t.Fatal(err)
	}
	if session.Data.Token == "" {
		t.Fatal("no session token")
	}
	for _, path := range []string{"/admin/settings", "/account/profile"} {
		w := call(http.MethodGet, path, "", session.Data.Token)
		if w.Code != http.StatusOK {
			t.Fatalf("admin session cannot read %s: %d %s", path, w.Code, w.Body)
		}
		if strings.Contains(w.Body.String(), cfg.Auth.AdminPassword) {
			t.Fatal("host endpoint leaked password")
		}
	}
	w := call(http.MethodPost, "/auth/refresh", "", session.Data.Token)
	if w.Code != http.StatusOK {
		t.Fatalf("refresh: %d %s", w.Code, w.Body)
	}
	if err := json.Unmarshal(w.Body.Bytes(), &session); err != nil {
		t.Fatal(err)
	}
	w = call(http.MethodGet, "/admin/settings", "", session.Data.Token)
	if w.Code != http.StatusOK {
		t.Fatalf("refreshed token lost admin access: %d %s", w.Code, w.Body)
	}
	member := &user.User{Email: "member@example.test", EmailVerified: true}
	if err := modules.Users.Create(t.Context(), member); err != nil {
		t.Fatal(err)
	}
	memberToken, err := modules.Auth.Deps.TokenSigner.Issue(authdomain.Identity{UserID: member.ID, Email: member.Email, Role: authdomain.RoleUser}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		token  string
		status int
	}{{"", http.StatusUnauthorized}, {memberToken.Value, http.StatusForbidden}} {
		w := call(http.MethodGet, "/admin/settings", "", tc.token)
		if w.Code != tc.status {
			t.Fatalf("admin boundary: got%d want%d %s", w.Code, tc.status, w.Body)
		}
	}
	w = call(http.MethodPost, "/auth/admin/login", `{"email":"member@example.test","password":"Isolated-admin-route-pass!"}`, memberToken.Value)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("member used admin password: %d %s", w.Code, w.Body)
	}
}
