# Shared SaaS Schema Boundary Design

## Status

- Date: 2026-05-08
- Scope: `go-modules`
- Decision type: architecture / platform boundary / schema ownership
- Audience: maintainers of `modules/*`, `stacks/saascore`, and the quickstart templates

## Problem

`go-modules` is no longer just a runnable sample for one product. Its
real job is to be a reusable backend foundation for roughly 10+ SaaS
projects that intentionally share the same auth, billing, and referral
shape.

The current standard still mixes two different concerns into the shared
`users` schema:

1. identity and auth state
2. Stripe-specific commercial state

Today `modules/user` owns fields such as:

- `stripe_customer_id`
- `stripe_subscription_id`
- `stripe_price_id`
- `stripe_product_id`
- `billing_status`
- `billing_period_start`
- `billing_period_end`
- `cancel_effective_at`

That was acceptable for the first quickstart iteration, but it is not
the right long-term boundary for a shared multi-project platform:

- the user module becomes coupled to one billing implementation
- the standard schema becomes provider-specific instead of platform-level
- future providers must either reuse `stripe_*` names forever or force a
  disruptive schema rewrite
- the quickstart convenience becomes the de facto canonical architecture

The platform should optimize for a different standard:

- reusable module boundaries
- provider-agnostic shared packages
- a quickstart that is still easy to adopt
- controlled migration cost for projects already on the current shape

## Goals

- keep `stacks/saascore` as the default fast path for greenfield SaaS
  projects
- make `modules/user` a clean identity-domain module instead of a Stripe
  state container
- move provider mapping and subscription snapshot ownership to
  `modules/billing`
- preserve simple high-frequency reads for auth and entitlement checks
- avoid a flag day rewrite for existing projects already using the
  current schema

## Non-Goals

- building a full accounting ledger in the first refactor
- introducing a tenant/workspace model
- redesigning the public billing HTTP API
- removing all summary fields from `users`
- solving every possible future commerce model in one step

## Options

### Option A: keep the current `users` schema as the permanent standard

Pros:

- zero migration cost
- simplest quickstart
- current `saascore` docs remain accurate

Cons:

- shared user schema remains Stripe-flavored
- `modules/user` continues owning billing details it should not own
- future provider expansion gets harder every quarter
- the platform architecture stays template-shaped instead of module-shaped

Rejected.

### Option B: move all commercial state out of `users`, including `plan`
and `credits`

Pros:

- very clean domain separation
- maximum provider neutrality
- strongest long-term architecture

Cons:

- higher migration cost
- more joins or projection plumbing for common auth/product checks
- adds too much complexity for the next iteration of a quickstart-focused
  shared stack

Deferred.

### Option C: keep `users` small and stable, but allow summary projections
there; move provider state into billing-owned tables

Pros:

- preserves fast reads for common SaaS checks
- removes Stripe-specific fields from the shared user schema
- fits the current `saascore` event-driven composition model
- can be introduced additively with compatibility shims

Cons:

- some denormalized summary remains on `users`
- requires a dual-write migration phase

Recommended.

## Decision

Adopt Option C.

The standard platform boundary becomes:

- `modules/user` owns identity state and lightweight business summary
  fields
- `modules/billing` owns provider mappings and current commercial state
- `stacks/saascore` owns the standard projection logic from billing
  events into app-visible summary fields
- `templates/quickstart` demonstrates the stack; it no longer defines
  the canonical schema boundary

This keeps the quickstart practical without locking the platform into a
Stripe-centric `users` table.

## Target Architecture

```text
modules/user/
  domain/                 identity + role/plan enums
  adapter/gormrepo/       users schema + repo + migration
  adapter/authstore/      auth.UserStore + RoleResolver

modules/billing/
  adapter/repo/
    billing_event_gorm.go
    billing_customer_gorm.go
    billing_subscription_gorm.go
    user_resolver_gorm.go
  adapter/stripe/
  app/
  domain/
  port/

stacks/saascore/
  standard composition
  event listeners
  summary projection into users
```

## Target Table Ownership

### 1. `users`

Purpose:

- identity
- auth linkage
- lightweight app-facing summary

Recommended fields:

- `id`
- `email`
- `email_verified`
- `email_verified_at`
- `username`
- `avatar_url`
- `provider`
- `provider_subject`
- `role`
- `plan`
- `credits`
- `last_login_at`
- `created_at`
- `updated_at`

Rules:

- no `stripe_*` columns
- no provider lifecycle timestamps
- `plan` is an app-facing summary projection, not the source of truth
  for provider state
- `credits` is an app-facing balance summary; if several products later
  converge on the same wallet semantics, add a ledger and project into
  this field

Why keep `plan` and `credits` here:

- they are high-frequency reads for product gating
- they are common across many SaaS products even when the payment
  provider changes
- keeping them avoids unnecessary friction in the quickstart path

### 2. `billing_customers`

Purpose:

- map a platform user to a provider-side customer identity

Recommended fields:

- `id`
- `user_id`
- `provider`
- `provider_customer_id`
- `created_at`
- `updated_at`

Recommended constraints and indexes:

- unique `(provider, provider_customer_id)`
- unique `(user_id, provider)`
- index on `user_id`

Ownership rules:

- this is the source of truth for provider customer linkage
- `billing.port.CustomerStore.SaveCustomerID` should persist here
- `billing.port.UserResolver` should use this table before falling back
  to email

### 3. `billing_subscriptions`

Purpose:

- store the current provider-derived commercial snapshot for a user

Recommended fields:

- `id`
- `user_id`
- `provider`
- `provider_customer_id`
- `provider_subscription_id`
- `provider_price_id`
- `provider_product_id`
- `product_type`
- `plan`
- `billing_interval`
- `status`
- `cancel_at_period_end`
- `period_start`
- `period_end`
- `cancel_effective_at`
- `raw_snapshot_json`
- `created_at`
- `updated_at`

Recommended constraints and indexes:

- unique `(provider, provider_subscription_id)` when
  `provider_subscription_id` is non-empty
- index on `(user_id, provider)`
- index on `(user_id, status)`
- index on `provider_customer_id`

Ownership rules:

- this is the source of truth for commercial status derived from the
  payment provider
- recurring subscriptions live here naturally
- lifetime purchases also live here with:
  - `product_type = lifetime`
  - empty `provider_subscription_id`
  - empty `billing_interval`
- credits purchases do not need a row here unless the future platform
  chooses to model prepaid balances as a separate durable contract type

Why this table exists even though `billing_events` already exists:

- `billing_events` is audit + idempotency history
- `billing_subscriptions` is the current read model for a user's
  commercial state

### 4. `billing_events`

Purpose:

- webhook audit trail
- idempotency

Status:

- keep the existing table and repo
- the current domain already exposes `ProviderEventID`
- the physical `stripe_event_id` column name can stay for backward
  compatibility in the first migration wave

Future cleanup:

- after the new billing-owned tables are stable, consider renaming the
  physical column to `provider_event_id` in a later compatibility window

### 5. `credit_ledger` or `account_ledger`

Status:

- not part of the first boundary refactor
- add only if several products truly need auditable shared balance
  semantics

Until then:

- `users.credits` remains sufficient for the standard quickstart path

## Source of Truth Rules

The platform must be explicit about data ownership:

- user identity truth lives in `users`
- provider customer truth lives in `billing_customers`
- provider subscription/commercial truth lives in
  `billing_subscriptions`
- webhook audit and idempotency truth lives in `billing_events`
- app-facing fast-read entitlement summary lives in `users.plan` and
  `users.credits`

Important consequence:

- `users.plan` may be derived from billing events
- `users.plan` must not be used to reconstruct provider state
- `billing_subscriptions` is authoritative when provider state and user
  summary disagree

## Module Responsibilities After the Refactor

### `modules/user`

Owns:

- user entity
- shared `users` schema
- normalization of email, role, auth provider, and plan summary
- auth bridge adapters

Does not own:

- provider customer IDs
- provider subscription IDs
- billing lifecycle timestamps
- provider price/product snapshots

Practical consequence:

- `modules/user/adapter/billingstore` should be deprecated
- billing-related projection helpers should move out of `modules/user`

### `modules/billing`

Owns:

- payment provider adapters
- billing event persistence
- provider linkage persistence
- current commercial snapshot persistence
- billing-side user resolution

Should expose:

- GORM repos for `billing_customers` and `billing_subscriptions`
- a `UserResolver` implementation backed by billing-owned tables
- helper methods for writing/updating subscription snapshots

### `stacks/saascore`

Owns:

- the default multi-module wiring
- standard event subscriptions
- summary projection from billing state into `users.plan`
- credits projection into `users.credits`

Should not own:

- hidden schema shortcuts that bypass module boundaries

### `templates/quickstart`

Owns:

- runnable host shell
- config loading
- app-specific hooks

Should not own:

- the canonical user/billing schema contract

## Port and Adapter Strategy

The current `billing` ports are already close to the right level.

Keep:

- `billing.port.CustomerStore`
- `billing.port.UserResolver`

Change the default implementation:

- today they are implemented against `modules/user`
- target state is to implement them in `modules/billing/adapter/repo`
  against `billing_customers`, `billing_subscriptions`, and `users`

This avoids a public API break while still fixing the ownership boundary.

Recommended new adapter files:

- `modules/billing/adapter/repo/billing_customer_gorm.go`
- `modules/billing/adapter/repo/billing_subscription_gorm.go`
- `modules/billing/adapter/repo/user_resolver_gorm.go`

## Migration Plan

The migration should be additive first, destructive later.

### Phase 1: add billing-owned tables and repos

- add `billing_customers`
- add `billing_subscriptions`
- add default GORM repos under `modules/billing/adapter/repo`
- keep all existing `users` billing fields intact for compatibility

Result:

- no host app breaks
- new code paths can be introduced gradually

### Phase 2: dual-write from `saascore`

- on checkout/customer creation, persist customer linkage to
  `billing_customers`
- on subscription/lifetime events, update `billing_subscriptions`
- continue updating legacy `users` billing fields during this phase
- continue projecting `plan` and `credits` into `users`

Result:

- new and old read paths both stay valid
- data can be backfilled and compared safely

### Phase 3: switch read paths

- `CustomerStore.LoadCustomer` reads from `billing_customers`
- `UserResolver.Resolve` reads from `billing_customers` and
  `billing_subscriptions`
- `QueryService` treats `billing_subscriptions` as the local commercial
  snapshot and still asks the provider for richer live data where needed
- legacy `users` billing fields become fallback-only

Result:

- the effective architecture is corrected without yet dropping columns

### Phase 4: deprecate legacy user billing fields

- mark user Stripe/billing columns deprecated in docs and changelogs
- stop writing legacy fields by default in `saascore`
- keep one compatibility release where fallback reads remain possible

### Phase 5: remove legacy fields in a breaking release

- remove the deprecated fields from `modules/user`
- delete `modules/user/adapter/billingstore`
- drop obsolete columns from the standard quickstart migration path
- optionally rename `billing_events.stripe_event_id` to
  `provider_event_id`

## Changes Required in This Repo

High-confidence code changes:

- shrink `modules/user/domain/user.go`
- shrink `modules/user/adapter/gormrepo/model.go`
- deprecate and later remove `modules/user/adapter/billingstore`
- add billing-owned GORM models and repos
- update `stacks/saascore/saascore.go` to wire billing repos from
  `modules/billing`, not from `modules/user`
- update `docs/SAASCORE_GUIDE.md`
- update `modules/user/README.md`
- update `stacks/saascore/README.md`

Likely transitional helpers:

- backfill script from legacy `users.stripe_*` fields into
  `billing_customers` and `billing_subscriptions`
- consistency check comparing the legacy projection and the new billing
  state rows

## Risks and Mitigations

Risk: dual-write divergence during migration

Mitigation:

- treat `billing_subscriptions` as the new target state
- add temporary comparison checks in tests and migration tooling

Risk: existing hosts depend directly on `users.stripe_*`

Mitigation:

- keep a deprecation phase
- document fallback duration explicitly

Risk: over-engineering the platform before the package consumer base is
stable

Mitigation:

- do not introduce a separate entitlement module yet
- keep `plan` and `credits` in `users` for the first cleanup

## Acceptance Criteria

This design is successful when:

1. a new SaaS project can still bootstrap with `stacks/saascore` and the
   quickstart template with minimal glue
2. `modules/user` no longer needs Stripe-specific fields to be useful
3. `modules/billing` can support another provider without polluting the
   standard `users` schema
4. `users.plan` remains easy to read for feature gating
5. existing projects can migrate without a flag day cutover

## Final Recommendation

For this repo's actual product position, the right professional standard
is not:

- "keep everything in `users` forever"

and not:

- "remove every commercial summary from `users` immediately"

The right standard is:

- `users` stays identity-first and stable
- `billing` owns provider state
- `saascore` projects billing outcomes back into simple user summaries

That is the cleanest boundary that still respects the repo's real job as
the fast-start shared foundation for many similar SaaS products.
