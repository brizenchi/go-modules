// TEMPLATE-OWNED — avoid editing; changes here conflict on upgrade.
// Background work belongs in internal/bootstrap/host_jobs.go, not here.
package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/brizenchi/quickstart-template/internal/bootstrap"
)

func main() {
	app, err := bootstrap.New()
	if err != nil {
		log.Fatalf("bootstrap.New: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx); err != nil {
		log.Fatalf("run: %v", err)
	}
}
