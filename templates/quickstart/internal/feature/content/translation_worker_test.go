package content

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func farDeadline() time.Time {
	return time.Now().Add(7 * time.Minute).UTC()
}

func fixedClock(at time.Time) func() time.Time {
	return func() time.Time { return at }
}

func passingProvider() *fakeProvider {
	return &fakeProvider{
		generate: func(int) (Translation, error) { return goodCandidate(), nil },
		review:   func(int) (ReviewResult, error) { return ReviewResult{Passed: true}, nil },
		repair:   func(int) (Translation, error) { return goodCandidate(), nil },
	}
}

func TestWorkerRunsQueuedJobsSerially(t *testing.T) {
	s := newTranslationTestService(t)
	first := seedArticle(t, s, "zh")
	second := Save{Path: "/blog/worker-second", Kind: "blog", Translations: map[string]Translation{"zh": {Title: "第二篇", Summary: "s", Body: "b"}, "en": {Title: "Second", Summary: "s", Body: "b"}}}
	zh := "zh"
	second.SourceLocale = &zh
	secondArticle, err := s.Save(context.Background(), "", "actor-1", "seed-worker-second", second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateTranslationJob(context.Background(), jobRequest(first, "actor-1", "worker-1")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateTranslationJob(context.Background(), jobRequest(secondArticle, "actor-1", "worker-2")); err != nil {
		t.Fatal(err)
	}
	worker := NewTranslationWorker(s, NewTranslationPipeline(s, passingProvider(), "gemini-3.8-flash"), 7*time.Minute, time.Second, fixedClock(time.Now().UTC()))
	for range 2 {
		ran, err := worker.Step(context.Background())
		if err != nil || !ran {
			t.Fatalf("step ran=%v err=%v", ran, err)
		}
	}
	ran, err := worker.Step(context.Background())
	if err != nil || ran {
		t.Fatalf("step ran=%v err=%v want idle", ran, err)
	}
	for _, article := range []Article{first, secondArticle} {
		updated, err := s.Read(context.Background(), article.ID)
		if err != nil {
			t.Fatal(err)
		}
		if updated.Translations["en"].Title != "Quick Start" {
			t.Fatalf("article %s not translated: %+v", article.ID, updated.Translations["en"])
		}
		if updated.TranslationMeta["en"].CheckStatus != "passed" {
			t.Fatalf("meta=%+v", updated.TranslationMeta["en"])
		}
	}
}

func TestWorkerSetsLeaseDeadlineWithReclaimMargin(t *testing.T) {
	s := newTranslationTestService(t)
	article := seedArticle(t, s, "zh")
	if _, err := s.CreateTranslationJob(context.Background(), jobRequest(article, "actor-1", "worker-1")); err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	captured := TranslationJob{}
	capturing := &fakeProvider{
		generate: func(int) (Translation, error) {
			if err := s.db.Where("article_id = ?", article.ID).Take(&captured).Error; err != nil {
				t.Fatal(err)
			}
			return goodCandidate(), nil
		},
		review: func(int) (ReviewResult, error) { return ReviewResult{Passed: true}, nil },
	}
	worker := NewTranslationWorker(s, NewTranslationPipeline(s, capturing, "gemini-3.8-flash"), 7*time.Minute, time.Second, fixedClock(start))
	if ran, err := worker.Step(context.Background()); err != nil || !ran {
		t.Fatalf("ran=%v err=%v", ran, err)
	}
	if captured.LeaseToken == "" || captured.Deadline == nil {
		t.Fatalf("lease not set: %+v", captured)
	}
	want := start.Add(7*time.Minute + 42*time.Second)
	if !captured.Deadline.Equal(want) {
		t.Fatalf("deadline=%v want %v", captured.Deadline, want)
	}
}

func TestWorkerReclaimsExpiredJobsBeforeClaiming(t *testing.T) {
	s := newTranslationTestService(t)
	article := seedArticle(t, s, "zh")
	job, err := s.CreateTranslationJob(context.Background(), jobRequest(article, "actor-1", "worker-1"))
	if err != nil {
		t.Fatal(err)
	}
	stale := job
	stale.Status = "running"
	stale.LeaseToken = "lease-stale"
	stale.ActiveMarker = &stale.ArticleID
	past := time.Now().Add(-time.Hour).UTC()
	stale.Deadline = &past
	if err := s.db.Save(&stale).Error; err != nil {
		t.Fatal(err)
	}
	worker := NewTranslationWorker(s, NewTranslationPipeline(s, passingProvider(), "gemini-3.8-flash"), 7*time.Minute, time.Second, fixedClock(time.Now().UTC()))
	if ran, err := worker.Step(context.Background()); err != nil || ran {
		t.Fatalf("ran=%v err=%v want only reclaim", ran, err)
	}
	reclaimed, err := s.TranslationJob(context.Background(), job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reclaimed.Status != "failed" || reclaimed.ErrorReason != ReasonTranslationInterrupted {
		t.Fatalf("reclaimed=%+v", reclaimed)
	}
	if _, err := s.CreateTranslationJob(context.Background(), jobRequest(article, "actor-1", "worker-2")); err != nil {
		t.Fatalf("article must be free after reclaim: %v", err)
	}
}

func TestWorkerStopsOnCancelledContextWithoutClaiming(t *testing.T) {
	s := newTranslationTestService(t)
	article := seedArticle(t, s, "zh")
	if _, err := s.CreateTranslationJob(context.Background(), jobRequest(article, "actor-1", "worker-1")); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	worker := NewTranslationWorker(s, NewTranslationPipeline(s, passingProvider(), "gemini-3.8-flash"), 7*time.Minute, time.Second, fixedClock(time.Now().UTC()))
	if _, err := worker.Step(ctx); err == nil {
		t.Fatal("cancelled context must stop the worker")
	}
	var job TranslationJob
	if err := s.db.Where("article_id = ?", article.ID).Take(&job).Error; err != nil {
		t.Fatal(err)
	}
	if job.Status != "queued" || job.LeaseToken != "" {
		t.Fatalf("job was claimed after cancellation: %+v", job)
	}
}

func TestWorkerLeaseTokenIsUniquePerClaim(t *testing.T) {
	s := newTranslationTestService(t)
	first := seedArticle(t, s, "zh")
	if _, err := s.CreateTranslationJob(context.Background(), jobRequest(first, "actor-1", "worker-1")); err != nil {
		t.Fatal(err)
	}
	captured := TranslationJob{}
	capturing := &fakeProvider{
		generate: func(int) (Translation, error) {
			if err := s.db.Where("article_id = ?", first.ID).Take(&captured).Error; err != nil {
				t.Fatal(err)
			}
			return goodCandidate(), nil
		},
		review: func(int) (ReviewResult, error) { return ReviewResult{Passed: true}, nil },
	}
	worker := NewTranslationWorker(s, NewTranslationPipeline(s, capturing, "gemini-3.8-flash"), 7*time.Minute, time.Second, fixedClock(time.Now().UTC()))
	if _, err := worker.Step(context.Background()); err != nil {
		t.Fatal(err)
	}
	if captured.LeaseToken == uuid.Nil.String() || len(captured.LeaseToken) < 32 {
		t.Fatalf("lease token=%q", captured.LeaseToken)
	}
}
