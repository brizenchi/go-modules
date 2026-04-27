package slog_test

import (
	"log/slog"

	flog "github.com/brizenchi/go-modules/foundation/slog"
)

// Example shows the standard boot-time setup. After Setup returns,
// all log/slog calls produced by the rest of the program will inherit
// the configured handler and the default attributes.
func Example() {
	flog.Setup(flog.Config{
		Level:  "info",
		Format: flog.FormatJSON,
		Defaults: map[string]any{
			"service": "auth-svc",
			"env":     "prod",
		},
	})
	slog.Info("server starting", "port", 8080)
}
