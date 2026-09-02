package bootstrap

import "github.com/brizenchi/quickstart-template/internal/hostapi"

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
	_ = deps
	return nil
}
