# Changelog — foundation/tracing

All notable changes to this module are documented here. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions
follow [SemVer](https://semver.org/) within the rules described in the
top-level [VERSIONING.md](../../VERSIONING.md).

## [Unreleased]

### Added

- `Middleware(MiddlewareConfig)` with `SkipPaths`; `Trace` now delegates to
  it. Both are built on the official `otelgin` instrumentation, producing
  stable HTTP semantic-convention attributes and `http.server.*` metrics.
- `Config.Metrics`, `MetricsURLPath`, `MetricsInterval`: OTLP metric
  export through a global MeterProvider.
- `Config.ServiceVersion` (`service.version`).
- `Endpoint` accepts an `http://` / `https://` prefix; the scheme decides TLS.
- When `Endpoint` is empty, the standard `OTEL_EXPORTER_OTLP_ENDPOINT`,
  `OTEL_EXPORTER_OTLP_HEADERS` and `OTEL_EXPORTER_OTLP_PROTOCOL` variables
  configure the exporters, as in any OpenTelemetry SDK.
- Startup diagnostics: `tracing ready` reports `endpoint_source` and
  `auth_header`; quoted `OTEL_*` values and a missing Authorization header
  are logged as warnings.
- The SDK's internal errors are logged through slog.

### Changed

- Sampling is `ParentBased(TraceIDRatioBased)`: requests with a sampled
  `traceparent` are always recorded so distributed traces are not cut.
- Propagators are W3C TraceContext + Baggage.
- Resource includes SDK, host and runtime attributes and honours
  `OTEL_SERVICE_NAME` / `OTEL_RESOURCE_ATTRIBUTES`; the environment is
  exported as `deployment.environment.name`.
- Span attributes follow current semconv (`http.request.method`,
  `http.response.status_code`, ...) instead of a mix with legacy names.
- `http.request_id` is attached to server spans by a span processor.

### Deprecated

- `TraceIDKey` / `SpanIDKey`: the middleware no longer copies ids into the
  gin context. Use `TraceID(ctx)` / `SpanID(ctx)`.

### Added

- Initial release. OTLP tracing setup plus Gin middleware that exposes
  `trace_id` / `span_id` and attaches `request_id` to spans.
