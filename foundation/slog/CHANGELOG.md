# Changelog — foundation/slog

All notable changes to this module are documented here. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions
follow [SemVer](https://semver.org/) within the rules described in the
top-level [VERSIONING.md](../../VERSIONING.md).

## [Unreleased]

## [v0.1.0] — 2025

### Added

- Initial release. `Setup(Config)` configures the global `log/slog`
  handler (text or JSON), with optional default attributes and source
  line capture. Includes the Gin context helper for propagating request
  ids into log records.
