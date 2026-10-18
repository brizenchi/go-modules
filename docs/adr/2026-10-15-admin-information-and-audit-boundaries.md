# Administrator information architecture and shared audit boundary

Status: Implemented locally; not committed or deployed.

## Context

The operator console currently exposes implementation-oriented sections: users,
subscriptions, orders, referrals, credits, integrations and audit. The intended
operator wants to understand growth, manage individual customers, publish
localized content and configure the website with few navigation destinations.
The target is a practical starter that can launch quickly, not a general CMS or
an independently deployed audit platform.

Before these changes, source inspection found:
- Overview and membership/credit modals were added in the previous local change.
- Provider subscriptions, payments and credits are separate sources of truth;
  UI consolidation must not replace their business services with profile fields.
- Articles/docs/marketing detail pages are code-owned, with statically enumerated
  public routes. There is no persisted article publishing workflow.
- Runtime `SiteSettings.BrandName` / `Description` do not currently drive the
  public brand/homepage/metadata, which use build-time frontend configuration.
- Integration APIs return configured booleans for stored secrets, not their
  values. The frontend nevertheless exposes credential replacement forms.
- `operations.AuditEvent` and `claimAudit` exist, but business handlers construct
  events individually. Credit grants/refunds are recorded in their own ledger,
  not uniformly represented in the operator audit stream.

## Decisions

### Five main destinations

1. Overview
2. Customers
3. Content
4. Website settings
5. Audit log

Customers has Users / Payments / Invitations views. There are no independent
subscription, credit or service-integration sidebar entries. A user dialog has
Membership & subscription / Credit history / Payments / Invitations tabs. Keep
long article editing in a dedicated editor, with preview, rather than a modal.

Use the same bounded table/filter primitives across global and user-scoped
views. Preserve old deep links with redirects or compatibility views.

### Customer identity and payments

Display effective membership (Free/Starter/Pro/Premium/Lifetime), not the security
role. Keep administrator authorization separate. A manual membership changes site
access; it must not overwrite the provider subscription or silently cancel charges.

Keep a global payment query inside Customers for reconciliation. Suggested
columns: payment time, customer, purchased product, paid amount/currency, payment
state, reference and details. Filter by date, environment, status and customer.
A user-specific payment view uses an exact server-side user filter and the same
projection, never client filtering over one page of global payments. Verified
refund data is needed before claiming a refunded status; do not infer it from
access changes. Credit adjustments continue through the transactional credit
service/ledger.

### Lightweight localized publishing

Manage blog/resource articles, guides and feature/use-case detail content.
Specialized homepage/pricing/account layouts remain dedicated components.

Use a stable content identifier/path/placement plus locale-specific title,
summary, safe Markdown body, SEO title/description and publication state.
Support Markdown import, draft, preview, publish/unpublish, ordering and separate
translation states. A language can publish only when its own required content is
complete. No automatic claim that a translation exists. Public HTML, metadata,
canonical/hreflang and sitemap must use the same published revision. Existing
content needs an explicit import/source policy, avoiding divergent editor and
code copies. Public HTML, indexes, metadata and sitemap use 30-second fetch/page revalidation. Subsequent requests normally reflect publication within about a minute; this version does not promise immediate cache invalidation.

Use plain Markdown with HTML disabled/sanitized; no executable MDX or uploaded
scripts. An editor/CMS database is sufficient for v1; do not add a separate CMS
service or page builder without demonstrated need.

### Website settings and provider credentials

Website settings contains public identity, bilingual homepage copy, support contacts and non-secret mail configuration. Supported locales remain the template's code-owned English and Chinese; this version has no arbitrary-locale configuration UI. Make public identity the
shared source for visible brand/homepage introduction and SEO; preserve build-time
values as documented defaults.

Remove service integrations as a primary destination. Show a read-only payment
service status and actual configuration environment. Keep Stripe secret keys,
webhook secrets and mail API credentials in deployment configuration; the UI
should not retrieve, reveal or replace them. A configured badge is not proof of
successful connectivity.

Hiding forms alone is insufficient: the corresponding write endpoints/policy
must enforce the boundary. Existing database provider overrides require an
explicit compatibility/migration plan before closing those writes; never delete
or switch production credentials automatically.

Compatibility path (2026-10-18): deployments can explicitly set
`APP_BILLING_STRIPE_CONFIG_SOURCE=environment` to use their complete Stripe
configuration after a backend restart. An unset switch preserves legacy database
precedence. The switch does not write/delete the legacy row or change Resend's
source. Provision the Stripe keys, mode, prices and enablement together before
opting in; see [configuration migration](../CONFIG_STANDARD.md#配置优先级).

### One application audit service

Introduce a reusable `audit.Service` boundary within the application, injected by
the host composition root. No separate process, RPC dependency or message broker
is needed initially. It can be extracted into a versioned shared module when its
contract stabilizes. Reusable business modules remain independent and do not
import each other; the host injects the audit port or subscribes to their events.

Domain services submit structured, allowlisted events: actor, action, target,
allowlisted changes, reason, UTC time and outcome. Context carries request correlation into the database and logger; audit rows do not add manual trace fields. Include
membership/credit changes, article publication, website configuration and
sensitive administration/security actions. Ordinary search/pagination is not a
business change. Never store credentials, tokens or raw request bodies.

Financial/authorization mutations and their successful audit record share a
transaction, or use a durable outbox if they cross boundaries. Audit failure must
not silently drop a critical success record. Record relevant failed attempts
without masking the primary outcome. Keep append-only audit events separate from
mutable idempotency/reservation state; retries do not duplicate business success.
Administrator audit UI is query-only and supports filters and detail expansion.

## Trade-offs and alternatives

- Retaining all current sidebar sections is cheap but preserves navigation overlap.
- Folding all payments into individual users hides the global reconciliation view;
  a Payments tab within Customers keeps both workflows without another menu.
- A full external CMS / independent audit microservice introduces deployment and
  consistency work disproportionate to the 80-point launch target.
- Dynamic publishing and truly editable public branding require backend and public
  rendering changes, so they cannot be claimed as frontend-only cleanup.

## Suggested implementation sequence

1. Shared audit service and adapters; establish one reliable recording contract.
2. Customer views and one dialog; remove redundant sidebar entries while keeping
   global payments and exact per-user history filters.
3. Website settings/source consistency and deployment-owned credential boundary.
4. Localized content persistence/editor/publication, then public route/SEO/cache
   integration. Validate language state, unpublished access and revision consistency.

## Implementation and operating limits

Implemented in the template composition layer: shared audit service and transactional credit callback, five-destination navigation, customer dialogs/global reconciliation tabs, bilingual Markdown publishing, runtime website presentation and read-only service status. Existing legacy section URLs and absent legacy content endpoints remain compatible. Explicit unpublication never falls back to seed content.

New tables/columns are registered through host/preview GORM models; no applied migration is edited. No configured database has been migrated by this work. All mutating verification uses disposable SQLite fixtures. No production service, payment provider configuration, commit, push or deployment is performed.

The frontend requires a Next.js server runtime for publishing/ISR. Sharing artwork and specialized layouts remain code-owned. Mail sender identity updates on backend restart. Legacy database credential overrides are preserved until an explicit deployment migration. Audit is centralized within the application, with no microservice dependency. Authentication failures before the admin group and provider webhook history remain their respective service concerns.

Validation results are recorded in `docs/plans/2026-10-15-admin-operations-design.md`.
