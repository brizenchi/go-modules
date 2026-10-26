package bootstrap

import (
	"context"
	"log/slog"
	"time"

	"github.com/brizenchi/quickstart-template/internal/feature/billingrecovery"
	"github.com/brizenchi/quickstart-template/internal/feature/content"
	"github.com/brizenchi/quickstart-template/internal/hostapi"
	"github.com/brizenchi/quickstart-template/internal/platform"
)

// YOURS — edit freely.
//
// buildHostJobs registers background work: periodic jobs, queue consumers,
// anything that must run outside an HTTP request. Return an empty slice
// when you have none.
//
// Periodic job:
//
//	Every("stripe-reconcile", time.Hour, func(ctx context.Context) error {
//		return billing.Reconcile(ctx, deps.DB)
//	}),
//
// Long-lived consumer: implement Runner yourself and return it here. Start
// must block until ctx is cancelled.
//
// Note: with more than one replica every instance runs every job. Take a
// lock inside the job (foundation/rdx) or run jobs in a single dedicated
// replica.
func buildHostJobs(deps hostapi.Deps) []Runner {
	var runners []Runner
	if deps.Modules != nil && deps.Modules.Billing != nil {
		runners = append(runners, Every("billing-recovery", time.Minute, func(ctx context.Context) error {
			return billingrecovery.Run(ctx, deps.DB, deps.Modules.Billing)
		}))
	}
	if runner, ok := buildTranslationWorker(deps); ok {
		runners = append(runners, runner)
	}
	return runners
}

// buildTranslationWorker wires the AI translation worker. Its database lease
// already guarantees a single executor per job across replicas, so every
// instance may run the worker. The Gemini client copies the injected
// outbound client and refuses redirects.
func buildTranslationWorker(deps hostapi.Deps) (Runner, bool) {
	cfg := deps.Config.ContentTranslation
	if !cfg.Enabled {
		return nil, false
	}
	client := deps.HTTPClient
	if client == nil {
		client = platform.NewOutboundHTTPClient()
	}
	provider, err := content.NewGeminiClient(content.GeminiConfig{
		BaseURL:         cfg.BaseURL,
		APIKey:          cfg.APIKey,
		Model:           cfg.Model,
		MaxOutputTokens: cfg.MaxOutputTokens,
		Timeout:         time.Duration(cfg.RequestTimeoutSeconds) * time.Second,
	}, client)
	if err != nil {
		// Startup validation owns this configuration; reaching here means the
		// process and the job queue disagree. Keep the feature off rather than
		// crash-looping the runner.
		slog.Error("content translation worker disabled by adapter configuration", "error", err)
		return nil, false
	}
	service := content.New(deps.DB, deps.Audit)
	worker := content.NewTranslationWorker(
		service,
		content.NewTranslationPipeline(service, provider, cfg.Model),
		time.Duration(cfg.TaskTimeoutSeconds)*time.Second,
		5*time.Second,
		nil,
	)
	return blockingRunner("content-translation", worker.Run), true
}

type blockingJobRunner struct {
	name string
	run  func(context.Context)
}

func (r blockingJobRunner) Name() string                    { return r.name }
func (r blockingJobRunner) Start(ctx context.Context) error { r.run(ctx); return nil }

func blockingRunner(name string, run func(context.Context)) Runner {
	return blockingJobRunner{name: name, run: run}
}
