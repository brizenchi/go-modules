# Changelog — foundation/randx

All notable changes to this module are documented here. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions
follow [SemVer](https://semver.org/) within the rules described in the
top-level [VERSIONING.md](../../VERSIONING.md).

## [Unreleased]

## [v0.1.0] — 2026

### Added

- Initial release. Crypto/rand-backed helpers:
  - `Code(n, charset)`, `MustCode`, `NumericCode` for verification codes.
  - `Bytes(n)`, `HexToken`, `URLToken`, `Base32Token` for signing keys
    and link tokens.
  - Predefined `Numeric`, `LowerAlphaNum`, `AlphaNum`, `Unambiguous`
    charsets (the last strips `0/O/1/l/I` look-alikes).
