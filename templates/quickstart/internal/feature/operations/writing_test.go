package operations

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"mime/multipart"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/brizenchi/go-modules/foundation/aix"
	"github.com/brizenchi/quickstart-template/internal/feature/writer"
)

type scriptedModel struct {
	replies []string
	prompts []string
}

func (s *scriptedModel) Complete(_ context.Context, messages []aix.Message) (aix.Completion, error) {
	s.prompts = append(s.prompts, messages[len(messages)-1].Content)
	reply := s.replies[0]
	s.replies = s.replies[1:]
	return aix.Completion{Content: reply, FinishReason: "stop"}, nil
}

func TestWritingIsAdminOnlyAndExplainsMissingConfiguration(t *testing.T) {
	f := newFixture(t)
	brief := `{"kind":"features","keyword":"team workspaces","notes":"","locale":"en"}`
	if w := request(f, "POST", "/admin/articles/drafts", brief, "user", ""); w.Code != 403 {
		t.Fatalf("customers cannot draft: %d %s", w.Code, w.Body)
	}
	w := request(f, "POST", "/admin/articles/drafts", brief, "admin", "")
	if w.Code != 503 || !strings.Contains(w.Body.String(), "AI_NOT_CONFIGURED") {
		t.Fatalf("an unconfigured model has a stable reason: %d %s", w.Code, w.Body)
	}
}

func TestDraftAndTranslateReturnReviewableArticles(t *testing.T) {
	f := newFixture(t)
	model := &scriptedModel{replies: []string{
		`{"slug":"team-workspaces","title":"Team workspaces","summary":"Share one plan.","seo_title":"Team workspaces for SaaS","seo_description":"Share one subscription.","body":"Intro.\n\n## Setup\n\nSee [payments](/docs/payments).","images":[{"section":"Setup","alt":"Member list","description":"Members"}]}`,
		`{"title":"团队空间","summary":"共用一个订阅。","seo_title":"团队空间","seo_description":"共用订阅。","body":"介绍。\n\n## 设置\n\n查看[支付](/docs/payments)。"}`,
	}}
	f.module.writer = writer.New(model)
	w := request(f, "POST", "/admin/articles/drafts", `{"kind":"features","keyword":"team workspaces","notes":"Invite members","locale":"en"}`, "admin", "")
	if w.Code != 200 {
		t.Fatalf("draft: %d %s", w.Code, w.Body)
	}
	var draft struct {
		Data writer.Draft `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &draft); err != nil || draft.Data.Path != "/features/team-workspaces" || len(draft.Data.Images) != 1 {
		t.Fatalf("draft response: %s", w.Body)
	}
	if strings.Contains(draft.Data.Translation.Body, "/docs/payments") {
		t.Fatalf("no published page exists, so the link must become text: %s", draft.Data.Translation.Body)
	}
	if n, _ := countArticles(f); n != 0 {
		t.Fatal("drafting must not save an article")
	}
	source, _ := json.Marshal(map[string]any{"from": "en", "to": []string{"zh"}, "translation": draft.Data.Translation})
	w = request(f, "POST", "/admin/articles/translations", string(source), "admin", "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), "团队空间") {
		t.Fatalf("translate: %d %s", w.Code, w.Body)
	}
	for _, body := range []string{`{"from":"en","to":[],"translation":{"title":"x","body":"y"}}`, `{"from":"en","to":["en"],"translation":{"title":"x","body":"y"}}`, `{"from":"en","to":["zh","zh"],"translation":{"title":"x","body":"y"}}`, `{"from":"en","to":["pt"],"translation":{"title":"x","body":"y"}}`} {
		if w := request(f, "POST", "/admin/articles/translations", body, "admin", ""); w.Code != 400 {
			t.Fatalf("invalid targets accepted: %s %d", body, w.Code)
		}
	}
}

func countArticles(f fixture) (int64, error) {
	var n int64
	err := f.db.Table("content_articles").Count(&n).Error
	return n, err
}

func siteImageRequest(t *testing.T, f fixture, body []byte, role string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	form := multipart.NewWriter(&buf)
	part, err := form.CreateFormFile("file", "figure.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := form.Close(); err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("POST", "/api/v1/admin/site-images", &buf)
	r.Header.Set("Content-Type", form.FormDataContentType())
	r.Header.Set("X-Test-Role", role)
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, r)
	return w
}

func TestArticleImagesArePublicWithDimensionsAndLongCaching(t *testing.T) {
	f := newFixture(t)
	var picture bytes.Buffer
	if err := png.Encode(&picture, image.NewRGBA(image.Rect(0, 0, 4, 3))); err != nil {
		t.Fatal(err)
	}
	if w := siteImageRequest(t, f, picture.Bytes(), "user"); w.Code != 403 {
		t.Fatalf("only administrators upload article images: %d", w.Code)
	}
	w := siteImageRequest(t, f, picture.Bytes(), "admin")
	var uploaded struct {
		Data SiteImage `json:"data"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &uploaded) != nil || uploaded.Data.URL != "/api/v1/site/images/"+uploaded.Data.ID+"?w=4&h=3" {
		t.Fatalf("upload: %d %s", w.Code, w.Body)
	}
	public := request(f, "GET", "/site/images/"+uploaded.Data.ID, "", "", "")
	if public.Header().Get("Expires") != "" || public.Header().Get("Last-Modified") == "" {
		t.Fatalf("caching headers: %v", public.Header())
	}
	if public.Code != 200 || public.Header().Get("Content-Type") != "image/png" || public.Header().Get("Cache-Control") != "public, max-age=31536000, immutable" || !bytes.Equal(public.Body.Bytes(), picture.Bytes()) {
		t.Fatalf("public image: %d %v", public.Code, public.Header())
	}
	if w := request(f, "GET", "/site/images/not-an-image", "", "", ""); w.Code != 404 {
		t.Fatalf("unknown image: %d", w.Code)
	}
	if w := siteImageRequest(t, f, []byte("<svg onload=alert(1)>"), "admin"); w.Code != 400 {
		t.Fatalf("non-images are rejected by content: %d", w.Code)
	}
}

func TestArticlesPublishInEveryContentLanguage(t *testing.T) {
	f := newFixture(t)
	body := `{"path":"/docs/setup","kind":"docs","position":1,"version":0,"translations":{"en":{"title":"Setup","summary":"Set up.","body":"Steps."},"ja":{"title":"セットアップ","summary":"設定します。","body":"手順。"},"de":{"title":"Einrichtung","summary":"Einrichten.","body":"Schritte."}},"reason":"Multilingual guide"}`
	w := request(f, "POST", "/admin/articles", body, "admin", "multilingual-1")
	var saved struct {
		Data struct {
			ID      string `json:"id"`
			Version int64  `json:"version"`
		} `json:"data"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &saved) != nil {
		t.Fatalf("save: %d %s", w.Code, w.Body)
	}
	publish := `{"version":` + strconv.FormatInt(saved.Data.Version, 10) + `,"locale":"ja","action":"publish","reason":"Publish Japanese"}`
	if w := request(f, "POST", "/admin/articles/"+saved.Data.ID+"/publications", publish, "admin", "multilingual-2"); w.Code != 200 {
		t.Fatalf("publish ja: %d %s", w.Code, w.Body)
	}
	if w := request(f, "GET", "/site/articles/resolve?path=/docs/setup&locale=ja", "", "", ""); w.Code != 200 || !strings.Contains(w.Body.String(), "セットアップ") || !strings.Contains(w.Body.String(), `"locales":["ja"]`) {
		t.Fatalf("resolve ja: %s", w.Body)
	}
	if w := request(f, "GET", "/site/articles/resolve?path=/docs/setup&locale=de", "", "", ""); w.Code != 200 || !strings.Contains(w.Body.String(), `"article":null`) {
		t.Fatalf("unpublished German stays private: %s", w.Body)
	}
	if w := request(f, "GET", "/site/articles?locale=pt", "", "", ""); w.Code != 400 {
		t.Fatalf("unsupported locale: %d", w.Code)
	}
}
