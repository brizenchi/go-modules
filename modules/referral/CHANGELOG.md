# Changelog — referral

All notable changes to this module are documented here. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions
follow [SemVer](https://semver.org/) within the rules described in the
top-level [VERSIONING.md](../../VERSIONING.md).

## [Unreleased]

## [v0.1.0] — 2025

### Added

- Initial release. Schema-owning C2C referral module: code generation
  (`adapter/codegen` deterministic + random), GORM persistence with
  `AutoMigrateModels`, and an in-process event bus emitting
  `ReferralRegistered` / `ReferralActivated`. Two host integration
  calls — `Attribute.AttributeReferral` (from auth's UserSignedUp
  listener) and `Attribute.ActivateReferral` (from billing's
  SubscriptionActivated listener) — are the only points the host
  invokes; everything else is HTTP queries on the Gin handler.
