package operations

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/brizenchi/quickstart-template/internal/credit"
	"gorm.io/gorm"
	"strings"
	"testing"

	billingdomain "github.com/brizenchi/go-modules/modules/billing/domain"
	"github.com/brizenchi/quickstart-template/internal/audit"
	"github.com/brizenchi/quickstart-template/internal/feature/content"
)

func TestContentDraftPublicationLocalesAndAudit(t *testing.T) {
	f := newFixture(t)
	body := `{"path":"/blog/new-guide","kind":"blog","position":1,"version":0,"translations":{"en":{"title":"New guide","summary":"A useful guide","body":"## Start\n\nSafe content","seo_title":"","seo_description":""}},"reason":"Write article"}`
	if w := request(f, "POST", "/admin/articles", body, "user", "content-create"); w.Code != 403 {
		t.Fatal("nonadmin content write accepted")
	}
	w := request(f, "POST", "/admin/articles", body, "admin", "content-create")
	if w.Code != 200 {
		t.Fatalf("save: %s", w.Body)
	}
	id := data(t, w)["id"].(string)
	w = request(f, "POST", "/admin/articles", body, "admin", "content-create")
	if w.Code != 200 || data(t, w)["id"] != id {
		t.Fatalf("retry: %s", w.Body)
	}
	resolve := func(locale string) any {
		t.Helper()
		w := request(f, "GET", "/site/articles/resolve?path=/blog/new-guide&locale="+locale, "", "", "")
		if w.Code != 200 {
			t.Fatalf("resolve: %s", w.Body)
		}
		return data(t, w)["article"]
	}
	if resolve("en") != nil {
		t.Fatal("draft was public")
	}
	publish := `{"version":1,"locale":"en","action":"publish","reason":"Publish English"}`
	w = request(f, "POST", "/admin/articles/"+id+"/publications", publish, "admin", "publish-en")
	if w.Code != 200 {
		t.Fatalf("publish: %s", w.Body)
	}
	if resolve("en") == nil || resolve("zh") != nil {
		t.Fatal("publication leaked a missing translation")
	}
	body = strings.Replace(body, `"version":0`, `"version":2`, 1)
	body = strings.Replace(body, "Safe content", "Unpublished edit", 1)
	w = request(f, "PUT", "/admin/articles/"+id, body, "admin", "edit-draft")
	if w.Code != 200 {
		t.Fatalf("draft edit: %s", w.Body)
	}
	raw, _ := json.Marshal(resolve("en"))
	if strings.Contains(string(raw), "Unpublished edit") {
		t.Fatal("draft overwrote live snapshot")
	}
	w = request(f, "POST", "/admin/articles/"+id+"/publications", `{"version":3,"locale":"en","action":"unpublish","reason":"Unpublish English"}`, "admin", "unpublish-en")
	if w.Code != 200 || resolve("en") != nil {
		t.Fatalf("unpublish: %s", w.Body)
	}
	w = request(f, "PUT", "/admin/articles/"+id, body, "admin", "stale-edit")
	if w.Code != 409 {
		t.Fatalf("stale version: %s", w.Body)
	}
	var events int64
	f.db.Model(&audit.Event{}).Where("outcome = ?", "succeeded").Count(&events)
	if events != 4 {
		t.Fatalf("audit events: %d", events)
	}
}
func TestSeedUnpublicationIsAuthoritativeAndInvalidPathsAreRejected(t *testing.T) {
	f := newFixture(t)
	seed := content.Seed()[0]
	w := request(f, "POST", "/admin/articles/"+seed.ID+"/publications", `{"version":0,"locale":"en","action":"unpublish","reason":"Remove starter article"}`, "admin", "unpublish-seed")
	if w.Code != 200 {
		t.Fatalf("seed: %s", w.Body)
	}
	w = request(f, "GET", "/site/articles/resolve?locale=en&path="+seed.Path, "", "", "")
	if data(t, w)["article"] != nil {
		t.Fatal("unpublished seed reappeared")
	}
	w = request(f, "GET", "/site/articles/resolve?locale=zh&path="+seed.Path, "", "", "")
	if data(t, w)["article"] == nil {
		t.Fatal("unrelated locale unpublished")
	}
	for i, path := range []string{"//evil.test/a", "/admin/users", "/blog/../../admin", "/blog/UPPER"} {
		body := fmt.Sprintf(`{"path":%q,"kind":"blog","version":0,"translations":{},"reason":"Invalid path"}`, path)
		if w := request(f, "POST", "/admin/articles", body, "admin", fmt.Sprint("invalid", i)); w.Code != 400 {
			t.Fatalf("invalid path accepted: %s", w.Body)
		}
	}
}

func TestCreditServiceAuditIsAtomicAndFailedAdminRequestsAreVisible(t *testing.T) {
	f := newFixture(t)
	if _, err := f.module.deps.Credits.GrantCreditsWithExpiry(t.Context(), credit.Grant{UserID: "owner", Amount: 20, ActorID: "owner", Reason: "Support grant", Source: "admin_grant", SourceID: "atomic-grant"}); err != nil {
		t.Fatal(err)
	}
	var events int64
	f.db.Model(&audit.Event{}).Where("action = ?", "credits.grant").Count(&events)
	if events != 1 {
		t.Fatalf("missing grant audit: %d", events)
	}
	if err := f.db.Callback().Create().Before("gorm:create").Register("test:fail-audit", func(tx *gorm.DB) {
		if tx.Statement.Table == "audit_outcome_events" {
			tx.AddError(errors.New("audit store failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	_, err := f.module.deps.Credits.GrantCreditsWithExpiry(t.Context(), credit.Grant{UserID: "owner", Amount: 10, ActorID: "owner", Reason: "Support grant", Source: "admin_grant", SourceID: "failed-grant"})
	if err == nil {
		t.Fatal("audit failure did not roll back grant")
	}
	if err := f.db.Callback().Create().Remove("test:fail-audit"); err != nil {
		t.Fatal(err)
	}
	balance, err := f.module.deps.Credits.GetCreditSummary(t.Context(), "owner")
	if err != nil || balance.Balance != 20 {
		t.Fatalf("balance after failed audit: %+v %v", balance, err)
	}
	w := request(f, "PATCH", "/admin/settings", `{"brand_name":"","reason":"sk_test_not-a-real-key"}`, "admin", "invalid-secret-reason")
	if w.Code != 400 {
		t.Fatalf("invalid settings: %s", w.Body)
	}
	w = request(f, "GET", "/admin/audit?status=failed", "", "admin", "")
	if w.Code != 200 || data(t, w)["total"] != float64(1) || strings.Contains(w.Body.String(), "not-a-real") {
		t.Fatalf("failure audit: %s", w.Body)
	}
}

func TestSettingsAuditKeepsBeforeAfterAndReplaysOriginalResult(t *testing.T) {
	f := newFixture(t)
	body := `{"brand_name":"Updated product","headline_zh":"新的介绍","reason":"Update website identity"}`
	w := request(f, "PATCH", "/admin/settings", body, "admin", "website-change")
	if w.Code != 200 {
		t.Fatalf("save: %s", w.Body)
	}
	var event audit.Event
	if err := f.db.Where("action = ? AND outcome = ?", "settings.update", "succeeded").Take(&event).Error; err != nil {
		t.Fatal(err)
	}
	var changes struct {
		Before SiteSettings `json:"before"`
		After  SiteSettings `json:"after"`
	}
	if err := json.Unmarshal([]byte(event.Details), &changes); err != nil || changes.Before.BrandName != DefaultSettings().BrandName || changes.After.BrandName != "Updated product" {
		t.Fatalf("missing settings changes: %s", event.Details)
	}
	w = request(f, "PATCH", "/admin/settings", `{"brand_name":"Later product","reason":"Another identity change"}`, "admin", "later-change")
	if w.Code != 200 {
		t.Fatalf("later change: %s", w.Body)
	}
	w = request(f, "PATCH", "/admin/settings", body, "admin", "website-change")
	if w.Code != 200 || data(t, w)["brand_name"] != "Updated product" {
		t.Fatalf("retry lost original result: %s", w.Body)
	}
}

func TestAdminPaymentsUseExactCustomerEnvironmentAndDateFilters(t *testing.T) {
	f := newFixture(t)
	for i, owner := range []string{"owner", "other"} {
		row := billingdomain.BillingEvent{UserID: owner, Provider: "stripe", ProviderEventID: fmt.Sprint("evt_filter_", i), EventType: "checkout.session.completed", Payload: json.RawMessage(fmt.Sprintf(`{"livemode":false,"data":{"object":{"id":"cs_filter_%d","created":1767225600,"amount_total":500,"currency":"usd","payment_status":"paid"}}}`, i))}
		if err := f.db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	base := "/admin/orders?user_id=owner&mode=test&status=paid"
	w := request(f, "GET", base+"&from=2026-01-01T00:00:00Z&to=2026-01-02T00:00:00Z", "", "admin", "")
	if w.Code != 200 || data(t, w)["total"] != float64(1) || strings.Contains(w.Body.String(), "cs_filter_1") {
		t.Fatalf("exact filters: %s", w.Body)
	}
	for _, query := range []string{"user_id=owner&mode=live", "user_id=missing", "to=2026-01-01T00:00:00Z"} {
		w = request(f, "GET", "/admin/orders?"+query, "", "admin", "")
		if w.Code != 200 || data(t, w)["total"] != float64(0) {
			t.Fatalf("unexpected payment: %s", w.Body)
		}
	}
	for _, query := range []string{"mode=invalid", "from=invalid", "status=refunded", "from=2026-02-01T00:00:00Z&to=2026-01-01T00:00:00Z"} {
		if w := request(f, "GET", "/admin/orders?"+query, "", "admin", ""); w.Code != 400 {
			t.Fatalf("invalid filter accepted: %s", w.Body)
		}
	}
}
