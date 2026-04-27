# Changelog — foundation/jwt

All notable changes to this module are documented here. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions
follow [SemVer](https://semver.org/) within the rules described in the
top-level [VERSIONING.md](../../VERSIONING.md).

## [Unreleased]

## [v0.1.0] — 2025

### Added

- Initial release. `NewHS256` and `NewRS256` signers built on
  `golang-jwt/v5`, with mandatory `exp`+`iat`, configurable `Leeway`,
  and an alg-confusion guard so verify can't be tricked into treating
  an HS256 token as RS256.
- `Claims` struct with `Subject`, `TTL`, `Audience`, and arbitrary
  `Extra map[string]any`.
- Sentinel errors: `ErrSecretRequired`, `ErrInvalidToken`, `ErrExpired`,
  `ErrTTLRequired`.
