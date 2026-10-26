package operations

import (
	"strconv"
	"strings"
	"testing"

	authdomain "github.com/brizenchi/go-modules/modules/auth/domain"
	authhttp "github.com/brizenchi/go-modules/modules/auth/http"
	"github.com/brizenchi/quickstart-template/internal/hostapi"
	"github.com/brizenchi/quickstart-template/internal/hostcfg"
	"github.com/brizenchi/quickstart-template/internal/user"
	"github.com/gin-gonic/gin"
)

func setTestIdentity(c *gin.Context, id, role string) {
	authhttp.SetIdentity(c, &authdomain.Identity{UserID: id, Role: authdomain.Role(role)})
}

func enabledTranslationConfig() hostcfg.ContentTranslationConfig {
	return hostcfg.ContentTranslationConfig{
		Enabled:               true,
		BaseURL:               "https://api.aigocode.dev",
		APIKey:                "aistudio_test_not-a-real-key",
		Model:                 "gemini-3.8-flash",
		MaxOutputTokens:       16384,
		MaxZhCharacters:       5000,
		MaxEnWords:            3000,
		RequestTimeoutSeconds: 90,
		TaskTimeoutSeconds:    420,
	}
}

func newTranslationFixture(t *testing.T, configure func(*hostcfg.ContentTranslationConfig)) fixture {
	t.Helper()
	f := newFixture(t)
	translation := enabledTranslationConfig()
	if configure != nil {
		configure(&translation)
	}
	f.module = New(hostapi.Deps{DB: f.db, Users: user.NewRepository(f.db), Config: hostcfg.Config{
		ContentTranslation: translation,
		Uploads:            hostcfg.UploadConfig{Enabled: true, Directory: t.TempDir()},
	}})
	gin.SetMode(gin.TestMode)
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
			setTestIdentity(c, id, role)
		}
		c.Next()
	})
	f.module.Register(hostapi.Groups{Public: public, User: authed, Admin: authed})
	f.router = r
	return f
}

func seedTranslationArticle(t *testing.T, f fixture, name, sourceLocale string) (string, int64) {
	t.Helper()
	body := `{"path":"/blog/translation-api-` + name + `","kind":"blog","position":1,"version":0,"translations":{"zh":{"title":"快速开始","summary":"五分钟","body":"# 快速开始\n\n安装 SDK。","seo_title":"","seo_description":""}},"reason":"Seed article"}`
	if sourceLocale != "" {
		body = strings.Replace(body, `"reason":`, `"`+"source_locale"+`":"`+sourceLocale+`","reason":`, 1)
	}
	w := request(f, "POST", "/admin/articles", body, "admin", "seed-"+t.Name()+"-"+name)
	if w.Code != 200 {
		t.Fatalf("seed article: %s", w.Body)
	}
	id, _ := data(t, w)["id"].(string)
	version, _ := data(t, w)["version"].(float64)
	return id, int64(version)
}

const translationCreateBody = `{"version":%d,"source_locale":"zh","target_locale":"en","reason":"AI 翻译：zh → en"}`

func TestTranslationEndpointsRequireAdmin(t *testing.T) {
	f := newTranslationFixture(t, nil)
	id, version := seedTranslationArticle(t, f, "main", "zh")
	body := strings.Replace(`{"version":0,"source_locale":"zh","target_locale":"en","reason":"AI 翻译：zh → en"}`, `"version":0`, `"version":`+itoa(version), 1)
	for _, tc := range []struct{ method, path, key string }{
		{"POST", "/admin/articles/" + id + "/translations", "tr-1"},
		{"GET", "/admin/content-translations/any", ""},
		{"GET", "/admin/content-translation-settings", ""},
	} {
		w := request(f, tc.method, tc.path, body, "user", tc.key)
		if w.Code != 403 {
			t.Fatalf("%s %s: code=%d want 403", tc.method, tc.path, w.Code)
		}
	}
}

func TestCreateTranslationRejectedWhileFeatureDisabled(t *testing.T) {
	f := newFixture(t)
	id, version := seedTranslationArticle(t, f, "main", "zh")
	w := request(f, "POST", "/admin/articles/"+id+"/translations", strings.Replace(translationCreateBody, "%d", itoa(version), 1), "admin", "tr-1")
	if w.Code != 503 || !strings.Contains(w.Body.String(), "CONTENT_TRANSLATION_DISABLED") {
		t.Fatalf("code=%d body=%s", w.Code, w.Body)
	}
}

func TestCreateTranslationValidatesInput(t *testing.T) {
	f := newTranslationFixture(t, nil)
	unconfirmedID, unconfirmedVersion := seedTranslationArticle(t, f, "unconfirmed", "")
	confirmedID, version := seedTranslationArticle(t, f, "confirmed", "zh")
	tests := []struct {
		name   string
		id     string
		body   string
		key    string
		want   string
		status int
	}{
		{"missing idempotency key", confirmedID, strings.Replace(translationCreateBody, "%d", itoa(version), 1), "", "Idempotency-Key", 400},
		{"source not confirmed", unconfirmedID, strings.Replace(translationCreateBody, "%d", itoa(unconfirmedVersion), 1), "k1", "CONTENT_SOURCE_REQUIRED", 400},
		{"source mismatch", confirmedID, `{"version":` + itoa(version) + `,"source_locale":"en","target_locale":"zh","reason":"AI 翻译：en → zh"}`, "k2", "CONTENT_SOURCE_MISMATCH", 400},
		{"same language", confirmedID, `{"version":` + itoa(version) + `,"source_locale":"zh","target_locale":"zh","reason":"AI 翻译"}`, "k3", "CONTENT_TRANSLATION_LANGUAGE_INVALID", 400},
		{"unsupported language", confirmedID, `{"version":` + itoa(version) + `,"source_locale":"zh","target_locale":"ja","reason":"AI 翻译"}`, "k4", "CONTENT_TRANSLATION_LANGUAGE_INVALID", 400},
		{"stale version", confirmedID, strings.Replace(translationCreateBody, "%d", itoa(version+5), 1), "k5", "CONTENT_VERSION_CONFLICT", 409},
		{"missing article", "00000000-0000-0000-0000-000000000000", strings.Replace(translationCreateBody, "%d", itoa(version), 1), "k6", "not found", 404},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := request(f, "POST", "/admin/articles/"+tt.id+"/translations", tt.body, "admin", tt.key)
			if w.Code != tt.status || !strings.Contains(w.Body.String(), tt.want) {
				t.Fatalf("code=%d want %d (%s) body=%s", w.Code, tt.status, tt.want, w.Body)
			}
		})
	}
}

func TestCreateTranslationRejectsOverLimitSource(t *testing.T) {
	f := newTranslationFixture(t, func(c *hostcfg.ContentTranslationConfig) { c.MaxZhCharacters = 3 })
	id, version := seedTranslationArticle(t, f, "main", "zh")
	w := request(f, "POST", "/admin/articles/"+id+"/translations", strings.Replace(translationCreateBody, "%d", itoa(version), 1), "admin", "tr-1")
	if w.Code != 400 || !strings.Contains(w.Body.String(), "CONTENT_TRANSLATION_TOO_LONG") {
		t.Fatalf("code=%d body=%s", w.Code, w.Body)
	}
}

func TestCreateTranslationIsIdempotentAndQueriable(t *testing.T) {
	f := newTranslationFixture(t, nil)
	id, version := seedTranslationArticle(t, f, "main", "zh")
	body := strings.Replace(translationCreateBody, "%d", itoa(version), 1)
	w := request(f, "POST", "/admin/articles/"+id+"/translations", body, "admin", "tr-1")
	if w.Code != 202 {
		t.Fatalf("create: %s", w.Body)
	}
	jobID, _ := data(t, w)["id"].(string)
	if jobID == "" || data(t, w)["status"] != "queued" || data(t, w)["phase"] != "queued" {
		t.Fatalf("create data=%v", data(t, w))
	}
	w = request(f, "POST", "/admin/articles/"+id+"/translations", body, "admin", "tr-1")
	if w.Code != 202 || data(t, w)["id"] != jobID {
		t.Fatalf("retry: code=%d data=%v", w.Code, data(t, w))
	}
	drifted := strings.Replace(translationCreateBody, "%d", itoa(version+1), 1)
	w = request(f, "POST", "/admin/articles/"+id+"/translations", drifted, "admin", "tr-1")
	if w.Code != 409 || !strings.Contains(w.Body.String(), "CONTENT_TRANSLATION_IDEMPOTENCY_CONFLICT") {
		t.Fatalf("idempotency conflict: code=%d body=%s", w.Code, w.Body)
	}
	w = request(f, "POST", "/admin/articles/"+id+"/translations", body, "admin", "tr-2")
	if w.Code != 409 || !strings.Contains(w.Body.String(), "CONTENT_TRANSLATION_IN_PROGRESS") {
		t.Fatalf("in progress: code=%d body=%s", w.Code, w.Body)
	}
	w = request(f, "GET", "/admin/content-translations/"+jobID, "", "admin", "")
	if w.Code != 200 {
		t.Fatalf("query: %s", w.Body)
	}
	payload := data(t, w)
	if payload["article_id"] != id || payload["status"] != "queued" || payload["phase"] != "queued" {
		t.Fatalf("job data=%v", payload)
	}
	if _, hasMeta := payload["check_status"]; hasMeta {
		t.Fatal("queued job must omit check_status")
	}
	w = request(f, "GET", "/admin/articles/"+id, "", "admin", "")
	if w.Code != 200 || data(t, w)["active_translation_job_id"] != jobID {
		t.Fatalf("article detail: %s", w.Body)
	}
}

func TestTranslationSettingsExposeOnlyNonSensitiveFields(t *testing.T) {
	f := newTranslationFixture(t, nil)
	w := request(f, "GET", "/admin/content-translation-settings", "", "admin", "")
	if w.Code != 200 {
		t.Fatalf("settings: %s", w.Body)
	}
	payload := data(t, w)
	if payload["enabled"] != true || payload["max_zh_characters"].(float64) != 5000 || payload["max_en_words"].(float64) != 3000 {
		t.Fatalf("settings data=%v", payload)
	}
	raw := w.Body.String()
	if strings.Contains(raw, "api_key") || strings.Contains(raw, "aistudio_test") || strings.Contains(raw, "base_url") {
		t.Fatalf("settings leaked sensitive config: %s", raw)
	}
}

func TestPublicContentNeverExposesTranslationState(t *testing.T) {
	f := newTranslationFixture(t, nil)
	id, version := seedTranslationArticle(t, f, "main", "zh")
	if w := request(f, "POST", "/admin/articles/"+id+"/translations", strings.Replace(translationCreateBody, "%d", itoa(version), 1), "admin", "tr-1"); w.Code != 202 {
		t.Fatalf("create: %s", w.Body)
	}
	publish := `{"version":` + itoa(version) + `,"locale":"zh","action":"publish","reason":"Publish source"}`
	if w := request(f, "POST", "/admin/articles/"+id+"/publications", publish, "admin", "publish-1"); w.Code != 200 {
		t.Fatalf("publish: %s", w.Body)
	}
	w := request(f, "GET", "/site/articles/resolve?path=/blog/translation-api&locale=zh", "", "", "")
	if w.Code != 200 {
		t.Fatalf("resolve: %s", w.Body)
	}
	raw := w.Body.String()
	for _, leaked := range []string{"translation_meta", "active_translation_job_id", "source_locale", "draft"} {
		if strings.Contains(raw, leaked) {
			t.Fatalf("public response leaked %q: %s", leaked, raw)
		}
	}
}

func itoa(value int64) string {
	return strconv.FormatInt(value, 10)
}
