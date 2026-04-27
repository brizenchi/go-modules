# Changelog — foundation/ginx

All notable changes to this module are documented here. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions
follow [SemVer](https://semver.org/) within the rules described in the
top-level [VERSIONING.md](../../VERSIONING.md).

## [Unreleased]

## [v0.1.0] — 2025

### Added

- Initial release. Standard Gin middleware: `Recover` (panic → slog +
  500 envelope), `RequestID` (X-Request-ID propagation), `AccessLog`
  (one structured slog record per request, with `SkipPaths`), `CORS`
  (allowlist origins/methods/headers), `NoCache` (anti-cache headers),
  `Secure` (HSTS + optional CSP).
