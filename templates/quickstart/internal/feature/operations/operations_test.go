package operations

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"mime/multipart"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	authdomain "github.com/brizenchi/go-modules/modules/auth/domain"
	authhttp "github.com/brizenchi/go-modules/modules/auth/http"
	billingdomain "github.com/brizenchi/go-modules/modules/billing/domain"
	"github.com/brizenchi/go-modules/modules/referral"
	refbus "github.com/brizenchi/go-modules/modules/referral/adapter/eventbus"
	refrepo "github.com/brizenchi/go-modules/modules/referral/adapter/gormrepo"
	refdomain "github.com/brizenchi/go-modules/modules/referral/domain"
	refevent "github.com/brizenchi/go-modules/modules/referral/event"
	"github.com/brizenchi/quickstart-template/internal/hostapi"
	"github.com/brizenchi/quickstart-template/internal/hostcfg"
	"github.com/brizenchi/quickstart-template/internal/platform"
	"github.com/brizenchi/quickstart-template/internal/user"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type fixture struct {
	router *gin.Engine
	db     *gorm.DB
	module *Module
	owner  user.User
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	gin.SetMode(gin.TestMode)
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
	if err := user.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	models := append(Models(), refrepo.AutoMigrateModels()...)
	models = append(models, &billingdomain.BillingEvent{}, &billingdomain.BillingSubscription{})
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatal(err)
	}
	owner := user.User{ID: "owner", Email: "owner@example.com"}
	if err := user.NewRepository(db).Create(context.Background(), &owner); err != nil {
		t.Fatal(err)
	}
	m := New(hostapi.Deps{DB: db, Users: user.NewRepository(db), Config: hostcfg.Config{Uploads: hostcfg.UploadConfig{Enabled: true, Directory: t.TempDir()}}})
	r := gin.New()
	public := r.Group("/api/v1")
	authed := r.Group("/api/v1")
	authed.Use(func(c *gin.Context) {
		role := c.GetHeader("X-Test-Role")
		if role != "" {
			id := c.GetHeader("X-Test-User")
			if id == "" {
				id = "owner"
			}
			authhttp.SetIdentity(c, &authdomain.Identity{UserID: id, Role: authdomain.Role(role)})
		}
		c.Next()
	})
	m.Register(hostapi.Groups{Public: public, User: authed, Admin: authed})
	return fixture{r, db, m, owner}
}
func request(f fixture, method, path, body, role, key string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "/api/v1"+path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Test-Role", role)
	if key != "" {
		r.Header.Set("Idempotency-Key", key)
	}
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, r)
	return w
}
func data(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var envelope struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	return envelope.Data
}

func TestAdminEndpointsRejectUsersBeforeAnyWork(t *testing.T) {
	f := newFixture(t)
	for _, path := range []string{"/admin/overview", "/admin/users", "/admin/subscriptions", "/admin/orders", "/admin/referrals", "/admin/audit", "/admin/settings"} {
		for _, role := range []string{"user", ""} {
			want := 403
			if role == "" {
				want = 401
			}
			w := request(f, "GET", path, "", role, "")
			if w.Code != want {
				t.Fatalf("%s role=%q got%d: %s", path, role, w.Code, w.Body)
			}
		}
	}
	for _, tc := range []struct{ method, path string }{{"PATCH", "/admin/settings"}, {"POST", "/admin/referrals/1/retry-reward"}} {
		w := request(f, tc.method, tc.path, `{}`, "user", "")
		if w.Code != 403 {
			t.Fatalf("write got%d", w.Code)
		}
	}
	if w := request(f, "GET", "/site/settings", "", "", ""); w.Code != 200 || data(t, w)["export_credit_cost"] != float64(1) {
		t.Fatalf("public defaults: %s", w.Body)
	}
}

func TestSettingsValidateAllowlistAndAuditIdempotently(t *testing.T) {
	f := newFixture(t)
	for _, body := range []string{`{"stripe_secret_key":"secret","reason":"setup"}`, `{"support_url":"javascript:alert(1)","reason":"setup"}`, `{"support_email":"Name <owner@example.com>","reason":"setup"}`, `{"export_credit_cost":0,"reason":"setup"}`, `{"brand_name":"","reason":"setup"}`, `{"brand_name":"Valid","reason":"x"}`, `{"reason":"setup"}`} {
		if w := request(f, "PATCH", "/admin/settings", body, "admin", "invalid"); w.Code != 400 {
			t.Fatalf("expected400: %s", w.Body)
		}
	}
	body := `{"brand_name":"Launch Kit","support_email":"help@example.com","export_credit_cost":3,"reason":"Brand launch"}`
	if w := request(f, "PATCH", "/admin/settings", body, "admin", ""); w.Code != 400 {
		t.Fatal("missing key accepted")
	}
	for i := 0; i < 2; i++ {
		w := request(f, "PATCH", "/admin/settings", body, "admin", "settings-1")
		if w.Code != 200 || data(t, w)["brand_name"] != "Launch Kit" {
			t.Fatalf("settings update: %s", w.Body)
		}
	}
	w := request(f, "PATCH", "/admin/settings", strings.Replace(body, "Launch Kit", "Changed", 1), "admin", "settings-1")
	if w.Code != 409 {
		t.Fatalf("conflicting retry: %s", w.Body)
	}
	var audits []AuditEvent
	if err := f.db.Find(&audits).Error; err != nil {
		t.Fatal(err)
	}
	if len(audits) != 1 || audits[0].ActorID != "owner" || audits[0].Reason != "Brand launch" || audits[0].Status != "succeeded" {
		t.Fatalf("audits: %+v", audits)
	}
	w = request(f, "GET", "/site/settings", "", "", "")
	if w.Code != 200 || data(t, w)["export_credit_cost"] != float64(3) || strings.Contains(w.Body.String(), "idempotency") {
		t.Fatalf("public settings: %s", w.Body)
	}
}

func TestListsSearchPaginateAndDoNotLeakRawBillingPayloads(t *testing.T) {
	f := newFixture(t)
	for _, email := range []string{"Alice@example.com", "alice.two@example.com", "bob@example.com"} {
		if err := f.db.Create(&user.User{Email: email}).Error; err != nil {
			t.Fatal(err)
		}
	}
	w := request(f, "GET", "/admin/users?query=ALICE&page=2&limit=1", "", "admin", "")
	if w.Code != 200 || data(t, w)["total"] != float64(2) || len(data(t, w)["items"].([]any)) != 1 {
		t.Fatalf("search: %s", w.Body)
	}
	w = request(f, "GET", "/admin/users?query=%25", "", "admin", "")
	if w.Code != 200 || data(t, w)["total"] != float64(0) {
		t.Fatalf("wildcard search: %s", w.Body)
	}
	for _, query := range []string{"page=0", "limit=101", "page=999999999999999999999", "query=" + strings.Repeat("a", 201)} {
		if w := request(f, "GET", "/admin/users?"+query, "", "admin", ""); w.Code != 400 {
			t.Fatalf("bad pagination accepted: %s", query)
		}
	}
	sub := billingdomain.BillingSubscription{UserID: "owner", Provider: "stripe", Plan: "pro", Status: "active", RawSnapshotJSON: json.RawMessage(`{"secret":"private"}`)}
	if err := f.db.Create(&sub).Error; err != nil {
		t.Fatal(err)
	}
	w = request(f, "GET", "/admin/subscriptions?query=owner@example.com", "", "admin", "")
	if w.Code != 200 || data(t, w)["total"] != float64(1) || strings.Contains(w.Body.String(), "private") {
		t.Fatalf("subscription: %s", w.Body)
	}
}

func TestOrdersDeduplicateInvoiceAndCheckoutAndKeepMissingAmountsNull(t *testing.T) {
	f := newFixture(t)
	events := []billingdomain.BillingEvent{
		{ProviderEventID: "evt_checkout", EventType: "checkout.session.completed", Payload: json.RawMessage(`{"livemode":false,"data":{"object":{"id":"cs_1","invoice":"in_1","amount_total":900,"currency":"usd","payment_status":"paid"}}}`)},
		{ProviderEventID: "evt_invoice", EventType: "invoice.paid", Payload: json.RawMessage(`{"livemode":false,"data":{"object":{"id":"in_1","amount_paid":800,"currency":"usd","private":"secret"}}}`)},
		{ProviderEventID: "evt_duplicate", EventType: "invoice.payment_succeeded", Payload: json.RawMessage(`{"livemode":false,"data":{"object":{"id":"in_1","amount_paid":800,"currency":"usd"}}}`)},
		{ProviderEventID: "evt_unknown", EventType: "checkout.session.completed", Payload: json.RawMessage(`{"data":{"object":{"id":"cs_unknown"}}}`)},
		{ProviderEventID: "evt_ignored", EventType: "customer.subscription.updated", Payload: json.RawMessage(`{"data":{"object":{"id":"sub_1"}}}`)},
		{ProviderEventID: "evt_legacy", EventType: "invoice.paid", Payload: json.RawMessage(`{}`)},
	}
	for _, event := range events {
		event.Provider = "stripe"
		event.UserID = "owner"
		if err := f.db.Create(&event).Error; err != nil {
			t.Fatal(err)
		}
	}
	w := request(f, "GET", "/admin/orders", "", "admin", "")
	if w.Code != 200 || data(t, w)["total"] != float64(3) || strings.Contains(w.Body.String(), "secret") {
		t.Fatalf("orders: %s", w.Body)
	}
	for _, raw := range data(t, w)["items"].([]any) {
		item := raw.(map[string]any)
		if item["id"] == "stripe:in_1" && item["amount"] != float64(800) {
			t.Fatalf("invoice amount: %+v", item)
		}
		if item["id"] == "stripe:cs_unknown" && item["amount"] != nil {
			t.Fatalf("unknown amount must null: %+v", item)
		}
		if item["id"] == "stripe:evt_legacy" && (item["amount"] != nil || item["record_type"] != "payment_event") {
			t.Fatalf("legacy event: %+v", item)
		}
	}
}

func TestReferralReconciliationCannotActivatePendingAndRewardsStayIdempotent(t *testing.T) {
	f := newFixture(t)
	bus := refbus.NewInProc()
	f.module.deps.Modules = &platform.Modules{Referral: &referral.Module{Deps: referral.Deps{Bus: bus}}}
	repo := refrepo.NewReferralRepo(f.db)
	ref, err := repo.Create(context.Background(), refdomain.Referral{Code: "INV_CODE", ReferrerID: "owner", RefereeID: "new-user", Status: refdomain.StatusPending})
	if err != nil {
		t.Fatal(err)
	}
	path := "/admin/referrals/" + strconv.FormatUint(ref.ID, 10) + "/retry-reward"
	if w := request(f, "POST", path, `{"reason":"Repair delivery"}`, "admin", "repair-1"); w.Code != 409 {
		t.Fatalf("pending activated: %s", w.Body)
	}
	if _, err := repo.Activate(context.Background(), "new-user", 25); err != nil {
		t.Fatal(err)
	}
	calls := 0
	bus.Subscribe(refevent.KindReferralActivated, func(ctx context.Context, e refevent.Envelope) error {
		calls++
		if calls == 1 {
			return errors.New("temporary delivery failure")
		}
		r := e.Payload.(refevent.ReferralActivated).Referral
		return f.module.deps.Users.GrantCredits(ctx, r.ReferrerID, "referral", strconv.FormatUint(r.ID, 10), int64(r.RewardCredits))
	})
	if w := request(f, "POST", path, `{"reason":"Repair delivery"}`, "admin", "repair-1"); w.Code != 502 {
		t.Fatalf("expected retryable error: %s", w.Body)
	}
	for _, key := range []string{"repair-1", "repair-1", "repair-2"} {
		if w := request(f, "POST", path, `{"reason":"Repair delivery"}`, "admin", key); w.Code != 200 {
			t.Fatalf("replay: %s", w.Body)
		}
	}
	var owner user.User
	if err := f.db.Where("id = ?", "owner").Take(&owner).Error; err != nil {
		t.Fatal(err)
	}
	if owner.Credits != 25 || calls != 3 {
		t.Fatalf("credits=%d calls=%d", owner.Credits, calls)
	}
	var audits []AuditEvent
	f.db.Find(&audits)
	if len(audits) != 2 || audits[0].Status != "succeeded" {
		t.Fatalf("audits %+v", audits)
	}
}

func TestReferralListsComputeExpiryWithoutChangingQualification(t *testing.T) {
	f := newFixture(t)
	past := time.Now().UTC().Add(-time.Hour)
	repo := refrepo.NewReferralRepo(f.db)
	_, err := repo.Create(context.Background(), refdomain.Referral{Code: "INV_CODE", ReferrerID: "owner", RefereeID: "new-user", Status: refdomain.StatusPending, ExpiresAt: &past})
	if err != nil {
		t.Fatal(err)
	}
	w := request(f, "GET", "/admin/referrals?status=expired&query=owner", "", "admin", "")
	if w.Code != 200 || data(t, w)["total"] != float64(1) {
		t.Fatalf("expired query: %s", w.Body)
	}
	w = request(f, "GET", "/admin/overview", "", "admin", "")
	if w.Code != 200 || data(t, w)["pending_referrals"] != float64(0) {
		t.Fatalf("counts: %s", w.Body)
	}
}

func uploadRequest(t *testing.T, f fixture, body []byte, role, owner string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "../../untrusted.svg")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("POST", "/api/v1/uploads/images", &buf)
	r.Header.Set("Content-Type", writer.FormDataContentType())
	r.Header.Set("X-Test-Role", role)
	r.Header.Set("X-Test-User", owner)
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, r)
	return w
}
func TestImageUploadSniffsContentAndEnforcesPrivateOwnership(t *testing.T) {
	f := newFixture(t)
	var pngBytes bytes.Buffer
	if err := png.Encode(&pngBytes, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	if w := uploadRequest(t, f, pngBytes.Bytes(), "", ""); w.Code != 401 {
		t.Fatal("anonymous upload accepted")
	}
	for _, body := range [][]byte{[]byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`), []byte("not an image"), bytes.Repeat([]byte("a"), int(MaxImageBytes+1))} {
		if w := uploadRequest(t, f, body, "user", "owner"); w.Code != 400 {
			t.Fatalf("invalid upload accepted: %d", w.Code)
		}
	}
	w := uploadRequest(t, f, pngBytes.Bytes(), "user", "owner")
	if w.Code != 200 {
		t.Fatalf("upload: %s", w.Body)
	}
	info := data(t, w)
	if info["content_type"] != "image/png" || !strings.HasSuffix(info["filename"].(string), ".png") || strings.Contains(info["filename"].(string), "untrusted") {
		t.Fatalf("metadata: %+v", info)
	}
	for _, owner := range []string{"owner", "other"} {
		r := httptest.NewRequest("GET", info["url"].(string), nil)
		r.Header.Set("X-Test-Role", "user")
		r.Header.Set("X-Test-User", owner)
		got := httptest.NewRecorder()
		f.router.ServeHTTP(got, r)
		if owner == "other" {
			if got.Code != 404 {
				t.Fatalf("private image leaked: %d", got.Code)
			}
		} else if got.Code != 200 || !bytes.Equal(got.Body.Bytes(), pngBytes.Bytes()) || got.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatalf("image read: %d %s", got.Code, got.Body)
		}
		listReq := httptest.NewRequest("GET", "/api/v1/uploads/images", nil)
		listReq.Header.Set("X-Test-Role", "user")
		listReq.Header.Set("X-Test-User", owner)
		listResponse := httptest.NewRecorder()
		f.router.ServeHTTP(listResponse, listReq)
		want := float64(1)
		if owner == "other" {
			want = 0
		}
		if listResponse.Code != 200 || data(t, listResponse)["total"] != want {
			t.Fatalf("private image list for %s: %s", owner, listResponse.Body)
		}
	}
}
