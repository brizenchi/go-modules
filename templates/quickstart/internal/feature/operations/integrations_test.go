package operations

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"strings"
	"testing"

	"github.com/brizenchi/quickstart-template/internal/platform"
	"github.com/brizenchi/quickstart-template/internal/serviceconfig"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func integrationFixture(t *testing.T) (fixture, platform.Config) {
	t.Helper()
	f := newFixture(t)
	off := false
	env := platform.Config{Email: platform.EmailConfig{Provider: "log"}, Billing: platform.BillingConfig{Enabled: &off}}
	f.module.deps.ServiceSettings = serviceconfig.NewManager(f.db, env, false)
	if _, err := f.module.deps.ServiceSettings.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	return f, env
}

func TestIntegrationAuthorizationAndWriteOnlyAudit(t *testing.T) {
	f, _ := integrationFixture(t)
	for _, role := range []string{"", "user"} {
		want := 403
		if role == "" {
			want = 401
		}
		for _, method := range []string{"GET", "PATCH"} {
			path := "/admin/integrations"
			if method == "PATCH" {
				path += "/resend"
			}
			if w := request(f, method, path, `{}`, role, "unauthorized-key"); w.Code != want {
				t.Fatalf("%s role=%q returned%d", method, role, w.Code)
			}
		}
	}
	const secret = "re_StoredPlaintextCredential"
	body := `{"version":0,"source":"database","enabled":true,"fields":{"sender_email":"hello@example.com","sender_name":"Launch","email_auth_enabled":"true"},"secrets":{"api_key":"` + secret + `"},"reason":"Configure mail ` + secret + `"}`
	if w := request(f, "PATCH", "/admin/integrations/resend", body, "admin", ""); w.Code != 400 {
		t.Fatal("missing idempotency key accepted")
	}
	for i := 0; i < 2; i++ {
		w := request(f, "PATCH", "/admin/integrations/resend", body, "admin", "resend-first")
		if w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" || data(t, w)["version"] != float64(1) || strings.Contains(w.Body.String(), secret) {
			t.Fatalf("save/replay failed: %d %s", w.Code, w.Body)
		}
	}
	if w := request(f, "PATCH", "/admin/integrations/resend", strings.Replace(body, secret, "re_OtherPrivateCredential", 1), "admin", "resend-first"); w.Code != 409 {
		t.Fatalf("changed credential reused idempotency key: %s", w.Body)
	}
	if w := request(f, "PATCH", "/admin/integrations/resend", body, "admin", "stale-version"); w.Code != 409 {
		t.Fatalf("stale optimistic version accepted: %s", w.Body)
	}
	for _, path := range []string{"/admin/integrations", "/admin/audit", "/site/settings"} {
		w := request(f, "GET", path, "", "admin", "")
		if w.Code != 200 || strings.Contains(w.Body.String(), secret) {
			t.Fatalf("read failed or exposed secret: %s", path)
		}
	}
	var audits []AuditEvent
	if err := f.db.Find(&audits).Error; err != nil || len(audits) != 1 {
		t.Fatalf("audit count=%d error=%v", len(audits), err)
	}
	raw, _ := json.Marshal(audits)
	if strings.Contains(string(raw), secret) || audits[0].Action != "integration.update" || audits[0].TargetID != "resend" {
		t.Fatal("audit contains credential or wrong action")
	}
	var row serviceconfig.Record
	if err := f.db.Where("provider = ?", "resend").Take(&row).Error; err != nil || !strings.Contains(row.Payload, secret) {
		t.Fatal("private database did not retain plaintext credential")
	}
}

func TestIntegrationReplayReportsCurrentRunningState(t *testing.T) {
	f, env := integrationFixture(t)
	body := `{"version":0,"source":"database","enabled":true,"fields":{"sender_email":"hello@example.com"},"secrets":{"api_key":"re_StoredCredential"},"reason":"Configure mail"}`
	w := request(f, "PATCH", "/admin/integrations/resend", body, "admin", "first")
	if w.Code != 200 || data(t, w)["restart_required"] != true || data(t, w)["active_enabled"] != false {
		t.Fatalf("initial save: %s", w.Body)
	}
	f.module.deps.ServiceSettings = serviceconfig.NewManager(f.db, env, false)
	if _, err := f.module.deps.ServiceSettings.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	w = request(f, "PATCH", "/admin/integrations/resend", body, "admin", "first")
	if w.Code != 200 || data(t, w)["restart_required"] != false || data(t, w)["active_enabled"] != true {
		t.Fatalf("replay falsely returned old active state: %s", w.Body)
	}
	next := `{"version":1,"source":"database","fields":{"sender_name":"New brand"},"reason":"Rename sender"}`
	if w := request(f, "PATCH", "/admin/integrations/resend", next, "admin", "second"); w.Code != 200 {
		t.Fatal(w.Body)
	}
	w = request(f, "PATCH", "/admin/integrations/resend", body, "admin", "first")
	if w.Code != 200 || data(t, w)["version"] != float64(2) || data(t, w)["fields"].(map[string]any)["sender_name"] != "New brand" {
		t.Fatalf("old retry replaced newer form state: %s", w.Body)
	}
	var count int64
	f.db.Model(&AuditEvent{}).Count(&count)
	if count != 2 {
		t.Fatal("retry created duplicate audit")
	}
}

func TestIntegrationFailedSQLIsGenericQuietAndAtomic(t *testing.T) {
	f, env := integrationFixture(t)
	const private = "re_DoNotExposeDatabaseFailure"
	if err := f.db.Exec(`CREATE TRIGGER reject_service_setting BEFORE INSERT ON service_provider_settings BEGIN SELECT RAISE(FAIL, '` + private + `'); END`).Error; err != nil {
		t.Fatal(err)
	}
	var logs bytes.Buffer
	loud := f.db.Session(&gorm.Session{Logger: logger.New(log.New(&logs, "", 0), logger.Config{LogLevel: logger.Info})})
	f.module.deps.DB = loud
	f.module.deps.ServiceSettings = serviceconfig.NewManager(loud, env, false)
	body := `{"version":0,"source":"database","enabled":true,"fields":{"sender_email":"hello@example.com"},"secrets":{"api_key":"` + private + `"},"reason":"Configure email"}`
	w := request(f, "PATCH", "/admin/integrations/resend", body, "admin", "fail")
	if w.Code != 503 || strings.Contains(w.Body.String(), private) || strings.Contains(logs.String(), private) {
		t.Fatalf("failure was not safely hidden: code=%d", w.Code)
	}
	var count int64
	f.db.Model(&AuditEvent{}).Count(&count)
	if count != 0 {
		t.Fatal("failed provider update left a successful audit")
	}
}

func TestIntegrationStrictDecodeAndValidationRollback(t *testing.T) {
	f, _ := integrationFixture(t)
	for _, body := range []string{
		`{"version":0,"source":"database","reason":"Configure","password":"secret"}`,
		`{"version":0,"source":"database","reason":"Configure","fields":{"user_jwt_secret":"secret"}}`,
		`{"version":0,"source":"database","reason":"Configure","enabled":true}`,
		`{"version":0,"source":"environment","reason":"Configure","secrets":{"api_key":"re_Unexpected"}}`,
		`{"version":0,"source":"database","reason":"Configure"} {}`,
	} {
		if w := request(f, "PATCH", "/admin/integrations/resend", body, "admin", "bad"); w.Code != 400 {
			t.Fatalf("invalid configuration accepted: %s", w.Body)
		}
	}
	if w := request(f, "PATCH", "/admin/integrations/unknown", `{"source":"database","reason":"Configure"}`, "admin", "unknown"); w.Code != 404 {
		t.Fatalf("unknown provider accepted: %s", w.Body)
	}
	var count int64
	f.db.Model(&AuditEvent{}).Count(&count)
	if count != 0 {
		t.Fatal("invalid writes created audit rows")
	}
}
