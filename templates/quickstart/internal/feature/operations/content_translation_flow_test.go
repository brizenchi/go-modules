package operations

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/brizenchi/quickstart-template/internal/feature/content"
)

type flowProvider struct {
	generate func(int) (content.Translation, error)
	review   func(int) (content.ReviewResult, error)
	repair   func(int) (content.Translation, error)
	gen      int
	rev      int
	rep      int
}

func (p *flowProvider) Generate(context.Context, content.Translation, string, string) (content.Translation, error) {
	p.gen++
	return p.generate(p.gen)
}

func (p *flowProvider) Review(context.Context, content.Translation, content.Translation, string, string) (content.ReviewResult, error) {
	p.rev++
	return p.review(p.rev)
}

func (p *flowProvider) Repair(_ context.Context, _ content.Translation, candidate content.Translation, _ content.ReviewResult, _, _ string) (content.Translation, error) {
	p.rep++
	if p.repair != nil {
		return p.repair(p.rep)
	}
	return candidate, nil
}

// TestContentTranslationEndToEndFlow ties the admin API, the background
// worker and the article lifecycle together with a fake provider: the old
// translation stays public while a job runs, the finished draft replaces only
// the target language, warnings remain publishable and failures keep every
// previous draft.
func TestContentTranslationEndToEndFlow(t *testing.T) {
	f := newTranslationFixture(t, nil)
	provider := &flowProvider{
		generate: func(int) (content.Translation, error) {
			return content.Translation{Title: "Quick Start", Summary: "Five minutes", Body: "# Quick Start\n\nInstall the [SDK](https://example.com)."}, nil
		},
		review: func(int) (content.ReviewResult, error) {
			return content.ReviewResult{Passed: false, Issues: []content.ReviewIssue{{Code: "meaning", Field: "body", MessageZH: "语气偏差"}}}, nil
		},
	}
	service := content.New(f.db, nil)
	worker := content.NewTranslationWorker(service, content.NewTranslationPipeline(service, provider, "gemini-3.8-flash"), 7*time.Minute, time.Second, func() time.Time { return time.Now().UTC() })

	id, version := seedTranslationArticle(t, f, "main", "zh")
	if w := request(f, "POST", "/admin/articles/"+id+"/publications", `{"version":`+itoa(version)+`,"locale":"zh","action":"publish","reason":"发布原文"}`, "admin", "publish-source"); w.Code != 200 {
		t.Fatalf("publish source: %s", w.Body)
	} else {
		version = int64(data(t, w)["version"].(float64))
	}
	body := strings.Replace(translationCreateBody, "%d", itoa(version), 1)
	w := request(f, "POST", "/admin/articles/"+id+"/translations", body, "admin", "flow-1")
	if w.Code != 202 {
		t.Fatalf("create job: %s", w.Body)
	}
	jobID, _ := data(t, w)["id"].(string)

	resolve := func(locale string) map[string]any {
		t.Helper()
		w := request(f, "GET", "/site/articles/resolve?path=/blog/translation-api-main&locale="+locale, "", "", "")
		if w.Code != 200 {
			t.Fatalf("resolve %s: %s", locale, w.Body)
		}
		article, _ := data(t, w)["article"].(map[string]any)
		return article
	}
	if got := resolve("en"); got != nil {
		t.Fatal("english draft leaked to the public site before translation")
	}
	if ran, err := worker.Step(context.Background()); err != nil || !ran {
		t.Fatalf("worker step ran=%v err=%v", ran, err)
	}
	w = request(f, "GET", "/admin/content-translations/"+jobID, "", "admin", "")
	if w.Code != 200 {
		t.Fatalf("query job: %s", w.Body)
	}
	payload := data(t, w)
	if payload["status"] != "succeeded" || payload["check_status"] != "warning" {
		t.Fatalf("job=%v", payload)
	}
	issues, _ := payload["issues"].([]any)
	if len(issues) != 1 {
		t.Fatalf("issues=%v", payload["issues"])
	}
	w = request(f, "GET", "/admin/articles/"+id, "", "admin", "")
	if w.Code != 200 {
		t.Fatalf("article detail: %s", w.Body)
	}
	detail := data(t, w)
	version2 := int64(detail["version"].(float64))
	if got := resolve("en"); got != nil {
		t.Fatal("warning draft became public without an explicit publish")
	}
	publish := `{"version":` + itoa(version2) + `,"locale":"en","action":"publish","reason":"发布译文"}`
	if w := request(f, "POST", "/admin/articles/"+id+"/publications", publish, "admin", "publish-target"); w.Code != 200 {
		t.Fatalf("publish warning draft: %s", w.Body)
	}
	english := resolve("en")
	if english == nil || english["translation"].(map[string]any)["title"] != "Quick Start" {
		t.Fatalf("published english article: %v", english)
	}
	if got := resolve("zh"); got == nil || got["translation"].(map[string]any)["title"] != "快速开始" {
		t.Fatalf("chinese publication changed: %v", got)
	}

	// A second article keeps its drafts untouched when the provider fails.
	failing := &flowProvider{generate: func(int) (content.Translation, error) {
		return content.Translation{}, content.ErrProviderTimeout
	}}
	failWorker := content.NewTranslationWorker(service, content.NewTranslationPipeline(service, failing, "gemini-3.8-flash"), 7*time.Minute, time.Second, func() time.Time { return time.Now().UTC() })
	w = request(f, "POST", "/admin/articles", `{"path":"/blog/translation-api-fail","kind":"blog","position":2,"version":0,"translations":{"zh":{"title":"失败用例","summary":"摘要","body":"正文","seo_title":"","seo_description":""},"en":{"title":"Old draft","summary":"s","body":"b","seo_title":"","seo_description":""}},"source_locale":"zh","reason":"Seed failing case"}`, "admin", "seed-fail")
	if w.Code != 200 {
		t.Fatalf("seed failing article: %s", w.Body)
	}
	failID, _ := data(t, w)["id"].(string)
	failVersion := int64(data(t, w)["version"].(float64))
	w = request(f, "POST", "/admin/articles/"+failID+"/translations", strings.Replace(translationCreateBody, "%d", itoa(failVersion), 1), "admin", "flow-2")
	if w.Code != 202 {
		t.Fatalf("create failing job: %s", w.Body)
	}
	if ran, err := failWorker.Step(context.Background()); err != nil || !ran {
		t.Fatalf("fail worker step ran=%v err=%v", ran, err)
	}
	w = request(f, "GET", "/admin/articles/"+failID, "", "admin", "")
	if w.Code != 200 {
		t.Fatalf("failing article detail: %s", w.Body)
	}
	failedDetail := data(t, w)
	translations := failedDetail["translations"].(map[string]any)
	if translations["en"].(map[string]any)["title"] != "Old draft" {
		t.Fatalf("failed job overwrote the draft: %v", translations["en"])
	}
}
