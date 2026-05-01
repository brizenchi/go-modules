# Changelog — foundation/tracing

All notable changes to this module are documented here. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions
follow [SemVer](https://semver.org/) within the rules described in the
top-level [VERSIONING.md](../../VERSIONING.md).

## [Unreleased]

### Added

- Initial release. OTLP tracing setup plus Gin middleware that exposes
  `trace_id` / `span_id` and attaches `request_id` to spans.
