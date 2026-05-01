# Changelog — billing

All notable changes to this module are documented here. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions
follow [SemVer](https://semver.org/) within the rules described in the
top-level [VERSIONING.md](../../VERSIONING.md).

## [Unreleased]

## [v0.2.0] — 2026-04

### Added

- HTTP `CreateCheckoutSession` accepts a `metadata` object on the
  request body. Caller fields are passed through to Stripe Session
  metadata after sanitisation (trim, drop empty pairs and reserved
  keys, cap at 20 entries). Enables Rewardful integration:
  `metadata.referral` is read by Rewardful from the Stripe Session.
- `domain.ReservedMetadataKeys` + `domain.IsReservedMetadataKey`
  centralise the system-owned metadata field names; the stripe
  provider asserts the set in its tests so adding a new system field
  without updating the reserved set fails CI.
- `adapter/stripe/provider_test.go`: full httpmock-backed coverage of
  `EnsureCustomer`, `CreateCheckout`, `CancelSubscription`,
  `ReactivateSubscription`, `GetSubscription`, `ListInvoices`,
  `GetDefaultPaymentMethod`. Coverage 39% → 76.1%.

### Changed

- `adapter/stripe.buildCheckoutMetadata` writes system fields **last**
  so caller metadata can never spoof `user_id` / `email` / `plan` etc.

## [v0.1.0] — 2025

### Added

- Initial release. Stripe-backed checkout (subscription + credits) +
  webhook idempotency + cancel/reactivate flows. DDD layout with
  pluggable `port.Provider` so non-Stripe back-ends can be added
  without touching app or http layers. Hosts integrate via
  `port.CustomerStore`, `port.UserResolver`, and event listeners.
