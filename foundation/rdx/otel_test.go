package rdx

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestInstrumentTracingOmitsArguments(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { otel.SetTracerProvider(prev); _ = tp.Shutdown(context.Background()) })

	cli := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", DialTimeout: 50 * time.Millisecond, MaxRetries: -1})
	defer cli.Close()
	if err := instrument(cli, Config{Tracing: true, Metrics: true}); err != nil {
		t.Fatalf("instrument() error = %v", err)
	}
	_ = cli.Set(context.Background(), "session", "redis-secret", time.Minute).Err()

	var sawSet bool
	for _, s := range recorder.Ended() {
		if s.Name() == "set" {
			sawSet = true
		}
		for _, kv := range s.Attributes() {
			if strings.Contains(kv.Value.Emit(), "redis-secret") {
				t.Fatalf("command argument leaked into %s", kv.Key)
			}
		}
	}
	if !sawSet {
		t.Fatalf("expected a span for SET, got %d spans", len(recorder.Ended()))
	}
}
