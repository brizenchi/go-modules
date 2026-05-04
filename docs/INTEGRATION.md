# Host Integration Guide

This is the canonical guide for consuming `go-modules` from another
application repository.

## Choose the right path

1. New backend using the shared SaaS model
   Use `stacks/saascore` and start from `templates/quickstart`
2. Existing backend with its own user table or billing/auth shape
   Use the lower-level `modules/*` packages directly
3. Infra-only adoption
   Use only `foundation/*`

Use [SAASCORE_GUIDE.md](./SAASCORE_GUIDE.md) for the first path. This
document is for the integration contract, not the copy-paste checklist.
Use [CONFIG_STANDARD.md](./CONFIG_STANDARD.md) for the recommended
config/bootstrap ownership split.

## What belongs where

Keep these in the host project:

- env naming and config ownership
- router root and route tree
- role policy
- host-specific listeners and entitlements
- any non-standard user fields
- any product-specific websocket, proxy, terminal, or bot behavior

Keep these in `go-modules`:

- reusable business flows
- provider adapters
- event contracts
- standard shared compositions
- shared schemas that are intentionally identical across projects
- reusable HTTP helpers and middleware

## Recommended backend split

For a host that matches the shared SaaS model:

```text
your-app/
  cmd/
  internal/
    reward/      host reward or entitlement logic
    listener/    extra business listeners
  deploy/
  .env
```

Preferred customization surface:

- `saascore.HostHooks`
- `saascore.PolicyHooks`

If your host already has its own `users` table or a materially different
billing/auth model, do not force `saascore`. Keep your schema and adapt
the required ports directly.

## Standard backend composition

Recommended boot order:

1. Load config
2. Setup structured logging
3. Init tracing if enabled
4. Open DB
5. Init `stacks/saascore` or raw `modules/*`
6. Register host-specific listeners
7. Setup Gin + foundation middleware
8. Mount public and authenticated routes
9. Start server

Recommended foundations:

- `foundation/ginx`
- `foundation/httpresp`
- `foundation/slog`
- `foundation/tracing`
- `foundation/pgx`
- `foundation/rdx`

Recommended route split:

- public group
  - auth public routes
  - billing webhook routes
- authenticated user group
  - `stack.RequireUser()`
  - auth refresh/logout routes
  - billing user routes
  - referral user routes

## Module-by-module host responsibility

### `modules/auth`

Host owns:

- `port.UserStore`
- optional role policy
- route tree placement

### `modules/billing`

Host owns:

- customer/subscription persistence bridge
- user resolution from webhook hints
- business effects after billing events

### `modules/email`

Host owns:

- provider credentials
- sender identity

### `modules/referral`

Host owns:

- reward semantics
- any product-specific activation rules beyond the shared default

### `modules/user`

Use only when multiple projects intentionally share the same `users`
schema. Otherwise keep your existing user table in the host app.

## Frontend and callback ownership

Important separation:

- backend OAuth callback URLs belong to the backend domain
- backend Stripe webhook URLs belong to the backend domain
- frontend login/success/cancel/invite pages belong to the frontend domain

The paired frontend reference is
[`templates/quickstart-nextjs`](../templates/quickstart-nextjs/).

## Local development

While iterating locally across this repo and a host app:

- prefer a temporary host-side replace:
  `replace github.com/brizenchi/go-modules => /absolute/path/to/go-modules`
- or create a host-side `go.work` that includes both repos

That is a development convenience, not the production integration
contract.
