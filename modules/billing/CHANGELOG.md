# Changelog — billing

All notable changes to this module are documented here. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions
follow [SemVer](https://semver.org/) within the rules described in the
top-level [VERSIONING.md](../../VERSIONING.md).

## [Unreleased]

### Fixed

- Preserve the remaining trial across subscription plan and billing interval
  changes, with no payment due until the original trial ends.

- Subscription previews distinguish invoice collection from the next renewal
  date and expose unused-time credits in integer minor units when available.

- Verify Lifetime uses an active one-time Stripe price before opening Checkout.
  Invalid catalog prices return 503 with reason `billing_price_configuration`
  instead of an unexplained internal error; existing recurring access is untouched.

- Include the Checkout Session ID in paid credit events so hosts can deduplicate
  one purchase across return confirmation, webhook retries and async events.

- Qualify PostgreSQL subscription snapshot conflict columns so successful Stripe
  webhooks can persist subscription state without an ambiguous-column error.

### Added

- Additive `LifetimeSubscriptionRetirer` provider capability and
  `WebhookService.WithCustomers` wiring. Verified Lifetime payments stop the
  superseded recurring contract without refund/proration, preserving retryability
  before replacing its canonical entitlement.

- Optional `CreditsCheckoutReader` capability and Stripe implementation for
  authenticated payment-return confirmation. Checks settled payment, server-owned
  product metadata, configured price and actual line-item quantity without charging.

- Optional `CheckoutService.WithObserver` and read-only checkout result facts for
  host-owned checkout history. Observation does not verify payment or change
  checkout, subscription or entitlement decisions.

- Optional `CheckoutSessionExpirer` capability; Stripe can expire abandoned open
  checkouts before switching to a different offer, while preserving paid and
  concurrently completed checkout protection.

- Optional `ContextWebhookParser`, `EventRetriever`, and `ScheduledChangeCanceler`
  provider capabilities; authenticated event replay and bounded Stripe event feed.
- Scheduled plan and pending-payment snapshots, plus an authenticated scheduled
  change cancellation endpoint. Existing provider interfaces remain compatible.
- One-time Checkout invoice creation and subscription user metadata.

- `stripe.Config.HTTPClient`: replaces the global stripe-go backend client
  when set.

### Changed

- Allow a full-price Lifetime Checkout alongside a recurring subscription and
  safely replace that subscription's settled Checkout reservation. Unpaid
  completed sessions and existing Lifetime ownership remain protected.
- Apply higher-tier upgrades immediately even when switching from yearly to
  monthly billing; preview and confirmation use the same prorated/reset policy.

- Fix Stripe schedule creation's incompatible `from_subscription` / `end_behavior`
  parameters and mutually exclusive reactivation parameters. Release completed
  schedules before subsequent plan changes or cancellation.

- Support both legacy and modern Stripe invoice/subscription webhook shapes;
  retain failed-payment status when cancellation is also scheduled.
- Downgrades across billing intervals wait until period end. Pending changes
  block conflicting mutations with a stable HTTP 409 reason.

- Stripe API calls carry the caller's `context.Context`, so they join the
  request trace and honour cancellation.
- stripe-go's own diagnostics go through slog instead of stderr.

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
