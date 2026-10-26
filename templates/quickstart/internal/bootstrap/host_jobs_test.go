package bootstrap

import (
	"testing"

	"github.com/brizenchi/quickstart-template/internal/hostapi"
	"github.com/brizenchi/quickstart-template/internal/hostcfg"
)

func TestBuildHostJobsRegistersTranslationWorkerOnlyWhenEnabled(t *testing.T) {
	for _, runner := range buildHostJobs(hostapi.Deps{}) {
		if runner.Name() == "content-translation" {
			t.Fatal("translation worker registered while disabled")
		}
	}
	enabled := hostapi.Deps{Config: hostcfg.Config{ContentTranslation: enabledContentTranslationForTest()}}
	names := map[string]bool{}
	for _, runner := range buildHostJobs(enabled) {
		names[runner.Name()] = true
	}
	if !names["content-translation"] {
		t.Fatalf("translation worker missing: %v", names)
	}
}
