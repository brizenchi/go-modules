# user

> Standard reusable user-domain module: shared `users` schema, auth bridge, and app-facing summary fields.

[![Go Reference](https://pkg.go.dev/badge/github.com/brizenchi/go-modules/modules/user.svg)](https://pkg.go.dev/github.com/brizenchi/go-modules/modules/user)

Use this module when multiple projects intentionally share the same user
table shape for identity, auth linkage, and lightweight entitlement
summary. It turns the old template-only `users` shape into a real
versioned module.

Projects with an existing user table can still skip this module and
implement the `auth` / `billing` ports directly.

## Install

```bash
go get github.com/brizenchi/go-modules/modules/user
```

## Layering

```text
domain/   pure user type + enums
adapter/
  gormrepo/      GORM schema + repo + AutoMigrate
  authstore/     auth.port.UserStore + auth.port.RoleResolver
  billingstore/  deprecated compatibility adapters for legacy
                 users-table-backed billing linkage + summary sync
```

## What it owns

- standard `users` schema
- email / OAuth identity linkage
- standard `role` and `plan` normalization
- app-facing summary fields such as `plan` and `credits`
- optional `credits` field for common wallet-style flows

## Deprecation status

`modules/user` is moving toward an identity-first boundary.

The current schema still contains legacy compatibility fields:

- `stripe_customer_id`
- `stripe_subscription_id`
- `stripe_price_id`
- `stripe_product_id`
- `billing_status`
- `billing_period_start`
- `billing_period_end`
- `cancel_effective_at`

These fields are still written by `stacks/saascore` for compatibility,
but new billing linkage and subscription state now live in
`modules/billing` tables:

- `billing_customers`
- `billing_subscriptions`
- `billing_events`

Treat the Stripe/billing columns on `users` as deprecated projection
fields. New integrations should not build fresh logic on top of them.

## What stays in the host app

- env loading
- router tree
- business listeners
- project-specific reward logic
- any extra user fields not shared across projects

## Quick start

```go
import (
	"log/slog"

	"github.com/brizenchi/go-modules/modules/user/adapter/authstore"
	"github.com/brizenchi/go-modules/modules/user/adapter/gormrepo"
)

if err := gormrepo.AutoMigrate(db); err != nil {
	slog.Error("migrate users failed", "error", err)
}

users := gormrepo.New(db)

authUsers := authstore.New(users)
roles := authstore.NewConfigRoleResolver()

_, _ = authUsers, roles
```

## Integration pattern

### With `modules/auth`

```go
auth.New(auth.Deps{
	UserStore:    authstore.New(usersRepo),
	RoleResolver: authstore.NewConfigRoleResolver(),
	// ... other auth deps
})
```

### With `modules/billing`

```go
import billingrepo "github.com/brizenchi/go-modules/modules/billing/adapter/repo"

billing.New(billing.Deps{
	Customers:    billingrepo.NewCustomerStore(db),
	UserResolver: billingrepo.NewUserResolver(db),
	// ... other billing deps
})
```

`stacks/saascore` already dual-writes billing state into the new billing
tables and keeps the legacy `users` billing columns in sync for
compatibility. If you integrate modules manually, prefer persisting
provider state via `modules/billing/adapter/repo` and treat `users.plan`
or `users.credits` as app-facing summaries.

If you still need legacy summary projection into `users`, the helper is
still available:

```go
import "github.com/brizenchi/go-modules/modules/user/adapter/billingstore"

_ = billingstore.ApplySubscriptionSnapshot(ctx, usersRepo, userID, snapshot)
```

## Testing

```bash
go test -race ./...
```

## Changelog

See [CHANGELOG.md](./CHANGELOG.md).
