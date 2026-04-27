# Changelog — foundation/httpresp

All notable changes to this module are documented here. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions
follow [SemVer](https://semver.org/) within the rules described in the
top-level [VERSIONING.md](../../VERSIONING.md).

## [Unreleased]

## [v0.1.0] — 2025

### Added

- Initial release. Uniform `{code, msg, data}` envelope with helpers
  for the common HTTP status codes (`OK`, `OKWith`, `BadRequest`,
  `Unauthorized`, `Forbidden`, `NotFound`, `Conflict`,
  `TooManyRequests`, `Internal`).
- All error helpers call `c.AbortWithStatusJSON` so middleware after
  the handler treats the chain as terminated.
