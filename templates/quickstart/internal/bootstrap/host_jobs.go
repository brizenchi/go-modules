package bootstrap

import (
	"context"
	"github.com/brizenchi/quickstart-template/internal/feature/billingrecovery"
	"github.com/brizenchi/quickstart-template/internal/hostapi"
	"time"
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
	if deps.Modules == nil || deps.Modules.Billing == nil {
		return nil
	}
	return []Runner{Every("billing-recovery", time.Minute, func(ctx context.Context) error { return billingrecovery.Run(ctx, deps.DB, deps.Modules.Billing) })}
}
