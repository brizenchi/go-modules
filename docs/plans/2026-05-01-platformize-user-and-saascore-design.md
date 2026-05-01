# Platformizing `users` and the future `saascore` stack

## Status

- Date: 2026-05-01
- Scope: `go-modules`
- Decision type: architecture / platform boundary

## Problem

`go-modules` already has reusable modules for `auth`, `billing`, `email`,
and `referral`, but the current runnable backend template still treats the
host app's `internal/appmodel/user.go` as the de facto standard user
schema.

That creates three problems:

1. New projects can copy the template quickly, but there is no formal
   shared `users` module to version, test, or document.
2. `auth`, `billing`, and `referral` all end up depending on host-side
   glue around the same user/Stripe fields, which increases repeated
   maintenance across projects.
3. The current reference architecture tells consumers to replace
   `internal/appmodel`, even when their user table is intentionally
   identical across multiple projects.

The target state is a professional multi-project shared package repo:

- shared modules are explicit and versioned
- standard tables and flows are documented in `go-modules`
- new projects can start from a template and only replace env values,
  domain branding, and product-specific reward policy
- consuming projects keep only the code that is truly product-specific

## Requirements

The platform baseline should support:

- JWT auth middleware and user session parsing
- passwordless email-code auth and OAuth user creation
- standard user schema
- standard Stripe customer/subscription linkage
- standard subscription state sync back into user data
- standard referral flow
- standard request logging and request IDs
- standard HTTP response envelope
- host-controlled reward policy only where products differ

Constraints:

- `go-modules` must remain usable by projects that already have an
  existing user table.
- Shared modules must not depend on a business project repo.
- The runnable reference and the integration docs must live in
  `go-modules`, not in `clawmesh-backend` or another host app.

## Decision

Introduce a new reusable business module:

- `modules/user`

It becomes the standard user-domain module for projects whose user table
and billing linkage are intentionally shared across products.

`modules/user` owns:

- the standard `users` schema
- GORM-backed repository helpers
- normalization rules for email, role, provider, and plan
- an `auth` adapter implementing `auth/port.UserStore`
- a config-driven `auth` role resolver
- `billing` adapters implementing `billing/port.CustomerStore` and
  `billing/port.UserResolver`
- subscription-sync helpers that copy billing snapshots back into
  standard user fields

`modules/user` does not own:

- product-specific reward semantics
- app-specific entitlement policy
- nonstandard profile fields
- app-specific RBAC or fine-grained permissions

## Architecture

### Current

```text
templates/quickstart/internal/appmodel/user.go
  ├─ auth_glue/user_store.go
  ├─ billing_glue/wire.go
  └─ referral_glue/wire.go
```

This works, but the standard user schema is hidden inside a template's
`internal` package, so it cannot be imported, versioned, or shared like a
real platform module.

### Target

```text
modules/user/
  domain/                 standard user entity + enums
  adapter/gormrepo/       GORM schema + repository + migration
  adapter/authstore/      auth.UserStore + RoleResolver
  adapter/billingstore/   billing.CustomerStore + UserResolver + sync
  README.md
  CHANGELOG.md

templates/quickstart/
  internal/auth_glue/     auth module wiring only
  internal/billing_glue/  billing module wiring only
  internal/referral_glue/ referral wiring + reward hook only
```

This keeps module logic reusable and keeps the template focused on host
assembly.

## Standard `users` schema

The shared schema covers the fields already repeated across projects:

- identity: `id`, `email`, `email_verified`, `email_verified_at`
- profile: `username`, `avatar_url`
- auth linkage: `provider`, `provider_subject`, `role`, `last_login_at`
- plan summary: `plan`, `billing_status`
- Stripe linkage:
  - `stripe_customer_id`
  - `stripe_subscription_id`
  - `stripe_price_id`
  - `stripe_product_id`
- billing period summary:
  - `billing_period_start`
  - `billing_period_end`
  - `cancel_effective_at`
- optional reward/accounting field:
  - `credits`

### Credits tradeoff

`credits` is kept in the first version for migration pragmatism because:

- the current quickstart already uses it
- several projects appear to share a common wallet-style field
- removing it immediately would force a larger refactor during the
  platformization step

But the module explicitly treats reward behavior as host-controlled.
Projects may:

- keep using `credits` directly
- ignore it
- later move rewards to a dedicated ledger module

## Host boundaries after this change

### Shared in `go-modules`

- user table definition and basic repo operations
- auth user creation / lookup bridge
- Stripe customer and subscription linkage bridge
- subscription summary synchronization
- referral core flow
- module docs and runnable templates

### Remains in host projects

- route tree
- env naming and config loading
- admin policy beyond the default email allowlist
- reward payout rules
- feature-specific user extensions
- product-specific listeners reacting to signup, billing, referral events

## Why not make `auth` own the user table?

Rejected.

Reason:

- `auth` should remain provider-agnostic and usable by projects with
  existing user tables
- the user schema now serves `auth`, `billing`, and `referral`
- a separate `modules/user` makes the cross-module ownership explicit

## Why not move directly to a monolithic `saascore` module?

Deferred.

Reason:

- the platform still benefits from independent `auth`, `billing`,
  `email`, `referral`, and `user` versioning
- the immediate pain is duplicated user/Stripe glue, not module wiring
- a stack/composition layer can be added later without collapsing module
  boundaries

## Future direction: `stacks/saascore`

After `modules/user` stabilizes, add an optional higher-level composition
layer such as:

```text
stacks/saascore/
  bootstrap/
  config/
  wiring/
```

That future stack should assemble:

- `email`
- `auth`
- `user`
- `billing`
- `referral`
- default Gin middleware and route mounting conventions

The goal of `stacks/saascore` is to reduce host glue for greenfield
projects even further, while preserving direct module access for mature
projects.

## Migration plan

### Phase 1

- add `modules/user`
- move the quickstart user schema into the module
- switch quickstart glue packages to import `modules/user`
- delete `templates/quickstart/internal/appmodel`

### Phase 2

- update root docs and integration docs to prefer `modules/user`
- make templates position `modules/user` as the default standard path
- keep documenting host-side replacement for teams with existing schemas

### Phase 3

- reduce remaining host glue across `auth` and `billing`
- consider moving more shared wiring into `modules/user` helpers or
  future `stacks/saascore`

### Phase 4

- optionally add dedicated reward/ledger abstractions if multiple
  projects converge on the same accounting model
- optionally add tracing integration beyond request IDs and access logs

## Operational standard after this change

For a new project using the standard platform shape:

1. copy `templates/quickstart`
2. copy `templates/quickstart-nextjs`
3. set env/config values
4. run DB migrations
5. configure Google OAuth / Stripe / email provider
6. replace only product branding and reward listeners as needed

That is the professional baseline this repo should optimize for.

