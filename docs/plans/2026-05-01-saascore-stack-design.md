# saascore stack design

## Problem

`go-modules` already has reusable domain modules (`auth`, `billing`,
`email`, `referral`, `user`), but the standard full-stack SaaS
composition still lives mostly inside `templates/quickstart/internal/*`.

That is not a professional multi-project standard:

- the real reference integration is spread across a template, not a
  versioned shared package
- a new project still has to copy several "glue" files before it gets
  the standard auth + billing + referral flow
- cross-module conventions like signup attribution and billing-state sync
  are not owned by a reusable package
- behavior drift becomes likely once 10+ projects each tweak their own
  copy of the same wiring

The platform goal is stronger: when the user schema, Stripe linkage, auth
flow, and referral base logic are intentionally the same across many
projects, the shared repo should expose a higher-level composition layer
that a host app can bootstrap directly.

## Options

### Option A: keep wiring only in `templates/quickstart`

Pros:

- lowest immediate implementation cost
- no new module to version

Cons:

- template becomes the de facto platform
- host apps copy code instead of importing a stable abstraction
- maintenance cost grows linearly across projects

Rejected.

### Option B: move every glue piece into individual business modules

Pros:

- fewer top-level packages
- keeps composition close to the underlying modules

Cons:

- not all glue belongs to one domain module
- cross-module orchestration would leak into `auth`, `billing`, or
  `referral` unnecessarily
- host-facing composition API becomes fragmented

Rejected for first iteration.

### Option C: add `stacks/saascore`

Pros:

- clean place for standard multi-module composition
- preserves the lower-level modules as independent building blocks
- gives host apps one reusable "standard SaaS backend" entrypoint
- lets templates become thin examples instead of hidden platform code

Cons:

- one more module to version
- requires a clear scope boundary to avoid stack bloat

Recommended.

## Recommended scope

First `stacks/saascore` should own only the composition that is already
standardized across products:

- shared `modules/user` schema migration
- auth wiring against `modules/user`
- GORM-backed auth code + exchange stores
- standard email-code + Google OAuth setup
- standard billing wiring against `modules/user`
- billing webhook event repo migration
- standard referral module wiring
- standard cross-module listeners:
  - signup -> initialize free billing state
  - signup -> attribute referral when request carries `referral_code`
  - billing activation/renewal/update/reactivation/cancel/failure ->
    sync subscription summary into shared `users`
  - billing activation -> activate pending referral
- standard Gin route mounting for auth/billing/referral
- authenticated route middleware helper

`stacks/saascore` should not own product-specific semantics:

- referrer reward payout policy
- product quota/grant logic beyond shared user billing summary
- app-specific routes
- host-specific role rules if they differ from email-list config
- frontend env naming

## API shape

`stacks/saascore` should expose a small, typed entrypoint:

- `Config`
- `HostHooks`
- `PolicyHooks`
- `Stack`
- `New(db, cfg, hostHooks, policyHooks) (*Stack, error)`
- `Mount(publicGroup, userGroup)`
- `RequireUser() gin.HandlerFunc`

The stack returns the underlying `Auth`, `Billing`, `Referral`, `Email`,
and `Users` handles so hosts can still extend behavior without forking
the standard bootstrap.

## Host responsibilities after this change

For a new project reusing the standard shared schema, the host should
only need to do these things:

1. load config into the stack `Config`
2. open DB and setup infrastructure middleware
3. optionally register reward hooks / extra listeners
4. mount stack routes
5. add business-specific routes

That is materially closer to "copy template, replace env values, run".

## Migration plan

1. create `stacks/saascore`
2. move the reusable auth/email/billing/referral glue into the stack
3. migrate `templates/quickstart` to consume the stack
4. delete obsolete duplicated glue files from the template
5. update integration docs and template docs to point at the stack as
   the canonical full-stack backend composition path
