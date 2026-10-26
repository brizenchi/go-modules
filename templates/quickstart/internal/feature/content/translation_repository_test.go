package content

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/brizenchi/quickstart-template/internal/audit"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newTranslationTestService(t *testing.T) *Service {
	t.Helper()
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
	if err := db.AutoMigrate(append(audit.Models(), Models()...)...); err != nil {
		t.Fatal(err)
	}
	return New(db, nil)
}

func seedArticle(t *testing.T, s *Service, sourceLocale string) Article {
	t.Helper()
	zh := Translation{Title: "快速开始", Summary: "五分钟跑起来", Body: "# 快速开始\n\n安装 [SDK](https://example.com)。", SEOTitle: "", SEODescription: ""}
	en := Translation{Title: "Quick Start", Summary: "Run it in five minutes", Body: "# Quick Start\n\nInstall the [SDK](https://example.com).", SEOTitle: "", SEODescription: ""}
	in := Save{Path: "/blog/translation-fixture", Kind: "blog", Translations: map[string]Translation{"zh": zh, "en": en}, Reason: "seed"}
	if sourceLocale != "" {
		locale := sourceLocale
		in.SourceLocale = &locale
	}
	article, err := s.Save(context.Background(), "", "actor-1", "seed-"+t.Name(), in)
	if err != nil {
		t.Fatal(err)
	}
	if sourceLocale != "" {
		if _, err := s.Publish(context.Background(), article.ID, "actor-1", "publish-"+t.Name(), Publish{Version: article.Version, Locale: sourceLocale, Action: "publish", Reason: "seed publish"}); err != nil {
			t.Fatal(err)
		}
	}
	article, err = s.Read(context.Background(), article.ID)
	if err != nil {
		t.Fatal(err)
	}
	return article
}

func jobRequest(article Article, actor, key string) TranslationJobRequest {
	return TranslationJobRequest{
		ArticleID:      article.ID,
		ActorID:        actor,
		IdempotencyKey: key,
		Reason:         "AI 翻译：zh → en",
		SourceLocale:   "zh",
		TargetLocale:   "en",
		ArticleVersion: article.Version,
		Source:         article.Translations["zh"],
	}
}

func TestSaveSourceLocaleCompatibilityForLegacyClients(t *testing.T) {
	s := newTranslationTestService(t)
	ctx := context.Background()
	in := Save{Path: "/blog/locale-compat", Kind: "blog", Translations: map[string]Translation{"zh": {Title: "标题", Summary: "摘要", Body: "正文"}}, Reason: "legacy create"}
	article, err := s.Save(ctx, "", "actor-1", "legacy-create", in)
	if err != nil {
		t.Fatal(err)
	}
	if article.SourceLocale != "" {
		t.Fatalf("legacy save must leave source locale empty, got %q", article.SourceLocale)
	}
	if article.TranslationMeta == nil {
		t.Fatal("translation meta map must initialize")
	}
	zh := "zh"
	article, err = s.Save(ctx, article.ID, "actor-1", "set-source", Save{Version: article.Version, Path: article.Path, Kind: article.Kind, Translations: article.Translations, SourceLocale: &zh, Reason: "confirm source"})
	if err != nil {
		t.Fatal(err)
	}
	if article.SourceLocale != "zh" {
		t.Fatalf("source locale=%q want zh", article.SourceLocale)
	}
	article, err = s.Save(ctx, article.ID, "actor-1", "legacy-edit", Save{Version: article.Version, Path: article.Path, Kind: article.Kind, Translations: article.Translations, Reason: "legacy edit"})
	if err != nil {
		t.Fatal(err)
	}
	if article.SourceLocale != "zh" {
		t.Fatalf("legacy edit dropped source locale: %q", article.SourceLocale)
	}
	invalid := "ja"
	if _, err := s.Save(ctx, article.ID, "actor-1", "invalid-source", Save{Version: article.Version, Path: article.Path, Kind: article.Kind, Translations: article.Translations, SourceLocale: &invalid, Reason: "invalid"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("err=%v want ErrInvalid", err)
	}
}

func TestCreateTranslationJobIsIdempotentPerKeyAndRequest(t *testing.T) {
	s := newTranslationTestService(t)
	ctx := context.Background()
	article := seedArticle(t, s, "zh")
	job, err := s.CreateTranslationJob(ctx, jobRequest(article, "actor-1", "translate-1"))
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != "queued" || job.Phase != "queued" || job.SourceHash != HashTranslation(article.Translations["zh"]) {
		t.Fatalf("job=%+v", job)
	}
	replay, err := s.CreateTranslationJob(ctx, jobRequest(article, "actor-1", "translate-1"))
	if err != nil {
		t.Fatal(err)
	}
	if replay.ID != job.ID {
		t.Fatalf("retry created second job %s vs %s", replay.ID, job.ID)
	}
	var count int64
	s.db.Model(&TranslationJob{}).Count(&count)
	if count != 1 {
		t.Fatalf("jobs=%d want 1", count)
	}
	drifted := jobRequest(article, "actor-1", "translate-1")
	drifted.TargetLocale = "en"
	drifted.ArticleVersion = article.Version + 5
	if _, err := s.CreateTranslationJob(ctx, drifted); !errors.Is(err, audit.ErrConflict) {
		t.Fatalf("err=%v want idempotency conflict", err)
	}
}

func TestCreateTranslationJobValidatesArticleState(t *testing.T) {
	s := newTranslationTestService(t)
	ctx := context.Background()
	article := seedArticle(t, s, "zh")
	missing := jobRequest(article, "actor-1", "missing-article")
	missing.ArticleID = uuid.NewString()
	if _, err := s.CreateTranslationJob(ctx, missing); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err=%v want ErrNotFound", err)
	}
	stale := jobRequest(article, "actor-1", "stale-version")
	stale.ArticleVersion = article.Version + 1
	if _, err := s.CreateTranslationJob(ctx, stale); !errors.Is(err, ErrVersion) {
		t.Fatalf("err=%v want ErrVersion", err)
	}
	unconfirmed := jobRequest(article, "actor-1", "unconfirmed-source")
	unconfirmed.SourceLocale = "en"
	if _, err := s.CreateTranslationJob(ctx, unconfirmed); !errors.Is(err, ErrInvalid) {
		t.Fatalf("err=%v want ErrInvalid for source mismatch", err)
	}
	altered := jobRequest(article, "actor-1", "altered-source")
	altered.Source = Translation{Title: "另一篇", Summary: "s", Body: "b"}
	if _, err := s.CreateTranslationJob(ctx, altered); !errors.Is(err, ErrInvalid) {
		t.Fatalf("err=%v want ErrInvalid for altered source payload", err)
	}
}

func TestOneActiveTranslationJobPerArticle(t *testing.T) {
	s := newTranslationTestService(t)
	ctx := context.Background()
	article := seedArticle(t, s, "zh")
	if _, err := s.CreateTranslationJob(ctx, jobRequest(article, "actor-1", "first")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateTranslationJob(ctx, jobRequest(article, "actor-2", "second")); !errors.Is(err, ErrTranslationInProgress) {
		t.Fatalf("err=%v want in-progress conflict", err)
	}
}

func TestClaimNextTranslationJobLeasesOldestQueuedOnce(t *testing.T) {
	s := newTranslationTestService(t)
	ctx := context.Background()
	first := seedArticle(t, s, "zh")
	jobFirst, err := s.CreateTranslationJob(ctx, jobRequest(first, "actor-1", "first"))
	if err != nil {
		t.Fatal(err)
	}
	second := Save{Path: "/blog/translation-second", Kind: "blog", Translations: map[string]Translation{"zh": {Title: "第二篇", Summary: "s", Body: "b"}}}
	zh := "zh"
	second.SourceLocale = &zh
	articleSecond, err := s.Save(ctx, "", "actor-1", "seed-second", second)
	if err != nil {
		t.Fatal(err)
	}
	jobSecond, err := s.CreateTranslationJob(ctx, jobRequest(articleSecond, "actor-1", "second"))
	if err != nil {
		t.Fatal(err)
	}
	claimed, ok, err := s.ClaimNextTranslationJob(ctx, "lease-a", time.Now().Add(7*time.Minute).UTC())
	if err != nil || !ok || claimed.ID != jobFirst.ID {
		t.Fatalf("claim=%+v ok=%v err=%v want oldest job", claimed, ok, err)
	}
	if claimed.Status != "running" || claimed.Phase != "translating" || claimed.LeaseToken != "lease-a" || claimed.Deadline == nil {
		t.Fatalf("claimed=%+v", claimed)
	}
	if claimed.SourceTranslation().Title != "快速开始" {
		t.Fatalf("source snapshot lost: %+v", claimed.SourceTranslation())
	}
	claimed, ok, err = s.ClaimNextTranslationJob(ctx, "lease-b", time.Now().Add(7*time.Minute).UTC())
	if err != nil || !ok || claimed.ID != jobSecond.ID {
		t.Fatalf("claim=%+v ok=%v err=%v want second job", claimed, ok, err)
	}
	if _, ok, err := s.ClaimNextTranslationJob(ctx, "lease-c", time.Now().Add(7*time.Minute).UTC()); err != nil || ok {
		t.Fatalf("ok=%v err=%v want empty queue", ok, err)
	}
}

func TestCompleteTranslationJobReplacesOnlyTargetDraft(t *testing.T) {
	s := newTranslationTestService(t)
	ctx := context.Background()
	article := seedArticle(t, s, "zh")
	job, err := s.CreateTranslationJob(ctx, jobRequest(article, "actor-1", "translate-1"))
	if err != nil {
		t.Fatal(err)
	}
	claimed, ok, err := s.ClaimNextTranslationJob(ctx, "lease-a", time.Now().Add(7*time.Minute).UTC())
	if err != nil || !ok {
		t.Fatal(err)
	}
	candidate := Translation{Title: "Quick Start", Summary: "Run in five minutes", Body: "# Quick Start\n\nInstall the [SDK](https://example.com).", SEOTitle: "SEO", SEODescription: "Desc"}
	issues := []ReviewIssue{{Code: "format", Field: "body", MessageZH: "末段空行"}}
	updated, err := s.CompleteTranslationJob(ctx, claimed, candidate, "warning", issues, "gemini-3.8-flash")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Translations["en"].Title != "Quick Start" || updated.Translations["zh"].Title != "快速开始" {
		t.Fatalf("drafts=%+v", updated.Translations)
	}
	if updated.Published["zh"].Title != "快速开始" {
		t.Fatalf("published snapshot changed: %+v", updated.Published)
	}
	meta := updated.TranslationMeta["en"]
	if meta.CheckStatus != "warning" || meta.SourceHash != HashTranslation(article.Translations["zh"]) || meta.DraftHash != HashTranslation(candidate) || meta.JobID != job.ID || meta.Model != "gemini-3.8-flash" || len(meta.Issues) != 1 || meta.CompletedAt == "" {
		t.Fatalf("meta=%+v", meta)
	}
	if _, err := time.Parse(time.RFC3339, meta.CompletedAt); err != nil {
		t.Fatalf("completed_at not RFC3339: %v", err)
	}
	finished, err := s.TranslationJob(ctx, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if finished.Status != "succeeded" || finished.SourceSnapshot != "" || finished.ActiveMarker != nil {
		t.Fatalf("finished=%+v", finished)
	}
	next, err := s.CreateTranslationJob(ctx, jobRequest(updated, "actor-1", "translate-2"))
	if err != nil {
		t.Fatalf("article must accept a new job after completion: %v", err)
	}
	if next.ID == job.ID {
		t.Fatal("new job reused old id")
	}
}

func TestCompleteTranslationJobRejectsStaleWorkerAndVersionDrift(t *testing.T) {
	s := newTranslationTestService(t)
	ctx := context.Background()
	article := seedArticle(t, s, "zh")
	job, err := s.CreateTranslationJob(ctx, jobRequest(article, "actor-1", "translate-1"))
	if err != nil {
		t.Fatal(err)
	}
	claimed, ok, err := s.ClaimNextTranslationJob(ctx, "lease-a", time.Now().Add(7*time.Minute).UTC())
	if err != nil || !ok {
		t.Fatal(err)
	}
	hijacked := claimed
	hijacked.LeaseToken = "lease-b"
	candidate := Translation{Title: "Hijacked", Summary: "s", Body: "b"}
	if _, err := s.CompleteTranslationJob(ctx, hijacked, candidate, "passed", nil, "gemini-3.8-flash"); !errors.Is(err, ErrTranslationLease) {
		t.Fatalf("err=%v want lease rejection", err)
	}
	current, err := s.Read(ctx, article.ID)
	if err != nil {
		t.Fatal(err)
	}
	edited := current
	edited.Translations = map[string]Translation{"zh": current.Translations["zh"], "en": {Title: "Manual edit", Summary: "s", Body: "b"}}
	if _, err := s.Save(ctx, current.ID, "actor-2", "manual-edit", Save{Version: current.Version, Path: current.Path, Kind: current.Kind, Translations: edited.Translations, Reason: "manual"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CompleteTranslationJob(ctx, claimed, candidate, "passed", nil, "gemini-3.8-flash"); !errors.Is(err, ErrVersion) {
		t.Fatalf("err=%v want version conflict", err)
	}
	failed, err := s.TranslationJob(ctx, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if failed.Status != "failed" || failed.ErrorReason != ReasonTranslationVersionConflict {
		t.Fatalf("job=%+v", failed)
	}
	after, err := s.Read(ctx, article.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Translations["en"].Title == "Hijacked" {
		t.Fatal("rejected candidate overwrote the draft")
	}
}

func TestFailAndReclaimTranslationJobsKeepDraftsAndFreeTheArticle(t *testing.T) {
	s := newTranslationTestService(t)
	ctx := context.Background()
	article := seedArticle(t, s, "zh")
	job, err := s.CreateTranslationJob(ctx, jobRequest(article, "actor-1", "translate-1"))
	if err != nil {
		t.Fatal(err)
	}
	claimed, ok, err := s.ClaimNextTranslationJob(ctx, "lease-a", time.Now().Add(-time.Minute).UTC())
	if err != nil || !ok {
		t.Fatal(err)
	}
	if err := s.FailTranslationJob(ctx, claimed, ReasonTranslationTimeout); err != nil {
		t.Fatal(err)
	}
	failed, err := s.TranslationJob(ctx, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if failed.Status != "failed" || failed.ErrorReason != ReasonTranslationTimeout {
		t.Fatalf("job=%+v", failed)
	}
	after, err := s.Read(ctx, article.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Translations["en"].Title != "Quick Start" || after.Published["zh"].Title != "快速开始" {
		t.Fatalf("failure mutated content: %+v", after)
	}

	expiredJob, err := s.CreateTranslationJob(ctx, jobRequest(after, "actor-1", "translate-2"))
	if err != nil {
		t.Fatal(err)
	}
	stale := expiredJob
	stale.Status = "running"
	stale.LeaseToken = "lease-old"
	stale.ActiveMarker = &stale.ArticleID
	past := time.Now().Add(-time.Hour).UTC()
	stale.Deadline = &past
	if err := s.db.Save(&stale).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateTranslationJob(ctx, jobRequest(after, "actor-1", "translate-3")); !errors.Is(err, ErrTranslationInProgress) {
		t.Fatalf("err=%v want in-progress before reclaim", err)
	}
	if err := s.ReclaimExpiredTranslationJobs(ctx, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	reclaimed, err := s.TranslationJob(ctx, expiredJob.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reclaimed.Status != "failed" || reclaimed.ErrorReason != ReasonTranslationInterrupted {
		t.Fatalf("reclaimed=%+v", reclaimed)
	}
	if _, err := s.CreateTranslationJob(ctx, jobRequest(after, "actor-1", "translate-3")); err != nil {
		t.Fatalf("reclaim must free the article: %v", err)
	}
}

func TestUpdateTranslationJobPhaseRequiresRunningLease(t *testing.T) {
	s := newTranslationTestService(t)
	ctx := context.Background()
	article := seedArticle(t, s, "zh")
	if _, err := s.CreateTranslationJob(ctx, jobRequest(article, "actor-1", "translate-1")); err != nil {
		t.Fatal(err)
	}
	claimed, ok, err := s.ClaimNextTranslationJob(ctx, "lease-a", time.Now().Add(7*time.Minute).UTC())
	if err != nil || !ok {
		t.Fatal(err)
	}
	if err := s.UpdateTranslationJobPhase(ctx, claimed, "reviewing"); err != nil {
		t.Fatal(err)
	}
	updated, err := s.TranslationJob(ctx, claimed.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Phase != "reviewing" {
		t.Fatalf("phase=%s", updated.Phase)
	}
	stale := updated
	stale.LeaseToken = "lease-b"
	if err := s.UpdateTranslationJobPhase(ctx, stale, "repairing"); !errors.Is(err, ErrTranslationLease) {
		t.Fatalf("err=%v want lease rejection", err)
	}
}
