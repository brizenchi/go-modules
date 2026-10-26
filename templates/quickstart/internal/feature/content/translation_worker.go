package content

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// TranslationWorker claims queued jobs one at a time and runs each through
// the bounded pipeline. It lives on the application lifecycle: cancelling the
// context stops it after the current job.
type TranslationWorker struct {
	service     *Service
	pipeline    *TranslationPipeline
	taskTimeout time.Duration
	idleDelay   time.Duration
	now         func() time.Time
}

func NewTranslationWorker(service *Service, pipeline *TranslationPipeline, taskTimeout, idleDelay time.Duration, now func() time.Time) *TranslationWorker {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &TranslationWorker{service: service, pipeline: pipeline, taskTimeout: taskTimeout, idleDelay: idleDelay, now: now}
}

// Run blocks until ctx is cancelled, processing jobs serially.
func (w *TranslationWorker) Run(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}
		ran, err := w.Step(ctx)
		if err != nil && ctx.Err() == nil {
			slog.WarnContext(ctx, "translation worker step failed", "error", err)
		}
		if ran {
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(w.idleDelay):
		}
	}
}

// Step performs one reclaim pass plus at most one claim and execution. It
// returns true when a job ran; tests drive Step directly with a fake clock.
func (w *TranslationWorker) Step(ctx context.Context) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if err := w.service.ReclaimExpiredTranslationJobs(ctx, w.now()); err != nil {
		return false, err
	}
	// The lease outlives the task deadline by a reclaim margin so an expired
	// worker can never commit after its replacement claims the job.
	deadline := w.now().Add(w.taskTimeout + w.taskTimeout/10)
	job, ok, err := w.service.ClaimNextTranslationJob(ctx, uuid.NewString(), deadline)
	if err != nil || !ok {
		return false, err
	}
	jobCtx, cancel := context.WithTimeout(ctx, w.taskTimeout)
	defer cancel()
	if err := w.pipeline.Run(jobCtx, job); err != nil {
		slog.Warn("translation job ended without applying a result", "job_id", job.ID, "error", err)
	}
	return true, nil
}
