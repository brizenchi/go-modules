# Changelog — foundation/resilience

All notable changes to this module are documented here. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions
follow [SemVer](https://semver.org/) within the rules described in the
top-level [VERSIONING.md](../../VERSIONING.md).

## [Unreleased]

## [v0.1.0] — 2026

### Added

- Initial release. `Do(ctx, fn, Policy)` provides transport-agnostic retry
  with constant or exponential backoff and optional retry filtering.
- `NewBreaker(BreakerConfig)` adds a concurrency-safe circuit breaker with
  closed/open/half-open states and injectable failure classification.
