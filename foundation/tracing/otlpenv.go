package tracing

import (
	"log/slog"
	"net/url"
	"os"
	"strings"
)

// Standard OpenTelemetry exporter variables. When Config.Endpoint is empty
// and one of the endpoint variables is set, the exporters are configured
// entirely by the SDK from these variables (endpoint, headers, protocol,
// TLS), exactly like any other OpenTelemetry service.
const (
	envEndpoint        = "OTEL_EXPORTER_OTLP_ENDPOINT"
	envTracesEndpoint  = "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"
	envMetricsEndpoint = "OTEL_EXPORTER_OTLP_METRICS_ENDPOINT"
	envHeaders         = "OTEL_EXPORTER_OTLP_HEADERS"
	envTracesHeaders   = "OTEL_EXPORTER_OTLP_TRACES_HEADERS"
	envProtocol        = "OTEL_EXPORTER_OTLP_PROTOCOL"
	envTracesProtocol  = "OTEL_EXPORTER_OTLP_TRACES_PROTOCOL"
)

// exportTarget describes where telemetry goes after resolving Config and
// the standard environment.
type exportTarget struct {
	enabled  bool
	fromEnv  bool   // exporters read OTEL_EXPORTER_OTLP_* themselves
	endpoint string // for logging only
	protocol string // "http" or "grpc"
}

func resolveTarget(cfg Config) exportTarget {
	if cfg.Endpoint != "" {
		return exportTarget{enabled: true, endpoint: cfg.Endpoint, protocol: protocol(cfg)}
	}
	endpoint := firstEnv(envTracesEndpoint, envEndpoint)
	if endpoint == "" {
		return exportTarget{}
	}
	proto := "http"
	if strings.EqualFold(firstEnv(envTracesProtocol, envProtocol), "grpc") {
		proto = "grpc"
	}
	return exportTarget{enabled: true, fromEnv: true, endpoint: endpoint, protocol: proto}
}

func firstEnv(keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}

// hasAuthHeader reports whether an Authorization header will be sent,
// either from Config.Headers or from the standard header variables.
func hasAuthHeader(cfg Config) bool {
	for k := range cfg.Headers {
		if strings.EqualFold(k, "authorization") {
			return true
		}
	}
	for _, key := range []string{envTracesHeaders, envHeaders} {
		for _, pair := range strings.Split(os.Getenv(key), ",") {
			name, value, ok := strings.Cut(pair, "=")
			if !ok {
				continue
			}
			if decoded, err := url.PathUnescape(strings.TrimSpace(name)); err == nil {
				name = decoded
			}
			if strings.EqualFold(strings.TrimSpace(name), "authorization") && strings.TrimSpace(value) != "" {
				return true
			}
		}
	}
	return false
}

// warnMisquotedEnv flags values that still carry the quotes from a .env
// line. Deployment panels often pass them through verbatim, and the SDK
// then silently drops the header or rejects the endpoint.
func warnMisquotedEnv() {
	for _, key := range []string{envEndpoint, envTracesEndpoint, envMetricsEndpoint, envHeaders, envTracesHeaders, envProtocol} {
		v := strings.TrimSpace(os.Getenv(key))
		if len(v) >= 2 && (v[0] == '"' || v[0] == '\'') {
			slog.Warn("tracing: environment value is wrapped in quotes; remove them", "variable", key)
		}
	}
}
