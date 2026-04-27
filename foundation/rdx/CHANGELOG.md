# Changelog — foundation/rdx

All notable changes to this module are documented here. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions
follow [SemVer](https://semver.org/) within the rules described in the
top-level [VERSIONING.md](../../VERSIONING.md).

## [Unreleased]

## [v0.1.0] — 2025

### Added

- Initial release. `Open(ctx, Config)` returns a `*go-redis/v9` client
  with sane pool defaults and an optional key prefix. `HealthCheck`
  for `/healthz`. `Acquire` distributed lock via SET-NX-PX with a
  Lua-scripted compare-and-delete `Unlock` to prevent stealing.
