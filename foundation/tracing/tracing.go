// Package tracing provides OpenTelemetry distributed-tracing setup for
// Gin-based services.
//
// It initialises a global TracerProvider backed by an OTLP exporter
// (HTTP or gRPC) so spans are forwarded to Jaeger, Tempo, or any
// OTLP-compatible backend.
//
// Usage:
//
//	func main() {
//	    shutdown, err := tracing.Setup(tracing.Config{
//	        ServiceName: "my-service",
//	        Endpoint:    "localhost:4318",  // OTLP HTTP
//	    })
//	    if err != nil { log.Fatal(err) }
//	    defer shutdown(context.Background())
//
//	    r := gin.New()
//	    r.Use(ginx.RequestID(), tracing.Trace("my-service"), ginx.AccessLog(...))
//	    // trace_id and span_id now appear in every access-log record
//	}
package tracing

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Config controls tracing initialisation. Zero values mean "disabled"
// or "use sensible default".
type Config struct {
	// ServiceName identifies this service in the trace backend.
	// Required — Setup returns an error when empty.
	ServiceName string

	// Endpoint is the OTLP collector host:port (no scheme).
	//   "localhost:4318" — OTLP/HTTP (default protocol)
	//   "localhost:4317" — OTLP/gRPC
	// Empty string disables the exporter (no-op provider).
	Endpoint string

	// Protocol selects the OTLP transport: "http" (default) or "grpc".
	Protocol string

	// Insecure disables TLS for the OTLP connection.
	// Typical for local dev (jaeger:4318).
	Insecure bool

	// SampleRate is the fraction of traces to record.
	// 0 = trace nothing (default), 1 = trace everything.
	SampleRate float64
}

// Setup initialises the global TracerProvider and returns a shutdown
// function that flushes pending spans. Call it once at process start,
// before any HTTP servers begin accepting traffic.
func Setup(cfg Config) (shutdown func(context.Context) error, err error) {
	if cfg.ServiceName == "" {
		return nil, fmt.Errorf("tracing: service name is required")
	}
	otel.SetTextMapPropagator(propagation.TraceContext{})
	opts := []tracesdk.TracerProviderOption{
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(cfg.ServiceName),
		)),
		tracesdk.WithSampler(tracesdk.TraceIDRatioBased(cfg.SampleRate)),
	}
	if cfg.Endpoint == "" {
		slog.Info("tracing exporter disabled (no endpoint)")
	} else {
		exp, err := newExporter(cfg)
		if err != nil {
			return nil, fmt.Errorf("tracing: create exporter: %w", err)
		}
		opts = append(opts, tracesdk.WithBatcher(exp))
	}

	tp := tracesdk.NewTracerProvider(opts...)

	otel.SetTracerProvider(tp)

	slog.Info("tracing ready",
		"service", cfg.ServiceName,
		"endpoint", cfg.Endpoint,
		"protocol", cfg.Protocol,
		"sample_rate", cfg.SampleRate,
	)

	return tp.Shutdown, nil
}

// Shutdown is a convenience wrapper around the function returned by Setup.
func Shutdown(ctx context.Context, fn func(context.Context) error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := fn(ctx); err != nil {
		slog.Error("tracing shutdown", "error", err)
	}
}

func newExporter(cfg Config) (tracesdk.SpanExporter, error) {
	proto := cfg.Protocol
	if proto == "" {
		proto = "http"
	}
	switch proto {
	case "grpc":
		opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(cfg.Endpoint)}
		if cfg.Insecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}
		return otlptracegrpc.New(context.Background(), opts...)
	default:
		opts := []otlptracehttp.Option{otlptracehttp.WithEndpoint(cfg.Endpoint)}
		if cfg.Insecure {
			opts = append(opts, otlptracehttp.WithInsecure())
		} else {
			opts = append(opts, otlptracehttp.WithTLSClientConfig(&tls.Config{}))
		}
		return otlptracehttp.New(context.Background(), opts...)
	}
}
