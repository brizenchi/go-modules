package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	authdomain "github.com/brizenchi/go-modules/modules/auth/domain"
	"github.com/brizenchi/quickstart-template/internal/feature/credits"
	"github.com/brizenchi/quickstart-template/internal/feature/note"
	"github.com/brizenchi/quickstart-template/internal/feature/operations"
	"github.com/brizenchi/quickstart-template/internal/hostapi"
	apphttp "github.com/brizenchi/quickstart-template/internal/http"
	"github.com/brizenchi/quickstart-template/internal/platform"
	"github.com/brizenchi/quickstart-template/internal/user"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestOperationsWithRealJWTAndCreditLifecycle(t *testing.T) {
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
	cfg := platform.Config{ServiceName: "operations-integration", Email: platform.EmailConfig{Provider: "log"}, Auth: platform.AuthConfig{UserJWTSecret: "isolated-operations-test-signing-secret", Google: platform.GoogleConfig{Enabled: boolPtr(false)}, GitHub: platform.GitHubConfig{Enabled: boolPtr(false)}}, Billing: platform.BillingConfig{Enabled: boolPtr(false)}, Referral: platform.ReferralConfig{Enabled: boolPtr(false)}}
	if err := platform.Migrate(db, cfg); err != nil {
		t.Fatal(err)
	}
	models := []any{&note.Note{}}
	models = append(models, credits.Models()...)
	models = append(models, operations.Models()...)
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatal(err)
	}
	modules, err := platform.New(db, cfg)
	if err != nil {
		t.Fatal(err)
	}
	engine := BuildRouter(RouterConfig{ServiceName: "operations-integration"}, apphttp.NewRouter(modules, hostapi.Deps{DB: db, Modules: modules, Users: modules.Users}))
	account := &user.User{Email: "member@example.com", EmailVerified: true, Credits: 10}
	admin := &user.User{Email: "operator@example.com", EmailVerified: true, Role: user.RoleAdmin}
	for _, person := range []*user.User{account, admin} {
		if err := modules.Users.Create(context.Background(), person); err != nil {
			t.Fatal(err)
		}
	}
	issue := func(person *user.User, role authdomain.Role) string {
		token, err := modules.Auth.Deps.TokenSigner.Issue(authdomain.Identity{UserID: person.ID, Email: person.Email, Role: role}, time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		return token.Value
	}
	memberToken, adminToken := issue(account, authdomain.RoleUser), issue(admin, authdomain.RoleAdmin)
	call := func(method, path, body, token, key string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "/api/v1"+path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		if key != "" {
			r.Header.Set("Idempotency-Key", key)
		}
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, r)
		return w
	}
	decode := func(w *httptest.ResponseRecorder) map[string]any {
		var envelope struct {
			Data map[string]any `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
			t.Fatal(err)
		}
		return envelope.Data
	}
	for _, route := range []struct{ method, path string }{
		{"GET", "/admin/overview"}, {"GET", "/admin/users"}, {"GET", "/admin/subscriptions"}, {"GET", "/admin/orders"}, {"GET", "/admin/referrals"}, {"GET", "/admin/audit"}, {"GET", "/admin/settings"}, {"PATCH", "/admin/settings"}, {"POST", "/admin/referrals/1/retry-reward"}, {"GET", "/admin/credits/transactions"}, {"POST", "/admin/credits/grants"}, {"POST", "/admin/credits/refunds"},
	} {
		for _, token := range []string{"", memberToken} {
			want := http.StatusUnauthorized
			if token != "" {
				want = http.StatusForbidden
			}
			w := call(route.method, route.path, `{}`, token, "")
			if w.Code != want {
				t.Fatalf("%s %s: want%d got%d %s", route.method, route.path, want, w.Code, w.Body)
			}
		}
	}
	w := call("PATCH", "/admin/settings", `{"export_credit_cost":3,"reason":"Set paid export price"}`, adminToken, "set-export-price-3")
	if w.Code != 200 || decode(w)["export_credit_cost"] != float64(3) {
		t.Fatalf("price update: %s", w.Body)
	}
	n := note.Note{UserID: account.ID, Title: "Private note", Body: "Paid snapshot"}
	if err := db.Create(&n).Error; err != nil {
		t.Fatal(err)
	}
	path := "/notes/" + strconv.FormatUint(n.ID, 10) + "/export"
	w = call("POST", path, `{"expected_cost":1,"idempotency_key":"export-request-1"}`, memberToken, "")
	if w.Code != 409 || !strings.Contains(w.Body.String(), "price_changed") {
		t.Fatalf("stale quote: %s", w.Body)
	}
	summary, err := modules.Users.GetCreditSummary(t.Context(), account.ID)
	if err != nil || summary.Balance != 10 {
		t.Fatalf("stale quote charged: %+v %v", summary, err)
	}
	w = call("POST", path, `{"expected_cost":3,"idempotency_key":"export-request-1"}`, memberToken, "")
	if w.Code != 200 || decode(w)["balance"] != float64(7) {
		t.Fatalf("export failed: %s", w.Body)
	}
	transactionID := decode(w)["transaction_id"]
	// A successful request must return its saved snapshot even if the operator
	// later raises the price. The same key never incurs a second charge.
	w = call("PATCH", "/admin/settings", `{"export_credit_cost":5,"reason":"Adjust future export price"}`, adminToken, "set-export-price-5")
	if w.Code != 200 {
		t.Fatalf("second price: %s", w.Body)
	}
	w = call("POST", path, `{"expected_cost":3,"idempotency_key":"export-request-1"}`, memberToken, "")
	if w.Code != 200 || decode(w)["transaction_id"] != transactionID || decode(w)["balance"] != float64(7) {
		t.Fatalf("repeat changed transaction: %s", w.Body)
	}
	w = call("GET", "/account/profile", "", memberToken, "")
	if w.Code != 200 || decode(w)["credits"] != float64(7) {
		t.Fatalf("profile disagrees: %s", w.Body)
	}
	w = call("GET", "/credits/transactions", "", memberToken, "")
	if w.Code != 200 {
		t.Fatalf("ledger failed: %s", w.Body)
	}
	list := decode(w)["list"].([]any)
	var total float64
	consumptions := 0
	for _, raw := range list {
		item := raw.(map[string]any)
		total += item["amount"].(float64)
		if item["kind"] == "consume" {
			consumptions++
		}
	}
	if total != 7 || consumptions != 1 {
		t.Fatalf("ledger total=%v consumes=%d", total, consumptions)
	}
}
