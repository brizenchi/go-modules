# Changelog — foundation/config

All notable changes to this module are documented here. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions
follow [SemVer](https://semver.org/) within the rules described in the
top-level [VERSIONING.md](../../VERSIONING.md).

## [Unreleased]

## [v0.1.0] — 2025

### Added

- Initial release. `Load(path, envPrefix, out)` — viper-backed file +
  env-override loader. Supports YAML / TOML / JSON; `.` in key paths
  maps to `_` in env vars under the given prefix.
