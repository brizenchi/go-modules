// Package tracing provides OpenTelemetry setup for Gin-based services.
//
// Setup installs the global TracerProvider, MeterProvider (optional) and
// W3C propagators, exporting over OTLP (HTTP or gRPC) to Jaeger, Tempo,
// OpenObserve or any OTLP-compatible collector. Trace and Middleware
// instrument inbound Gin requests with the official otelgin
// instrumentation, so spans and metrics follow the OpenTelemetry HTTP
// semantic conventions.
//
// Usage:
//
//	func main() {
//	    flog.Setup(flog.Config{...}) // first: OTel's own errors go to slog
//	    shutdown, err := tracing.Setup(tracing.Config{
//	        ServiceName: "my-service",
//	        Endpoint:    "localhost:4318",  // OTLP HTTP
//	        SampleRate:  1,
//	        Metrics:     true,
//	    })
//	    if err != nil { log.Fatal(err) }
//	    defer tracing.Shutdown(context.Background(), shutdown)
//
//	    r := gin.New()
//	    r.Use(
//	        ginx.RequestID(),
//	        tracing.Middleware(tracing.MiddlewareConfig{ServiceName: "my-service", SkipPaths: []string{"/health"}}),
//	        ginx.AccessLog(ginx.AccessLogConfig{SkipPaths: []string{"/health"}}),
//	        ginx.Recover(),
//	    )
//	}
package tracing

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

const (
	defaultTracesPath      = "/v1/traces"
	defaultMetricsPath     = "/v1/metrics"
	defaultMetricsInterval = time.Minute
)

// Config controls tracing initialisation. Zero values mean "disabled"
// or "use sensible default".
type Config struct {
	// ServiceName identifies this service in the trace backend.
	// Required — Setup returns an error when empty.
	ServiceName string

	// ServiceVersion is exported as service.version when set.
	ServiceVersion string

	// Project identifies the SaaS/application grouping across services.
	// When set, it is exported as service.namespace.
	Project string

	// Environment identifies the deployment environment.
	// When set, it is exported as deployment.environment.name.
	Environment string

	// Endpoint is the OTLP collector host:port.
	//   "localhost:4318" — OTLP/HTTP (default protocol)
	//   "localhost:4317" — OTLP/gRPC
	// An "http://" or "https://" prefix is accepted and also decides
	// Insecure. Empty string disables the exporters. Spans are still
	// created so trace ids keep correlating logs.
	Endpoint string

	// Protocol selects the OTLP transport: "http" (default) or "grpc".
	Protocol string

	// Insecure disables TLS for the OTLP connection.
	// Typical for local dev (jaeger:4318).
	Insecure bool

	// SampleRate is the fraction of new traces to record.
	// 0 = trace nothing (default), 1 = trace everything. Requests that
	// arrive with a traceparent follow the caller's sampling decision.
	SampleRate float64

	// Headers are extra key-value pairs sent with every OTLP request.
	// Typical use: Authorization for backends like OpenObserve.
	Headers map[string]string

	// URLPath overrides the default OTLP HTTP trace path ("/v1/traces").
	// Required for backends with a custom path prefix, e.g.
	// "/api/default/v1/traces" for OpenObserve.
	// Ignored when Protocol is "grpc".
	URLPath string

	// Metrics enables the OTLP metric exporter and installs a global
	// MeterProvider, so HTTP server/client, database and Redis
	// instrumentation report RED metrics. Requires Endpoint.
	Metrics bool

	// MetricsURLPath overrides the OTLP HTTP metrics path. Empty derives
	// it from URLPath by replacing a trailing "/v1/traces" with
	// "/v1/metrics", otherwise "/v1/metrics". Ignored for gRPC.
	MetricsURLPath string

	// MetricsInterval is how often metrics are exported. Default 60s.
	MetricsInterval time.Duration
}

// Setup initialises the global TracerProvider (and MeterProvider when
// Metrics is set) and returns a shutdown function that flushes pending
// telemetry. Call it once at process start, after foundation/slog.Setup
// and before any HTTP servers begin accepting traffic.
//
// Standard OTEL_SERVICE_NAME and OTEL_RESOURCE_ATTRIBUTES environment
// variables override the resource attributes derived from cfg.
func Setup(cfg Config) (shutdown func(context.Context) error, err error) {
	if cfg.ServiceName == "" {
		return nil, fmt.Errorf("tracing: service name is required")
	}
	cfg, err = normalizeEndpoint(cfg)
	if err != nil {
		return nil, err
	}
	installDiagnostics()
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	res, err := newResource(cfg)
	if err != nil {
		return nil, err
	}

	opts := []tracesdk.TracerProviderOption{
		tracesdk.WithResource(res),
		tracesdk.WithSampler(tracesdk.ParentBased(tracesdk.TraceIDRatioBased(cfg.SampleRate))),
		tracesdk.WithSpanProcessor(requestIDProcessor{}),
	}
	if cfg.Endpoint == "" {
		slog.Info("tracing exporter disabled (no endpoint)")
	} else {
		exp, err := newTraceExporter(cfg)
		if err != nil {
			return nil, fmt.Errorf("tracing: create exporter: %w", err)
		}
		opts = append(opts, tracesdk.WithBatcher(exp))
	}
	tp := tracesdk.NewTracerProvider(opts...)
	otel.SetTracerProvider(tp)
	shutdowns := []func(context.Context) error{tp.Shutdown}

	metricsOn := cfg.Metrics && cfg.Endpoint != ""
	if metricsOn {
		mp, err := newMeterProvider(cfg, res)
		if err != nil {
			_ = tp.Shutdown(context.Background())
			return nil, fmt.Errorf("tracing: create metric exporter: %w", err)
		}
		otel.SetMeterProvider(mp)
		shutdowns = append(shutdowns, mp.Shutdown)
	}

	slog.Info("tracing ready",
		"service", cfg.ServiceName,
		"endpoint", cfg.Endpoint,
		"protocol", protocol(cfg),
		"sample_rate", cfg.SampleRate,
		"metrics", metricsOn,
	)

	return func(ctx context.Context) error {
		var errs []error
		for _, fn := range shutdowns {
			errs = append(errs, fn(ctx))
		}
		return errors.Join(errs...)
	}, nil
}

// Shutdown is a convenience wrapper around the function returned by Setup.
func Shutdown(ctx context.Context, fn func(context.Context) error) {
	if fn == nil {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := fn(ctx); err != nil {
		slog.Error("tracing shutdown", "error", err)
	}
}

// installDiagnostics routes the SDK's own errors and debug output (export
// failures, dropped spans) through slog instead of the stdlib logger.
func installDiagnostics() {
	otel.SetLogger(logr.FromSlogHandler(slog.Default().Handler()))
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		slog.Error("opentelemetry error", "component", "otel", "error", err)
	}))
}

func newResource(cfg Config) (*resource.Resource, error) {
	attrs := []attribute.KeyValue{semconv.ServiceName(cfg.ServiceName)}
	if cfg.ServiceVersion != "" {
		attrs = append(attrs, semconv.ServiceVersion(cfg.ServiceVersion))
	}
	if cfg.Project != "" {
		attrs = append(attrs, semconv.ServiceNamespace(cfg.Project))
	}
	if cfg.Environment != "" {
		attrs = append(attrs, semconv.DeploymentEnvironmentNameKey.String(cfg.Environment))
	}
	res, err := resource.New(context.Background(),
		resource.WithSchemaURL(semconv.SchemaURL),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithProcessRuntimeName(),
		resource.WithProcessRuntimeVersion(),
		resource.WithAttributes(attrs...),
		// Last so operators can override through the environment.
		resource.WithFromEnv(),
	)
	if errors.Is(err, resource.ErrPartialResource) {
		slog.Warn("tracing: partial resource", "error", err)
		return res, nil
	}
	if err != nil {
		return nil, fmt.Errorf("tracing: build resource: %w", err)
	}
	return res, nil
}

// normalizeEndpoint accepts "scheme://host:port" for convenience. The
// OTLP exporters expect a bare host:port; the scheme selects TLS.
func normalizeEndpoint(cfg Config) (Config, error) {
	raw := strings.TrimSpace(cfg.Endpoint)
	if !strings.Contains(raw, "://") {
		cfg.Endpoint = raw
		return cfg, nil
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return cfg, fmt.Errorf("tracing: invalid endpoint %q", cfg.Endpoint)
	}
	switch strings.ToLower(u.Scheme) {
	case "http":
		cfg.Insecure = true
	case "https":
		cfg.Insecure = false
	default:
		return cfg, fmt.Errorf("tracing: unsupported endpoint scheme %q", u.Scheme)
	}
	if p := strings.TrimRight(u.Path, "/"); p != "" && cfg.URLPath == "" {
		slog.Warn("tracing: endpoint path ignored; set URLPath", "endpoint", raw)
	}
	cfg.Endpoint = u.Host
	return cfg, nil
}

func protocol(cfg Config) string {
	if strings.EqualFold(cfg.Protocol, "grpc") {
		return "grpc"
	}
	return "http"
}

func tlsConfig() *tls.Config {
	return &tls.Config{MinVersion: tls.VersionTLS12}
}

func newTraceExporter(cfg Config) (tracesdk.SpanExporter, error) {
	if protocol(cfg) == "grpc" {
		opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(cfg.Endpoint)}
		if cfg.Insecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}
		if len(cfg.Headers) > 0 {
			opts = append(opts, otlptracegrpc.WithHeaders(cfg.Headers))
		}
		return otlptracegrpc.New(context.Background(), opts...)
	}
	opts := []otlptracehttp.Option{otlptracehttp.WithEndpoint(cfg.Endpoint)}
	if cfg.Insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	} else {
		opts = append(opts, otlptracehttp.WithTLSClientConfig(tlsConfig()))
	}
	if len(cfg.Headers) > 0 {
		opts = append(opts, otlptracehttp.WithHeaders(cfg.Headers))
	}
	if cfg.URLPath != "" {
		opts = append(opts, otlptracehttp.WithURLPath(cfg.URLPath))
	}
	return otlptracehttp.New(context.Background(), opts...)
}

func newMeterProvider(cfg Config, res *resource.Resource) (*metricsdk.MeterProvider, error) {
	exp, err := newMetricExporter(cfg)
	if err != nil {
		return nil, err
	}
	interval := cfg.MetricsInterval
	if interval <= 0 {
		interval = defaultMetricsInterval
	}
	return metricsdk.NewMeterProvider(
		metricsdk.WithResource(res),
		metricsdk.WithReader(metricsdk.NewPeriodicReader(exp, metricsdk.WithInterval(interval))),
	), nil
}

func newMetricExporter(cfg Config) (metricsdk.Exporter, error) {
	if protocol(cfg) == "grpc" {
		opts := []otlpmetricgrpc.Option{otlpmetricgrpc.WithEndpoint(cfg.Endpoint)}
		if cfg.Insecure {
			opts = append(opts, otlpmetricgrpc.WithInsecure())
		}
		if len(cfg.Headers) > 0 {
			opts = append(opts, otlpmetricgrpc.WithHeaders(cfg.Headers))
		}
		return otlpmetricgrpc.New(context.Background(), opts...)
	}
	opts := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpoint(cfg.Endpoint),
		otlpmetrichttp.WithURLPath(metricsURLPath(cfg)),
	}
	if cfg.Insecure {
		opts = append(opts, otlpmetrichttp.WithInsecure())
	} else {
		opts = append(opts, otlpmetrichttp.WithTLSClientConfig(tlsConfig()))
	}
	if len(cfg.Headers) > 0 {
		opts = append(opts, otlpmetrichttp.WithHeaders(cfg.Headers))
	}
	return otlpmetrichttp.New(context.Background(), opts...)
}

func metricsURLPath(cfg Config) string {
	if cfg.MetricsURLPath != "" {
		return cfg.MetricsURLPath
	}
	if prefix, ok := strings.CutSuffix(cfg.URLPath, defaultTracesPath); ok {
		return prefix + defaultMetricsPath
	}
	return defaultMetricsPath
}
