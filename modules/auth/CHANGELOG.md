# Changelog — auth

All notable changes to this module are documented here. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions
follow [SemVer](https://semver.org/) within the rules described in the
top-level [VERSIONING.md](../../VERSIONING.md).

## [Unreleased]

## [v0.1.1] — 2025

### Fixed

- `adapter/emailcode` no longer panics when the mailer fails in debug
  mode — surfaces the underlying email error instead.

## [v0.1.0] — 2025

### Added

- Initial release. Email-code passwordless flow + Google OAuth
  (`adapter/google`) + JWT session/WS-ticket signers (`adapter/jwt`) +
  in-memory rate-limit + exchange-code stores (`adapter/memstore`) +
  in-process event bus (`adapter/eventbus`).
- DDD layout: `domain/` (Identity, Token, OAuthProfile, errors),
  `event/` (UserSignedUp, UserLoggedIn), `port/` (`UserStore`,
  `RoleResolver`, `IdentityProvider`, `TokenSigner`, `WSTicketSigner`,
  `EmailCodeIssuer`/`Verifier`, `ExchangeCodeStore`, `EventBus`),
  `app/` use cases, `http/` Gin handlers + `RequireUser` middleware.
