package tracing

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/otel"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestSetupRequiresServiceName(t *testing.T) {
	_, err := Setup(Config{})
	if err == nil {
		t.Fatal("expected error when service name is empty")
	}
}

func TestSetupWithoutEndpointReturnsNoopShutdown(t *testing.T) {
	shutdown, err := Setup(Config{ServiceName: "svc"})
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	if shutdown == nil {
		t.Fatal("expected shutdown function")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown() error = %v", err)
	}
	if otel.GetTextMapPropagator() == nil {
		t.Fatal("expected global propagator to be initialized")
	}
}

func TestSetupWithoutEndpointStillCreatesTraceIDsWhenSampled(t *testing.T) {
	shutdown, err := Setup(Config{ServiceName: "svc", SampleRate: 1})
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	defer Shutdown(context.Background(), shutdown)

	_, span := otel.Tracer("svc").Start(context.Background(), "test", oteltrace.WithSpanKind(oteltrace.SpanKindInternal))
	if !span.SpanContext().HasTraceID() {
		t.Fatal("expected valid trace id even when exporter is disabled")
	}
	span.End()
}

func TestShutdownHandlesError(t *testing.T) {
	var called bool
	Shutdown(context.Background(), func(context.Context) error {
		called = true
		return errors.New("boom")
	})
	if !called {
		t.Fatal("expected shutdown function to be called")
	}
}
