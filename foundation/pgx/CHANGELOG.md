# Changelog — foundation/pgx

All notable changes to this module are documented here. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions
follow [SemVer](https://semver.org/) within the rules described in the
top-level [VERSIONING.md](../../VERSIONING.md).

## [Unreleased]

## [v0.1.0] — 2025

### Added

- Initial release. `Open(Config)` returns a `*gorm.DB` with standard
  pool sizing, slow-query logging via slog, and either DSN-string or
  discrete-field configuration. `HealthCheck(ctx, db)` is intended for
  Kubernetes `/healthz` handlers.
