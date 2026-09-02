// TEMPLATE-OWNED — avoid editing; changes here conflict on upgrade.
// Register your own background work in host_jobs.go instead.
package bootstrap

import (
	"context"
	"log/slog"
	"time"
)

// Runner is a background process that lives as long as the app does.
// Start must block until ctx is cancelled, then return.
//
// Runners are started after the HTTP server and stopped before it exits,
// so a slow runner delays shutdown up to the shutdown timeout.
type Runner interface {
	Name() string
	Start(ctx context.Context) error
}

// Every returns a Runner that calls fn once per interval until ctx is
// cancelled. It does not run fn immediately on start.
//
// A returned error is logged and the loop continues — a periodic job must
// never take the process down. If fn can overrun the interval, guard it
// yourself (a DB advisory lock, or foundation/rdx for a cross-instance
// lock when you run more than one replica).
func Every(name string, interval time.Duration, fn func(ctx context.Context) error) Runner {
	return &tickerRunner{name: name, interval: interval, fn: fn}
}

type tickerRunner struct {
	name     string
	interval time.Duration
	fn       func(ctx context.Context) error
}

func (t *tickerRunner) Name() string { return t.name }

func (t *tickerRunner) Start(ctx context.Context) error {
	if t.interval <= 0 {
		return nil
	}
	ticker := time.NewTicker(t.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := t.fn(ctx); err != nil {
				slog.ErrorContext(ctx, "job failed", "job", t.name, "error", err)
			}
		}
	}
}
