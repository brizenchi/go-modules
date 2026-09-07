# Template Priority Fixes Implementation Plan

**Goal:** Fix configured token lifetimes, make logout revoke its bearer token, restore detached build verification, and expose verified Stripe refund/dispute hooks without implementing SaaS policy.

**Architecture:** Keep the current JWT authentication API and add an optional persistent revocation store backed by the existing authentication database. Store token hashes rather than bearer credentials and fail closed when the store is unavailable. Publish typed payment lifecycle events through the existing billing event bus; template-owned hooks remain no-ops until a SaaS supplies its policy.

**Tech Stack:** Go, Gin, GORM, PostgreSQL/SQLite tests, JWT, Stripe webhooks, Next.js.

---

## Scope

- Do not change rate limiting, database health checks, production safety checks, or business entitlement rules.
- Do not execute migrations against configured databases, publish tags, push, or deploy.
- Logout revokes the submitted access token, not every device or independently issued session.
- No Redis dependency or server-side storage of raw bearer tokens.
- New shared-module interfaces require a release before a detached template can resolve them from GitHub. Verify unpublished changes with the existing detached-copy/local-replace flow; document the release gate explicitly rather than pretending the old version contains the new API.

## Task 1: Token lifetimes

Files: `modules/auth/module.go`, `templates/quickstart/internal/platform/auth_provider.go`, new module wiring tests and platform tests.

1. Add regression coverage for login, OAuth completion, refresh and WebSocket ticket lifetimes.
2. Add token and ticket lifetimes to module dependencies and pass them to all use cases.
3. Wire the template configuration into those dependencies; preserve defaults for consumers that omit them.
4. Run focused auth and platform tests.

## Task 2: Persistent logout revocation

Files: auth ports/domain/JWT/GORM/session/HTTP packages, platform authentication wiring, frontend logout callers, migration and documentation.

1. Define optional context-aware token verification/revocation capabilities without breaking existing TokenSigner implementations.
2. Add a GORM revocation table with token hash and expiry; make insertion idempotent and prune expired entries on logout.
3. Add unique JWT IDs, verify revocation after cryptographic validation, and revoke only valid access tokens.
4. Thread request contexts through verification and logout; return unavailable rather than success when persistence fails.
5. Wire persistence into the template and ensure frontend logout calls the API before removing local credentials.
6. Test old-token rejection on protected routes and refresh, independent sessions, invalid tokens, expiry, repeat revocation, database failure and persistence across adapter reconstruction.
7. Add an explicit SQL migration for existing installations; do not apply it remotely.

## Task 3: Stripe lifecycle hooks

Files: billing event definitions, Stripe event translation, webhook tests, template subscriptions and host hooks.

1. Add typed refund and dispute lifecycle events carrying provider identifiers, status, amount/currency and provenance.
2. Translate verified Stripe payloads without making additional provider calls or changing balances/subscriptions.
3. Bind empty template business hooks through existing retryable event listeners.
4. Cover partial/cumulative refunds, expanded/string IDs, lifecycle statuses, signature rejection, duplicates and retry after hook failure.
5. Document idempotency, ordering, missing user mapping and the SaaS owner's responsibility for entitlement decisions.

## Task 4: Standalone dependencies and release checks

Files: `templates/quickstart/go.mod`, `templates/quickstart/go.sum`, release verification script and deployment documentation.

1. Generate tidy module files in a temporary detached copy using the current local shared module.
2. Apply dependency/checksum changes without committing a local filesystem replace directive.
3. Verify the template outside go.work, including the Linux CGO-disabled Docker build command.
4. Keep the public release version check explicit: publish the shared changes and run `make pin-template-version VERSION=<released-version>` before building against that release.

## Final verification

- Focused regressions before and after implementation.
- Shared auth/billing suites and complete template Go tests.
- Frontend tests, lint, production build if frontend changes.
- Detached-copy verification and workspace Linux build.
- Formatting, diff checks, clean separation from user changes.
- Report any release-dependent validation honestly; never modify production state.

## Implementation notes

- Implemented module TTL propagation, persistent per-token logout, fail-closed verification and frontend retry behavior.
- JWT parsing rejects noncanonical signature encodings and embedded newlines so alternate encodings cannot evade hash-based revocation.
- Added separate individual-refund, cumulative-charge-refund and dispute events with no-op host hooks; existing business policies remain untouched.
- Added the missing AWS indirect requirements and checksums, without committing a local replace.
- Fixed both template-copy scripts: basename tar exclusions also removed `cmd/quickstart`, so root entries are now filtered explicitly.
- CI verifies current monorepo code plus a detached local-replace copy with a readonly Linux build before tidy. Published-version verification remains a separate release gate.
- Full shared Go tests and template Go tests pass; frontend 97 tests, lint and production build pass; focused vet passes.
- The currently pinned public module predates both this change and other existing template APIs. A published-version build fails on missing APIs (not AWS checksums); publish and pin a compatible release before distributing a detached template. Monorepo deployment does not need that release step.
- No configured database was contacted or migrated; no commit, tag, push or deployment was performed. Concurrent website/guide edits were preserved.
