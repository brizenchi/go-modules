package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/brizenchi/quickstart-template/internal/user"
)

func previewRequest(t *testing.T, app *previewApp, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var raw []byte
	if body != nil {
		var err error
		raw, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, "/api/v1"+path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	app.handler.ServeHTTP(w, req)
	return w
}
func previewData(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body.Data
}
func previewLogin(t *testing.T, app *previewApp, email, referralCode string) string {
	t.Helper()
	sent := previewData(t, previewRequest(t, app, "POST", "/auth/send-code", "", map[string]any{"email": email}))
	code, ok := sent["debug_code"].(string)
	if !ok || code == "" {
		t.Fatalf("preview code missing: %+v", sent)
	}
	verified := previewData(t, previewRequest(t, app, "POST", "/auth/verify-code", "", map[string]any{"email": email, "code": code, "referral_code": referralCode}))
	token, ok := verified["token"].(string)
	if !ok || token == "" {
		t.Fatalf("normal login did not issue a token: %+v", verified)
	}
	return token
}

func TestPreviewUsesOnlyIsolatedLocalConfigurationAndCleansUp(t *testing.T) {
	for _, key := range []string{"DATABASE_URL", "APP_DB_DSN", "APP_EMAIL_PROVIDER", "APP_BILLING_STRIPE_SECRET_KEY", "APP_AUTH_USER_JWT_SECRET", "RESEND_API_KEY"} {
		t.Setenv(key, "must-not-be-used")
	}
	app, err := newPreview("http://localhost:3100")
	if err != nil {
		t.Fatal(err)
	}
	directory := app.directory
	t.Cleanup(app.Close)
	if app.modules.DB.Dialector.Name() != "sqlite" || app.modules.Billing != nil || app.modules.Config.Email.Provider != "log" || !app.modules.Config.Auth.Email.Debug || len(app.modules.Auth.Deps.IdentityProviders) != 0 {
		t.Fatal("preview enabled a configured or external provider")
	}
	if _, err := os.Stat(directory); err != nil {
		t.Fatal(err)
	}
	if w := previewRequest(t, app, "GET", "/credits", "", nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous credits: %s", w.Body.String())
	}
	userToken := previewLogin(t, app, previewUser, "")
	if w := previewRequest(t, app, "GET", "/admin/users", userToken, nil); w.Code != http.StatusForbidden {
		t.Fatalf("ordinary user became admin: %s", w.Body.String())
	}
	if got := previewData(t, previewRequest(t, app, "GET", "/credits", userToken, nil))["balance"]; got != float64(50) {
		t.Fatalf("initial credits=%v", got)
	}
	adminToken := previewLogin(t, app, previewAdmin, "")
	if got := previewData(t, previewRequest(t, app, "GET", "/admin/users", adminToken, nil))["total"]; got != float64(2) {
		t.Fatalf("seed user count=%v", got)
	}
	if w := previewRequest(t, app, "POST", "/stripe/checkout/session", userToken, map[string]any{}); w.Code != http.StatusServiceUnavailable {
		t.Fatalf("billing unexpectedly enabled: %s", w.Body.String())
	}
	app.Close()
	if _, err := os.Stat(directory); !os.IsNotExist(err) {
		t.Fatalf("temporary preview data remained: %v", err)
	}
}

func TestPreviewRealRegistrationAttributesReferralAndRunsCreditFeature(t *testing.T) {
	app, err := newPreview("http://localhost:3100")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Close)
	adminToken := previewLogin(t, app, previewAdmin, "")
	ref := previewData(t, previewRequest(t, app, "GET", "/referral/code", adminToken, nil))
	code, ok := ref["code"].(string)
	if !ok || code == "" {
		t.Fatalf("referral response=%+v", ref)
	}
	token := previewLogin(t, app, "preview-invited@example.test", code)
	stats := previewData(t, previewRequest(t, app, "GET", "/referral/stats", adminToken, nil))
	if stats["Pending"] != float64(1) {
		t.Fatalf("signup attribution failed: %+v", stats)
	}
	if got := previewData(t, previewRequest(t, app, "GET", "/credits", token, nil))["balance"]; got != float64(50) {
		t.Fatalf("signup credits=%v", got)
	}
	created := previewData(t, previewRequest(t, app, "POST", "/notes", token, map[string]any{"title": "Paid preview note", "body": "Local content"}))
	id := created["id"].(float64)
	path := strings.TrimSuffix(jsonNumber(t, id), ".0")
	exported := previewData(t, previewRequest(t, app, "POST", "/notes/"+path+"/export", token, map[string]any{"idempotency_key": "local-preview-export", "expected_cost": 1}))
	if exported["balance"] != float64(49) || !strings.Contains(exported["content"].(string), "Local content") {
		t.Fatalf("export=%+v", exported)
	}
	// Actual referral activation uses the same credit listener without a fake
	// public payment route; billing itself remains disabled in this local command.
	account, err := app.modules.Users.FindByEmail(context.Background(), "preview-invited@example.test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.modules.Referral.Attribute.ActivateReferral(context.Background(), account.ID, 25); err != nil {
		t.Fatal(err)
	}
	admin, err := app.modules.Users.FindByEmail(context.Background(), previewAdmin)
	if err != nil {
		t.Fatal(err)
	}
	summary, err := app.modules.Users.GetCreditSummary(context.Background(), admin.ID)
	if err != nil || summary.Balance != 75 {
		t.Fatalf("referral reward=%+v err=%v", summary, err)
	}
	var grant user.CreditTransaction
	if err := app.modules.DB.Where("user_id = ? AND source = ?", admin.ID, "referral").Take(&grant).Error; err != nil || grant.Amount != 25 {
		t.Fatalf("missing real reward ledger: %+v err=%v", grant, err)
	}
}
func jsonNumber(t *testing.T, value float64) string {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestPreviewRejectsNonLocalFrontendOrigins(t *testing.T) {
	for _, raw := range []string{"https://example.com", "http://0.0.0.0:3100", "http://localhost.evil.test:3100", "http://user:pass@localhost:3100", "http://localhost:3100/path", "http://localhost:3100?key=x"} {
		if _, err := localOrigin(raw); err == nil {
			t.Fatalf("unsafe origin accepted: %q", raw)
		}
	}
	for _, raw := range []string{"http://localhost:3100", "http://127.0.0.1:3100", "http://[::1]:3100"} {
		if _, err := localOrigin(raw); err != nil {
			t.Fatalf("local origin rejected: %q: %v", raw, err)
		}
	}
}
