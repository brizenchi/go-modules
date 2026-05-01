# Changelog — foundation/httpx

All notable changes to this module are documented here. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions
follow [SemVer](https://semver.org/) within the rules described in the
top-level [VERSIONING.md](../../VERSIONING.md).

## [Unreleased]

## [v0.1.0] — 2026

### Added

- Initial release. `NewClient(Config)` builds outbound HTTP clients with
  optional retry, circuit breaker, default headers, and request timeout.
- `DefaultTransport()` exposes a cloned and tuned `http.Transport` for
  service-to-service traffic.
