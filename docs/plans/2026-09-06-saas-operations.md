# SaaS Operations Implementation Plan

> Execute this plan task by task in the current workspace; preserve the existing, uncommitted user-console implementation that this work extends.

**Goal:** Deliver a usable operator console, credit lifecycle and paid export example, secure uploads, editable business settings, and bilingual public content/SEO for the free SaaS starter.

**Architecture:** Keep reusable auth/billing/referral modules intact. Add host-owned Go features and protected API routes, reuse the existing user/admin identity groups, extend the Next.js console with typed clients, and keep public content local and editable. Independent backend credit, backend operations, and public-content work runs alongside root-owned frontend/integration work.

**Tech Stack:** Go, Gin, GORM, PostgreSQL deployment / isolated SQLite tests, Next.js 15, React 18, TypeScript, CSS modules, Node tests.

## Accepted scope and constraints

- Clarification from the user: the customer workspace is for managing one's own account, subscription and invitations. All site-wide management belongs to the separate `/admin` route tree, with its own administrator login, layout and navigation. Customer navigation and the account menu must not contain administrator controls.

- Include users, subscriptions, payment records, referrals, credit grants/refunds, audit history and business settings in the operator console. Authorization lives on the server; client role checks only control navigation.
- Credit grants, expiry, consumption and refund require stable idempotency, ownership, insufficient-balance protection and an auditable reason. Existing balances are preserved through a supplied explicit migration, never an automatic repair.
- A note-to-Markdown export is the paid business example. Show its actual configured cost, deduct once per export request and let retries retrieve the same output without another charge.
- Image uploads validate actual file content and size and enforce ownership; production object-storage configuration is explicit.
- Public content includes bilingual docs/pricing/blog/updates/contact/privacy/terms, search and metadata/sitemap/robots. Retain existing locale routing and OAuth URLs.
- Defer DAU/retention dashboards, AI generation, additional payment providers and Cloudflare-specific hosting.
- Do not migrate or repair any configured database, send real email, execute live payments, publish, or push. Persist migration artifacts; use isolated test databases and local preview fixtures.

## Task 1 — Credit lifecycle

Files: internal/user credit models/service/tests; internal/feature/credits HTTP/tests; dedicated migration SQL.

1. Cover concurrent spend, expiry, retry conflicts and refunds with SQLite tests.
2. Preserve GrantCredits signature; add balance, paginated ledger and controlled admin writes.
3. Add note ownership checks and atomic paid exports with persisted outputs.
4. Supply versioned legacy migration, compatibility notes and exact API DTOs.

## Task 2 — Operations, settings and uploads

Files: internal/feature/operations; internal/hostcfg upload config; dedicated migration SQL/tests.

1. Implement bounded, searchable lists and honest payment projections from stored billing events.
2. Implement activated-referral reward replay with reason/audit; reject pending invitations.
3. Add allowlisted public business settings and administrative writes with validation.
4. Add bounded image upload/read APIs, real MIME validation and owner checks.
5. Cover ordinary-user denial, invalid updates, pagination and private-file access.

## Task 3 — Customer and operator interfaces

Files: lib/operations-api.ts, reusable session/resource controls, app/admin, app/credits, app/notes, app/files and CSS modules; shared navigation integration.

1. Build typed clients and request/session race guards.
2. Build a separate administrator navigation with searchable lists, pagination, explicit failures and deliberate grant/refund/reconciliation forms.
3. Build balance/ledger, note editor/export and owned-file upload interfaces.
4. Preserve existing user console, add appropriate links, and keep new user-facing strings bilingual.

## Task 4 — Public content and configuration

Files: public content routes/data, docs/pricing, site-settings helper, metadata, sitemap/robots, CONTENT.md.

1. Implement editable bilingual local articles, search and changelog.
2. Add configurable contact links and starter privacy/terms content.
3. Exclude private routes from discovery; generate metadata only for real public routes.
4. Connect saved public brand/support settings to public and console UI.

## Task 5 — Integrate and verify

Files: host_routes.go, host_migrate.go, env/config examples, README/runbook and focused integration tests.

1. Register new host features and schema models without executing configured-environment migrations.
2. Run Go tests including race checks, frontend tests, lint and production build.
3. Exercise user/admin journeys against isolated local fixtures; verify permissions, invitation replay, export idempotency and negative cases.
4. Inspect frontend navigation, loading/error states and layout in the browser.
5. Record precise completed validation, deployment prerequisites and any live-service checks not performed.

## Completed validation

- Implemented all five tasks; migrations remain supplied artifacts, not executed against a configured database.
- `go test -race ./templates/quickstart/... ./foundation/ossx/s3` passed; after review fixes, operations and HTTP middleware race tests passed again. Real JWT tests cover anonymous/ordinary-user denial, changed export prices, idempotent consumption, and agreement between profile and ledger.
- Frontend: 79 tests passed, TypeScript and lint passed, and the final production build generated all 34 pages successfully.
- Browser checks against `cmd/preview` temporary SQLite: administrator login, ordinary-user denial of `/admin`, administrator logout and account switching, separate administrator/customer layouts, three-item customer navigation, settings save, preserved user filter from users to credits, note export balance 50→49, article search/detail, contact empty state, and 320px administrator layout without horizontal page overflow.
- Administrator OAuth return paths are tied to the exact browser flow and checked against a local allowlist. Ordinary sign-in continues to `/account`; admin-origin sign-in returns to the corresponding admin section after successful administrator authentication.
- Image upload server validation, ownership, multipart client handling and private authenticated reads passed automated tests. Browser automatic file selection was blocked by the Chrome extension's file-URL permission; no personal file was uploaded.
- No configured database migration, external email, live payment, deployment or git push was performed. Local preview billing is disabled; provider integration needs the operator's own test environment.
