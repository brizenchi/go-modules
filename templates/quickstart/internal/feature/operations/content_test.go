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
func TestStarterContentRequiresExplicitImportAndPersistsPublications(t *testing.T) {
	f := newFixture(t)
	w := request(f, "GET", "/site/articles?locale=en", "", "", "")
	if w.Code != 200 || data(t, w)["total"] != float64(0) {
		t.Fatalf("virtual articles leaked: %s", w.Body)
	}
	for _, role := range []string{"", "user"} {
		if w := request(f, "POST", "/admin/articles/import", `{"apply":true,"reason":"Initialize content"}`, role, "import-denied"); w.Code != 401 && w.Code != 403 {
			t.Fatalf("import authorization: %s", w.Body)
		}
	}
	w = request(f, "POST", "/admin/articles/import", `{"apply":false}`, "admin", "")
	if w.Code != 200 {
		t.Fatalf("preview: %s", w.Body)
	}
	var count int64
	f.db.Model(&content.Record{}).Count(&count)
	if count != 0 {
		t.Fatal("preview wrote content")
	}
	w = request(f, "POST", "/admin/articles/import", `{"apply":true,"reason":"Initialize verified content"}`, "admin", "import-starter")
	if w.Code != 200 {
		t.Fatalf("import: %s", w.Body)
	}
	f.db.Model(&content.Record{}).Count(&count)
	if count != int64(len(content.Seed())) {
		t.Fatalf("not persisted: %d", count)
	}
	var importEvents int64
	f.db.Model(&audit.Event{}).Where("action IN ? AND outcome = ?", []string{"content.save", "content.publish"}, "succeeded").Count(&importEvents)
	if importEvents != count*3 {
		t.Fatalf("missing import audit: %d", importEvents)
	}
	service := content.New(f.db, nil)
	row, err := service.Resolve(t.Context(), "/", "en")
	if err != nil || row == nil || row.Version != 3 {
		t.Fatalf("home snapshot: %+v %v", row, err)
	}
	w = request(f, "POST", "/admin/articles/import", `{"apply":true,"reason":"Preserve existing content"}`, "admin", "import-again")
	if w.Code != 200 {
		t.Fatalf("repeat import: %s", w.Body)
	}
	unchanged, _ := service.Read(t.Context(), row.ID)
	if unchanged.Version != row.Version {
		t.Fatal("import replaced existing content")
	}
	w = request(f, "POST", "/admin/articles/"+row.ID+"/publications", fmt.Sprintf(`{"version":%d,"locale":"en","action":"unpublish","reason":"Remove starter page"}`, row.Version), "admin", "unpublish-seed")
	if w.Code != 200 {
		t.Fatalf("seed: %s", w.Body)
	}
	w = request(f, "GET", "/site/articles/resolve?locale=en&path="+row.Path, "", "", "")
	if data(t, w)["article"] != nil {
		t.Fatal("unpublished seed reappeared")
	}
	w = request(f, "GET", "/site/articles/resolve?locale=zh&path="+row.Path, "", "", "")
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

func TestStarterImportRollsBackWhenPublicationAuditFails(t *testing.T) {
	f := newFixture(t)
	if err := f.db.Callback().Create().Before("gorm:create").Register("test:import-audit-failure", func(tx *gorm.DB) {
		if tx.Statement.Table == "audit_outcome_events" {
			tx.AddError(errors.New("audit unavailable"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.db.Callback().Create().Remove("test:import-audit-failure") })
	_, err := content.New(f.db, nil).ImportStarter(t.Context(), "owner", "atomic-import", "Initialize content", true)
	if err == nil {
		t.Fatal("failed audit accepted")
	}
	var count int64
	f.db.Model(&content.Record{}).Count(&count)
	if count != 0 {
		t.Fatalf("partial import: %d", count)
	}
}

func TestContentPackAndPagePathBoundaries(t *testing.T) {
	paths := map[string]bool{}
	for _, row := range content.Seed() {
		if paths[row.Path] {
			t.Fatalf("duplicate pack path: %s", row.Path)
		}
		paths[row.Path] = true
		if !content.Valid(content.Save{Path: row.Path, Kind: row.Kind, Position: row.Position, Translations: row.Translations}) {
			t.Fatalf("invalid pack record: %s", row.Path)
		}
		for _, locale := range []string{"en", "zh"} {
			text := row.Translations[locale]
			if text.Title == "" || text.Summary == "" || len(text.Body) < 80 {
				t.Fatalf("incomplete %s %s", row.Path, locale)
			}
		}
		if len(row.Published) != 0 {
			t.Fatal("pack invented publication timestamps")
		}
	}
	for _, path := range []string{"/admin", "/pricing", "/api/v1", "/account", "/zh", "/privacy", "/page/arbitrary"} {
		if content.Valid(content.Save{Path: path, Kind: "page"}) {
			t.Fatalf("page shadows application: %s", path)
		}
	}
	if content.Valid(content.Save{Path: "/", Kind: "blog"}) || content.Valid(content.Save{Path: "/blog/a", Kind: "page"}) {
		t.Fatal("mismatched kind accepted")
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

func TestAdminArticlesFilterBySectionLanguageAndState(t *testing.T) {
	f := newFixture(t)
	create := func(path, kind string, position int, translations string) string {
		t.Helper()
		body := fmt.Sprintf(`{"path":%q,"kind":%q,"position":%d,"version":0,"translations":%s,"reason":"Write article"}`, path, kind, position, translations)
		w := request(f, "POST", "/admin/articles", body, "admin", "create-"+path)
		if w.Code != 200 {
			t.Fatalf("create %s: %s", path, w.Body)
		}
		return data(t, w)["id"].(string)
	}
	publish := func(id string, version int, locale string) {
		t.Helper()
		body := fmt.Sprintf(`{"version":%d,"locale":%q,"action":"publish","reason":"Publish article"}`, version, locale)
		if w := request(f, "POST", "/admin/articles/"+id+"/publications", body, "admin", "publish-"+id+locale); w.Code != 200 {
			t.Fatalf("publish: %s", w.Body)
		}
	}
	en := `{"title":"Alpha","summary":"S","body":"Body","seo_title":"","seo_description":""}`
	ja := `{"title":"アルファ","summary":"S","body":"本文","seo_title":"","seo_description":""}`
	// Alpha: English live with an unpublished edit, Japanese draft.
	alpha := create("/features/alpha", "features", 1, `{"en":`+en+`,"ja":`+ja+`}`)
	publish(alpha, 1, "en")
	edit := fmt.Sprintf(`{"path":"/features/alpha","kind":"features","position":1,"version":2,"translations":{"en":%s,"ja":%s},"reason":"Edit draft"}`, strings.Replace(en, `"Body"`, `"Edited"`, 1), ja)
	if w := request(f, "PUT", "/admin/articles/"+alpha, edit, "admin", "edit-alpha"); w.Code != 200 {
		t.Fatalf("edit: %s", w.Body)
	}
	// Beta: English draft only; an empty Japanese entry does not count as a version.
	create("/blog/beta", "blog", 2, `{"en":`+en+`,"ja":{"title":" ","summary":"","body":"","seo_title":"","seo_description":""}}`)
	// Gamma: English live and unchanged.
	publish(create("/docs/gamma", "docs", 3, `{"en":`+en+`}`), 1, "en")

	list := func(query string) []string {
		t.Helper()
		w := request(f, "GET", "/admin/articles?"+query, "", "admin", "")
		if w.Code != 200 {
			t.Fatalf("list %s: %s", query, w.Body)
		}
		paths := []string{}
		for _, item := range data(t, w)["items"].([]any) {
			paths = append(paths, item.(map[string]any)["path"].(string))
		}
		if total := int(data(t, w)["total"].(float64)); total != len(paths) {
			t.Fatalf("list %s: total %d for %d items", query, total, len(paths))
		}
		return paths
	}
	for query, want := range map[string]string{
		"kind=features":                      "/features/alpha",
		"kind=blog":                          "/blog/beta",
		"status=published":                   "/features/alpha /docs/gamma",
		"status=changed":                     "/features/alpha",
		"status=draft":                       "/features/alpha /blog/beta",
		"locale=ja":                          "/features/alpha",
		"locale=ja&status=draft":             "/features/alpha",
		"locale=ja&status=missing":           "/blog/beta /docs/gamma",
		"locale=en&status=draft":             "/blog/beta",
		"locale=en&status=published":         "/features/alpha /docs/gamma",
		"kind=docs&locale=en&status=changed": "",
		"query=gamma&status=published":       "/docs/gamma",
	} {
		if got := strings.Join(list(query), " "); got != want {
			t.Errorf("%s: got %q, want %q", query, got, want)
		}
	}
	for _, query := range []string{"kind=news", "locale=it", "status=archived", "status=missing"} {
		if w := request(f, "GET", "/admin/articles?"+query, "", "admin", ""); w.Code != 400 {
			t.Errorf("%s accepted: %d", query, w.Code)
		}
	}
}
