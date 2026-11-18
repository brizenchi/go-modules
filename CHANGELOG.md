# Changelog — github.com/brizenchi/go-modules

Repository-level releases of the root module. Per-package details live in each
package's `CHANGELOG.md`; versioning rules are in [VERSIONING.md](VERSIONING.md).

## [v0.3.0] — 2026-11-18

First release of the single root module and of the template-owned composition
from [ADR-0001](docs/adr/0001-template-owned-composition-and-user-schema.md).
Products copied from the saascore template (`d6a572f`, 2026-08-07, or earlier)
upgrade with [docs/MIGRATE_FROM_SAASCORE.md](docs/MIGRATE_FROM_SAASCORE.md); it
covers code, frontend, data migration, verification and rollback, and is
rehearsed by `scripts/saascore-rehearsal/run.sh`.

### Breaking

Compared with `v0.2.1-0.20260807080018-d6a572f8ab3a` (checked with `apidiff`):

- Removed `stacks/saascore` and `modules/user` (all subpackages). The user table,
  module composition and event callbacks now belong to the copied
  `templates/quickstart`.
- Renamed `cmd/saascore-billing-migrate` to `cmd/legacy-billing-migrate`.
- `modules/auth`: access tokens carry `token_type`; tokens issued by earlier
  versions are rejected, so every user signs in again after upgrading.
  `POST /auth/logout` revokes the token (`jwt.Config.Revocations`).
- `modules/auth`: OAuth flows are bound to the browser.
  `OAuthService.StartOAuth`, `OAuthCallback`, `ExchangeToken` and
  `port.ExchangeCodeStore.Consume` (and `memstore.ExchangeStore.Consume`) take an
  extra challenge/verifier argument; browser clients send `challenge` to
  `/auth/:provider/authorize` and `oauth_verifier` to `/auth/exchange-token`.
- `modules/billing`: `port.EventBus.Publish` and `eventbus.InProc.Publish` return
  `error`; `port.Provider` adds `FindCheckoutSession` and `GetCheckoutSession`;
  `repo.NewCustomerStore` and `repo.NewUserResolver` take a `port.AccountLookup`;
  `repo.SubscriptionRepo.UpsertSnapshot` takes the event time and ID and reports
  whether the snapshot was applied; `app.NewWebhookService` takes a
  `port.SubscriptionRepository`; `app.CheckoutService` is no longer comparable.
- `modules/referral`: `port.EventBus.Publish` and `eventbus.InProc.Publish`
  return `error`.

### Added

- `foundation/aix`, plus the additions listed in each package's changelog.
- `cmd/legacy-billing-migrate --json` prints only the report on stdout, so it can
  be parsed; database warnings go to stderr.
