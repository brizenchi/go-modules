# Changelog — email

All notable changes to this module are documented here. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions
follow [SemVer](https://semver.org/) within the rules described in the
top-level [VERSIONING.md](../../VERSIONING.md).

## [Unreleased]

## [v0.2.0] — 2026-04

### Added

- `adapter/resend` — Resend HTTP API support backed by
  `github.com/resend/resend-go/v2`. No server-side templates (Resend
  doesn't have them); pair with `adapter/gotemplate` to render locally
  before sending. Coverage 85%.
- `adapter/brevo/sender_test.go` — first test suite for the Brevo
  adapter (httptest mock); coverage 0% → 68.2%.
- `adapter/smtp/smtp_test.go` — Validate, MIME assembly (text-only,
  html-only, multipart, headers, reply-to, attachment rejection),
  address formatting; coverage 0% → 68.9%.

## [v0.1.0] — 2025

### Added

- Initial release. Provider-agnostic transactional email module with
  Brevo / SMTP / log adapters and a `gotemplate`-backed Renderer.
  Multi-tenant `Manager` for per-project Sender configuration.
