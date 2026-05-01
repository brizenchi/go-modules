# user

> Standard reusable user-domain module: shared `users` schema, auth bridge, billing bridge.

[![Go Reference](https://pkg.go.dev/badge/github.com/brizenchi/go-modules/modules/user.svg)](https://pkg.go.dev/github.com/brizenchi/go-modules/modules/user)

Use this module when multiple projects intentionally share the same user
table and Stripe linkage. It turns the old template-only `users` shape
into a real versioned module.

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
  billingstore/  billing.port.CustomerStore + billing.port.UserResolver
                 + subscription sync helpers
```

## What it owns

- standard `users` schema
- email / OAuth identity linkage
- standard `role` and `plan` normalization
- Stripe customer / subscription fields
- billing status summary fields
- optional `credits` field for common wallet-style flows

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
	"github.com/brizenchi/go-modules/modules/user/adapter/billingstore"
	"github.com/brizenchi/go-modules/modules/user/adapter/gormrepo"
)

if err := gormrepo.AutoMigrate(db); err != nil {
	slog.Error("migrate users failed", "error", err)
}

users := gormrepo.New(db)

authUsers := authstore.New(users)
roles := authstore.NewConfigRoleResolver()

billingCustomers := billingstore.NewCustomerStore(users)
billingUsers := billingstore.NewUserResolver(users)

_, _, _, _ = authUsers, roles, billingCustomers, billingUsers
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
billing.New(billing.Deps{
	Customers:    billingstore.NewCustomerStore(usersRepo),
	UserResolver: billingstore.NewUserResolver(usersRepo),
	// ... other billing deps
})
```

When billing events arrive, sync the provider snapshot back into the
shared `users` fields:

```go
_ = billingstore.ApplySubscriptionSnapshot(ctx, usersRepo, userID, snapshot)
```

## Testing

```bash
go test -race ./...
```

## Changelog

See [CHANGELOG.md](./CHANGELOG.md).

