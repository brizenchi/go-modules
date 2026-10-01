# Changelog — foundation/httpx

All notable changes to this module are documented here. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions
follow [SemVer](https://semver.org/) within the rules described in the
top-level [VERSIONING.md](../../VERSIONING.md).

## [Unreleased]

### Added

- `Config.Logging`: one slog record per attempt (method, host, path,
  status, duration; never the query string).

### Changed

- `Config.Tracing` uses the official `otelhttp` transport: a client span
  and `http.client.request.duration` per attempt, plus TraceContext and
  Baggage injection. Tracing now sits below retry, so retries are visible
  as separate spans.

## [v0.1.0] — 2026

### Added

- Initial release. `NewClient(Config)` builds outbound HTTP clients with
  optional retry, circuit breaker, default headers, and request timeout.
- `DefaultTransport()` exposes a cloned and tuned `http.Transport` for
  service-to-service traffic.
