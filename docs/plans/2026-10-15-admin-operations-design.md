# Admin operations refresh

- Prioritize overview and users. Group referrals, payment records, subscriptions and credit history under customer operations; keep configuration and audit separate.
- Compact, consistent list controls. A bounded table region scrolls independently; filters/counts and pagination remain outside it, and column headings stick inside it. Controls use 40 px desktop / 44 px mobile heights.
- Overview: real registrations, attributed invitations, users, successful positive payments and historical paying users. Offer 7/30 day series, UTC+8 / UTC reporting and explicit live/test payment selection. A payment is deduplicated by provider invoice/checkout, not webhook count. Free invoices, unpaid checkout, missing/unknown mode and manual memberships are excluded. Paid users count only known current users; refunds do not erase historical conversion.
- Membership and administrator authorization remain distinct. Display effective membership in the user list. A modal can grant a timed Starter/Pro/Premium membership, Lifetime, Free, or revert to payment-derived membership. Manual overrides have a separate host table, audited and idempotent; never modify Stripe subscription or charge amounts. Expired overrides fall back to payment membership.
- Credit grants use the existing shared credit service and idempotency ledger, in a user modal. Keep transaction refunds in credit history rather than arbitrary balance rewrites.
- No new dependencies, production changes, commits, pushes or existing-service restarts. Disposable acceptance fixtures may be started and cleaned up. Complete the already-started support/navigation consistency work in the same local change.

Validation: focused offline handler/authorization/idempotency/time-boundary tests, frontend tests/lint/build/content checks, backend race/vet and required root checks. Browser runtime unavailable at the start; report that limitation honestly.


Earlier validation of the initial overview/user-modal change:
- Frontend `npm run verify`: 184 tests, lint, production build and 72 public-page content checks passed (isolated copy; the development `.next` output was not overwritten).
- Backend `go test -race ./...` and `go vet ./...` passed. Focused regressions cover administrator access, member/authorization separation, payment subscription preservation and automatic fallback, credit idempotency/overdraw, calendar boundaries and payment deduplication/environment separation.
- Root format, race, vet, purity and CI-version lint passed. The detached backend release-copy verification (tests/vet/native and Linux builds) also passed.
- Local HTTP checks returned 200 for EN/ZH features/help, the four account routes and admin routes; private noindex remained present. Shared navigation links and translated help/footer contents matched.
- Browser control was unavailable, so authenticated visual and mobile interaction acceptance remains unverified. The running local backend is older; new metrics/member operations require a local backend restart. Frontend authorization falls back to the old protected overview endpoint only on a 404, preserving existing admin access during a staggered update. No production change, service restart, SQL operation against a configured database, commit or push was performed.

## Completed five-destination administration (2026-10-15)

- Overview uses real reporting. Users contains global payment/invitation tabs and a single per-user membership/subscription/credits/payments/referrals dialog. Legacy sidebar routes redirect.
- Localized content supports Markdown import, draft/live separation, independent EN/ZH publication, optimistic revisions and safe rendering. Public pages, indexes, canonical/hreflang, language availability and sitemap use published snapshots. Later publications create routes without a rebuild. Cache refresh is request-driven, normally within about a minute.
- Website settings supplies runtime brand, bilingual homepage copy, support contacts and non-secret sender identity. Provider status is read-only; credential writes are rejected. Legacy stored overrides remain compatible. Sender identity updates on backend restart; sharing artwork remains code-owned.
- `internal/audit.Service` separates retry reservations from append-only outcomes. Membership, default credit grant/refund/deduction, website and article writes audit transactionally; failed authenticated admin writes have sanitized records. Authentication/webhook boundaries are documented separately.

Final local verification:
- Frontend `npm run verify` in an isolated copy: **191 tests**, lint, production build and **73 published-page checks** passed against a disposable backend, including an English-only CMS fixture. Legacy-backend verification also passed (188 tests and 72 pages before the last three metadata/fallback tests were added).
- Backend `go test -race ./...` and `go vet ./...` passed. The final operations race run additionally covers settings before/after/replay and exact payment customer/environment/date filters. Detached-copy tests/vet/native/Linux builds passed.
- Root format, race, purity, vet and CI-version lint passed; `git diff --check` passed. No dependency or module version changes.
- Disposable HTTP acceptance: published SSR/canonical, draft privacy, English-only hreflang/Chinese 404, routes published after build, runtime brand/homepage metadata, unpublication/ISR (33 seconds observed), sitemap and denied credential writes. Final built HTML confirms the visible heading, title/description, brand and absolute sharing image.
- Browser control remains unavailable: authenticated visual/mobile acceptance is **not verified**. All writes/AutoMigrate used disposable fixtures. No configured database, user-running service or production environment was modified. Existing local backends need a normal restart to load new endpoints/models. No commit, push or deployment.
