package writer

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/brizenchi/go-modules/foundation/aix"
	"github.com/brizenchi/quickstart-template/internal/feature/content"
)

type fakeModel struct {
	reply    aix.Completion
	err      error
	messages []aix.Message
}

func (f *fakeModel) Complete(_ context.Context, messages []aix.Message) (aix.Completion, error) {
	f.messages = messages
	return f.reply, f.err
}

const draftReply = "```json\n" + `{"slug":"team-workspaces","title":"团队空间：一个订阅全队使用","summary":"邀请同事共用一个订阅。","seo_title":"团队空间 SaaS 功能","seo_description":"了解团队空间如何让整个团队共用一个订阅。",
"body":"团队空间让团队共用订阅。\n\n## 如何开始\n\n先[配置支付](/docs/payments)，再看[竞品](https://example.com)和[不存在的页](/docs/missing)。\n\n![乱编的图](https://example.com/x.png)\n\n## 常见问题\n\n### 需要单独购买吗？\n\n不需要。",
"images":[{"section":"如何开始","alt":"团队空间设置界面","description":"成员列表"},{"section":"x","alt":"","description":"missing alt"}]}` + "\n```"

func TestDraftBuildsASavableArticleAndKeepsOnlyKnownLinks(t *testing.T) {
	model := &fakeModel{reply: aix.Completion{Content: draftReply, FinishReason: "stop"}}
	draft, err := New(model).Draft(context.Background(), Brief{Kind: "features", Keyword: " 团队空间 ", Notes: "支持邀请成员", Locale: "zh", Pages: []Page{{Path: "/docs/payments", Title: "配置支付"}}})
	if err != nil {
		t.Fatal(err)
	}
	if draft.Path != "/features/team-workspaces" || draft.Translation.Title != "团队空间：一个订阅全队使用" {
		t.Fatalf("draft: %+v", draft)
	}
	body := draft.Translation.Body
	if !strings.Contains(body, "[配置支付](/docs/payments)") || strings.Contains(body, "example.com") || strings.Contains(body, "/docs/missing") || !strings.Contains(body, "竞品") {
		t.Fatalf("only published internal links survive, as text otherwise: %s", body)
	}
	if strings.Contains(body, "![") {
		t.Fatalf("model-written images are dropped: %s", body)
	}
	if len(draft.Images) != 1 || draft.Images[0].Alt != "团队空间设置界面" {
		t.Fatalf("image suggestions: %+v", draft.Images)
	}
	prompt := model.messages[1].Content
	if !strings.Contains(model.messages[0].Content, "Simplified Chinese") || !strings.Contains(prompt, "团队空间") || !strings.Contains(prompt, "/docs/payments") || !strings.Contains(prompt, "支持邀请成员") {
		t.Fatalf("prompt lacks the brief: %s", prompt)
	}
}

func TestDraftRejectsBadBriefsAndUnusableReplies(t *testing.T) {
	ok := Brief{Kind: "features", Keyword: "team", Locale: "en"}
	if _, err := New(nil).Draft(context.Background(), ok); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("no model: %v", err)
	}
	for name, brief := range map[string]Brief{
		"kind": {Kind: "page", Keyword: "team", Locale: "en"}, "locale": {Kind: "features", Keyword: "team", Locale: "pt"},
		"keyword": {Kind: "features", Keyword: " ", Locale: "en"}, "long keyword": {Kind: "features", Keyword: strings.Repeat("k", 121), Locale: "en"},
	} {
		if _, err := New(&fakeModel{}).Draft(context.Background(), brief); !errors.Is(err, ErrInvalidBrief) {
			t.Errorf("%s: %v", name, err)
		}
	}
	for name, reply := range map[string]aix.Completion{
		"cut off":  {Content: draftReply, FinishReason: "length"},
		"not json": {Content: "Here is your article."},
		"bad slug": {Content: strings.Replace(draftReply, "team-workspaces", "Team Workspaces", 1)},
		"no title": {Content: strings.Replace(draftReply, `"title":"团队空间：一个订阅全队使用"`, `"title":""`, 1)},
	} {
		if _, err := New(&fakeModel{reply: reply}).Draft(context.Background(), ok); !errors.Is(err, ErrInvalidOutput) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := New(&fakeModel{err: aix.ErrProvider}).Draft(context.Background(), ok); !errors.Is(err, aix.ErrProvider) {
		t.Fatalf("provider failures stay detectable: %v", err)
	}
}

func TestTranslateKeepsImagesAndOnlySourceLinks(t *testing.T) {
	source := content.Translation{Title: "团队空间", Summary: "共用订阅。", Body: "介绍。\n\n![成员列表](/api/v1/site/images/abc)\n\n先[配置支付](/docs/payments)。"}
	reply := `{"title":"Team workspaces","summary":"Share one subscription.","seo_title":"Team workspaces","seo_description":"Share one plan.","body":"Intro.\n\n![Member list](/api/v1/site/images/abc)\n\nFirst [set up payments](/docs/payments) and [read more](/blog/new)."}`
	model := &fakeModel{reply: aix.Completion{Content: reply}}
	translated, err := New(model).Translate(context.Background(), "zh", "en", source)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(translated.Body, "![Member list](/api/v1/site/images/abc)") || !strings.Contains(translated.Body, "[set up payments](/docs/payments)") || strings.Contains(translated.Body, "/blog/new") {
		t.Fatalf("translated body: %s", translated.Body)
	}
	if !strings.Contains(model.messages[0].Content, "Simplified Chinese to English") {
		t.Fatalf("instructions: %s", model.messages[0].Content)
	}
	dropped := strings.Replace(reply, `![Member list](/api/v1/site/images/abc)`, "", 1)
	if _, err := New(&fakeModel{reply: aix.Completion{Content: dropped}}).Translate(context.Background(), "zh", "en", source); !errors.Is(err, ErrInvalidOutput) {
		t.Fatalf("a dropped image must be rejected: %v", err)
	}
	for _, pair := range [][2]string{{"zh", "zh"}, {"zh", "pt"}, {"xx", "en"}} {
		if _, err := New(model).Translate(context.Background(), pair[0], pair[1], source); !errors.Is(err, ErrInvalidBrief) {
			t.Errorf("%v: %v", pair, err)
		}
	}
}
