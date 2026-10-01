# Changelog — foundation/slog

All notable changes to this module are documented here. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions
follow [SemVer](https://semver.org/) within the rules described in the
top-level [VERSIONING.md](../../VERSIONING.md).

## [Unreleased]

### Added

- `ContextWithRequestID` / `RequestIDFromContext`: the canonical request-id
  context key shared by `ginx`, `tracing` and business code.
- `Config.RedactKeys` and `DefaultRedactKeys`: values of secret keys
  (password, token, authorization, cookie, api_key, ...) are written as
  `[REDACTED]`, including inside groups.
- `trace_flags` is emitted next to `trace_id` / `span_id`.

### Fixed

- Correlation fields are no longer duplicated when the caller already
  passed them or bound them with `Logger.With` (e.g. `With(c)` followed by
  `InfoContext`). Duplicate JSON keys broke some log backends.

## [v0.1.0] — 2025

### Added

- Initial release. `Setup(Config)` configures the global `log/slog`
  handler (text or JSON), with optional default attributes and source
  line capture. Includes the Gin context helper for propagating request
  ids into log records.
