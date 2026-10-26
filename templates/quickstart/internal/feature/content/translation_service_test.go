package content

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeProvider struct {
	generateCalls int
	reviewCalls   int
	repairCalls   int
	generate      func(call int) (Translation, error)
	review        func(call int) (ReviewResult, error)
	repair        func(call int) (Translation, error)
}

func (f *fakeProvider) Generate(context.Context, Translation, string, string) (Translation, error) {
	f.generateCalls++
	return f.generate(f.generateCalls)
}

func (f *fakeProvider) Review(context.Context, Translation, Translation, string, string) (ReviewResult, error) {
	f.reviewCalls++
	return f.review(f.reviewCalls)
}

func (f *fakeProvider) Repair(context.Context, Translation, Translation, ReviewResult, string, string) (Translation, error) {
	f.repairCalls++
	return f.repair(f.repairCalls)
}

func goodCandidate() Translation {
	return Translation{Title: "Quick Start", Summary: "Run it in five minutes", Body: "# Quick Start\n\nInstall the [SDK](https://example.com).", SEOTitle: "", SEODescription: ""}
}

func claimedJob(t *testing.T, s *Service) (Article, TranslationJob) {
	t.Helper()
	article := seedArticle(t, s, "zh")
	job, err := s.CreateTranslationJob(context.Background(), jobRequest(article, "actor-1", "translate-"+t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	claimed, ok, err := s.ClaimNextTranslationJob(context.Background(), "lease-"+t.Name(), farDeadline())
	if err != nil || !ok || claimed.ID != job.ID {
		t.Fatalf("claim ok=%v err=%v", ok, err)
	}
	return article, claimed
}

func TestTranslationPipelinePassesWithTwoModelCalls(t *testing.T) {
	s := newTranslationTestService(t)
	article, job := claimedJob(t, s)
	provider := &fakeProvider{
		generate: func(int) (Translation, error) { return goodCandidate(), nil },
		review:   func(int) (ReviewResult, error) { return ReviewResult{Passed: true, Issues: []ReviewIssue{}}, nil },
		repair:   func(int) (Translation, error) { t.Fatal("repair must not run"); return Translation{}, nil },
	}
	pipeline := NewTranslationPipeline(s, provider, "gemini-3.8-flash")
	if err := pipeline.Run(context.Background(), job); err != nil {
		t.Fatal(err)
	}
	if provider.generateCalls != 1 || provider.reviewCalls != 1 || provider.repairCalls != 0 {
		t.Fatalf("calls gen=%d review=%d repair=%d want 1/1/0", provider.generateCalls, provider.reviewCalls, provider.repairCalls)
	}
	finished, err := s.TranslationJob(context.Background(), job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if finished.Status != "succeeded" || finished.ErrorReason != "" {
		t.Fatalf("job=%+v", finished)
	}
	updated, err := s.Read(context.Background(), article.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Translations["en"].Title != "Quick Start" {
		t.Fatalf("draft=%+v", updated.Translations["en"])
	}
	if meta := updated.TranslationMeta["en"]; meta.CheckStatus != "passed" || len(meta.Issues) != 0 || meta.Model != "gemini-3.8-flash" {
		t.Fatalf("meta=%+v", meta)
	}
}

func TestTranslationPipelineRepairsOnceWithFourModelCalls(t *testing.T) {
	s := newTranslationTestService(t)
	article, job := claimedJob(t, s)
	repaired := Translation{Title: "Quick Start", Summary: "Run it in five minutes", Body: "# Quick Start\n\nInstall the [SDK](https://example.com) and continue.", SEOTitle: "", SEODescription: ""}
	provider := &fakeProvider{
		generate: func(int) (Translation, error) { return goodCandidate(), nil },
		review: func(call int) (ReviewResult, error) {
			if call == 1 {
				return ReviewResult{Passed: false, Issues: []ReviewIssue{{Code: "omission", Field: "body", MessageZH: "末段缺失"}}}, nil
			}
			return ReviewResult{Passed: true, Issues: []ReviewIssue{}}, nil
		},
		repair: func(int) (Translation, error) { return repaired, nil },
	}
	if err := NewTranslationPipeline(s, provider, "gemini-3.8-flash").Run(context.Background(), job); err != nil {
		t.Fatal(err)
	}
	if provider.generateCalls != 1 || provider.reviewCalls != 2 || provider.repairCalls != 1 {
		t.Fatalf("calls gen=%d review=%d repair=%d want 1/2/1", provider.generateCalls, provider.reviewCalls, provider.repairCalls)
	}
	updated, err := s.Read(context.Background(), article.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Translations["en"].Body != repaired.Body || updated.TranslationMeta["en"].CheckStatus != "passed" {
		t.Fatalf("draft=%+v meta=%+v", updated.Translations["en"], updated.TranslationMeta["en"])
	}
}

func TestTranslationPipelineSavesWarningWhenRecheckStillFails(t *testing.T) {
	s := newTranslationTestService(t)
	article, job := claimedJob(t, s)
	provider := &fakeProvider{
		generate: func(int) (Translation, error) { return goodCandidate(), nil },
		review: func(int) (ReviewResult, error) {
			return ReviewResult{Passed: false, Issues: []ReviewIssue{{Code: "meaning", Field: "body", MessageZH: "语气偏差"}}}, nil
		},
		repair: func(int) (Translation, error) { return goodCandidate(), nil },
	}
	if err := NewTranslationPipeline(s, provider, "gemini-3.8-flash").Run(context.Background(), job); err != nil {
		t.Fatal(err)
	}
	updated, err := s.Read(context.Background(), article.ID)
	if err != nil {
		t.Fatal(err)
	}
	meta := updated.TranslationMeta["en"]
	if meta.CheckStatus != "warning" || len(meta.Issues) != 1 || meta.Issues[0].Code != "meaning" {
		t.Fatalf("meta=%+v", meta)
	}
	finished, err := s.TranslationJob(context.Background(), job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if finished.Status != "succeeded" {
		t.Fatalf("quality warning must still succeed the job: %+v", finished)
	}
}

func TestTranslationPipelineMergesProgrammaticIssuesEvenWhenReviewPasses(t *testing.T) {
	s := newTranslationTestService(t)
	article, job := claimedJob(t, s)
	broken := Translation{Title: "Quick Start", Summary: "Run it in five minutes", Body: "# Quick Start\n\nInstall the SDK.", SEOTitle: "", SEODescription: ""}
	provider := &fakeProvider{
		generate: func(int) (Translation, error) { return broken, nil },
		review: func(int) (ReviewResult, error) {
			return ReviewResult{Passed: true, Issues: []ReviewIssue{}}, nil
		},
		repair: func(int) (Translation, error) { return broken, nil },
	}
	if err := NewTranslationPipeline(s, provider, "gemini-3.8-flash").Run(context.Background(), job); err != nil {
		t.Fatal(err)
	}
	updated, err := s.Read(context.Background(), article.ID)
	if err != nil {
		t.Fatal(err)
	}
	meta := updated.TranslationMeta["en"]
	if meta.CheckStatus != "warning" {
		t.Fatalf("program issues must downgrade to warning: %+v", meta)
	}
	found := false
	for _, issue := range meta.Issues {
		if issue.Code == "format" && strings.Contains(issue.MessageZH, "https://example.com") {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing link issue: %+v", meta.Issues)
	}
	if provider.repairCalls != 1 || provider.reviewCalls != 2 {
		t.Fatalf("program issues must trigger the repair round: gen=%d review=%d repair=%d", provider.generateCalls, provider.reviewCalls, provider.repairCalls)
	}
}

func TestTranslationPipelineTechnicalFailureKeepsDrafts(t *testing.T) {
	s := newTranslationTestService(t)
	article, job := claimedJob(t, s)
	provider := &fakeProvider{
		generate: func(int) (Translation, error) { return goodCandidate(), nil },
		review:   func(int) (ReviewResult, error) { return ReviewResult{}, ErrProviderInvalidOutput },
		repair:   func(int) (Translation, error) { t.Fatal("repair must not run"); return Translation{}, nil },
	}
	if err := NewTranslationPipeline(s, provider, "gemini-3.8-flash").Run(context.Background(), job); !errors.Is(err, ErrProviderInvalidOutput) {
		t.Fatalf("err=%v", err)
	}
	finished, err := s.TranslationJob(context.Background(), job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if finished.Status != "failed" || finished.ErrorReason != ReasonTranslationInvalidOutput {
		t.Fatalf("job=%+v", finished)
	}
	updated, err := s.Read(context.Background(), article.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Translations["en"].Title != "Quick Start" || updated.Published["zh"].Title != "快速开始" {
		t.Fatalf("technical failure mutated content: %+v", updated)
	}
}

func TestTranslationPipelineMapsProviderFailuresToReasons(t *testing.T) {
	tests := []struct {
		name   string
		failed error
		want   string
	}{
		{"rate limited", ErrProviderRateLimited, ReasonTranslationRateLimited},
		{"timeout", ErrProviderTimeout, ReasonTranslationTimeout},
		{"interrupted", ErrProviderInterrupted, ReasonTranslationInterrupted},
		{"incomplete", ErrProviderIncomplete, ReasonTranslationIncomplete},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTranslationTestService(t)
			_, job := claimedJob(t, s)
			provider := &fakeProvider{
				generate: func(int) (Translation, error) { return Translation{}, tt.failed },
				review:   func(int) (ReviewResult, error) { return ReviewResult{}, nil },
			}
			if err := NewTranslationPipeline(s, provider, "gemini-3.8-flash").Run(context.Background(), job); !errors.Is(err, tt.failed) {
				t.Fatalf("err=%v", err)
			}
			finished, err := s.TranslationJob(context.Background(), job.ID)
			if err != nil {
				t.Fatal(err)
			}
			if finished.Status != "failed" || finished.ErrorReason != tt.want {
				t.Fatalf("job=%+v want reason %s", finished, tt.want)
			}
		})
	}
}

func TestTranslationPipelineVersionConflictFailsJobAndKeepsDrafts(t *testing.T) {
	s := newTranslationTestService(t)
	article, job := claimedJob(t, s)
	current, err := s.Read(context.Background(), article.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Save(context.Background(), current.ID, "actor-2", "concurrent-edit", Save{Version: current.Version, Path: current.Path, Kind: current.Kind, Translations: current.Translations, Reason: "concurrent"}); err != nil {
		t.Fatal(err)
	}
	provider := &fakeProvider{
		generate: func(int) (Translation, error) { return goodCandidate(), nil },
		review:   func(int) (ReviewResult, error) { return ReviewResult{Passed: true}, nil },
	}
	if err := NewTranslationPipeline(s, provider, "gemini-3.8-flash").Run(context.Background(), job); !errors.Is(err, ErrVersion) {
		t.Fatalf("err=%v want ErrVersion", err)
	}
	finished, err := s.TranslationJob(context.Background(), job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if finished.Status != "failed" || finished.ErrorReason != ReasonTranslationVersionConflict {
		t.Fatalf("job=%+v", finished)
	}
	after, err := s.Read(context.Background(), article.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Translations["en"].Body != goodCandidate().Body {
		t.Fatal("conflicting candidate overwrote the concurrent edit")
	}
}

func TestStructuralIssuesRequireFenceStructureNotVerbatimContent(t *testing.T) {
	source := Translation{Title: "t", Summary: "s", Body: "运行：\n\n```bash\nnpm install demo-sdk  # 安装依赖\n```\n\n完成。", SEOTitle: "", SEODescription: ""}
	translatedComments := Translation{Title: "t", Summary: "s", Body: "Run:\n\n```bash\nnpm install demo-sdk  # install dependencies\n```\n\nDone.", SEOTitle: "", SEODescription: ""}
	if issues := programmaticIssues(source, translatedComments); len(issues) != 0 {
		t.Fatalf("translated comments inside a fence must pass: %+v", issues)
	}
	dropped := Translation{Title: "t", Summary: "s", Body: "Run:\n\nDone.", SEOTitle: "", SEODescription: ""}
	if issues := programmaticIssues(source, dropped); len(issues) == 0 {
		t.Fatal("a missing fence must be reported")
	}
	relabeled := Translation{Title: "t", Summary: "s", Body: "Run:\n\n```sh\nnpm install demo-sdk\n```\n\nDone.", SEOTitle: "", SEODescription: ""}
	issues := programmaticIssues(source, relabeled)
	if len(issues) == 0 || issues[0].MessageZH != "代码块结构缺失或被改写" {
		t.Fatalf("changed language tag must be reported: %+v", issues)
	}
	inlinePreserved := Translation{Title: "t", Summary: "s", Body: "Run `npm install demo-sdk` now.", SEOTitle: "", SEODescription: ""}
	sourceInline := Translation{Title: "t", Summary: "s", Body: "运行 `npm install demo-sdk`。", SEOTitle: "", SEODescription: ""}
	if issues := programmaticIssues(sourceInline, inlinePreserved); len(issues) != 0 {
		t.Fatalf("preserved inline code must pass: %+v", issues)
	}
}

func TestProgrammaticIssuesDetectNumbersAndPlaceholders(t *testing.T) {
	source := Translation{Title: "t", Summary: "s", Body: "价格 {{price}}，共 1,024 台，加速 3.5 倍。", SEOTitle: "", SEODescription: ""}
	candidate := Translation{Title: "t", Summary: "s", Body: "Price {{price}}, 1,024 units, 3.5x faster. Extra 42.", SEOTitle: "", SEODescription: ""}
	issues := programmaticIssues(source, candidate)
	var addition, meaning bool
	for _, issue := range issues {
		switch issue.Code {
		case "addition":
			addition = true
		case "meaning":
			meaning = true
		}
	}
	if !addition {
		t.Fatalf("extra number not reported: %+v", issues)
	}
	if meaning {
		t.Fatalf("matching numbers misreported: %+v", issues)
	}
	dropped := Translation{Title: "t", Summary: "s", Body: "Price 1,024 units.", SEOTitle: "", SEODescription: ""}
	issues = programmaticIssues(source, dropped)
	if len(issues) == 0 {
		t.Fatal("missing placeholder and numbers not reported")
	}
	clean := Translation{Title: "t", Summary: "s", Body: "Price {{price}}, 1,024 units, 3.5x faster.", SEOTitle: "", SEODescription: ""}
	if issues := programmaticIssues(source, clean); len(issues) != 0 {
		t.Fatalf("clean translation flagged: %+v", issues)
	}
}
