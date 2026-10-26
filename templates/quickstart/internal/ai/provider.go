// Package ai wires the reusable AI adapter into host-owned product features.
// It registers no HTTP routes and stores no conversations or usage counters.
package ai

import (
	"errors"
	"net/http"

	"github.com/brizenchi/go-modules/foundation/aix"
	"github.com/brizenchi/quickstart-template/internal/hostcfg"
)

// ErrDisabled reports that the host has not enabled AI integration.
var ErrDisabled = errors.New("host ai: integration disabled")

// New is called from the product's composition code with the injected client
// (deps.Modules.Config.HTTPClient). Inject the result into your own service.
func New(cfg hostcfg.AIConfig, client *http.Client) (aix.Generator, error) {
	if !cfg.Enabled {
		return nil, ErrDisabled
	}
	return aix.New(cfg.Client(), client)
}
