package ai

import (
	"errors"
	"net/http"
	"testing"

	"github.com/brizenchi/go-modules/foundation/aix"
	"github.com/brizenchi/quickstart-template/internal/hostcfg"
)

func TestFactoryRequiresExplicitConfigurationAndInjectedClient(t *testing.T) {
	if generator, err := New(hostcfg.AIConfig{}, nil); generator != nil || !errors.Is(err, ErrDisabled) {
		t.Fatalf("generator=%v err=%v", generator, err)
	}
	cfg := hostcfg.AIConfig{Enabled: true, APIKey: "sk_test_not-a-real-key", Model: "example-model"}
	if _, err := New(cfg, nil); !errors.Is(err, aix.ErrInvalidConfig) {
		t.Fatal(err)
	}
	if generator, err := New(cfg, &http.Client{}); err != nil || generator == nil {
		t.Fatal(err)
	}
}
