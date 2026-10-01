# Changelog — foundation/ginx

All notable changes to this module are documented here. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions
follow [SemVer](https://semver.org/) within the rules described in the
top-level [VERSIONING.md](../../VERSIONING.md).

## [Unreleased]

### Added

- `AccessLog` records `user_agent`, `request_size`, `response_size` and
  gin errors; panics that reach it are logged as 500 and re-raised.
- `Recover` records the panic on the active span (exception event + error
  status), logs the full stack, re-raises `http.ErrAbortHandler`, and logs
  client disconnects at WARN.
- `MaxRequestIDLength`.

### Changed

- `RequestID` replaces inbound ids that are longer than 128 characters or
  contain characters outside `[A-Za-z0-9._:-]` (log-injection guard), and
  also stores the id under `foundation/slog.RequestIDKey`, so every
  `slog.*Context` record carries `request_id`.
- `AccessLog` reads `trace_id` / `span_id` from the request context instead
  of gin keys. Recommended order is now
  `RequestID → tracing → AccessLog → Recover`.

### Added

- `RequestIDFromContext(*gin.Context)` helper for retrieving the
  propagated request id from handlers and middleware.

## [v0.1.0] — 2025

### Added

- Initial release. Standard Gin middleware: `Recover` (panic → slog +
  500 envelope), `RequestID` (X-Request-ID propagation), `AccessLog`
  (one structured slog record per request, with `SkipPaths`), `CORS`
  (allowlist origins/methods/headers), `NoCache` (anti-cache headers),
  `Secure` (HSTS + optional CSP).
