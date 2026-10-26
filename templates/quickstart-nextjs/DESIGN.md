# Frontend design conventions

The starter has three surfaces: a public product website, a personal account
center and an independent administrator console. They share primitives and
tokens; navigation and information density follow the task in each surface.

## Foundations

Use the CSS variables in `app/globals.css`. Extend semantic tokens rather than
introducing another page-specific palette. Warm neutral surfaces, dark text and
green brand accents remain the default identity. Success, warning, error and
information messages have their own semantic colors and include text labels.

| Element | Default |
| --- | --- |
| Account title | 32px desktop / 28px mobile, weight 650 |
| Section title | 20px |
| Body and forms | 14–16px |
| Auxiliary text | 12–13px |
| Spacing | 4px base; prefer 8, 12, 16, 20, 24, 32px |
| Control / card / dialog radius | 8 / 12 / 16px |
| Main controls | 44px minimum height; mobile inputs 16px text |
| Account content | Maximum 1200px including horizontal padding |

Use Manrope for UI with explicit Chinese fallbacks. Chinese headings use natural
letter spacing and a 1.35 line height. Prices and data use tabular numbers. Large
display typography belongs to public marketing surfaces, not account forms.

## Layout and interaction

- Customer navigation remains account, subscription and referrals. Credit history remains an auxiliary account service. Do not add product-specific
  note, upload or chat pages to the default customer area; admin has its own navigation.
- Each page has one main heading, short supporting copy and a clear next action.
- Tables/lists represent comparable records; cards group distinct tasks or key
  account states. Keep table overflow within its own container on small screens.
- Use `Panel`, `Notice`, `EmptyState`, `LoadingState` and `ResourceFailure` for
  consistent hierarchy and feedback. Place action feedback beside its controls.
- Preserve visible focus, native labels, keyboard disclosure behavior and the
  reduced-motion preference. Keep locale controls reachable at mobile widths.
- Ordinary text should meet WCAG AA contrast (4.5:1). The 44px control height is
  this project's touch comfort target, not a claim about WCAG AA's minimum size.

## Subscription surfaces

Public prices are examples for a free starter. Monthly/yearly offers share a
selector; lifetime access and credit packages are one-time products. Annual
examples show the annual total, not only an implied monthly amount.

`billingIntentHref` and `readBillingIntent` carry a selection to billing.
`isBillingIntentAvailable` validates it against capabilities before purchasing.
Never silently replace a linked, unavailable product with a different purchase.
Signing in inside the customer area, including through OAuth, retains the
validated customer path, purchase query parameters and anchor. A return path is
a navigation hint, not authorization; reject destinations outside the allowed
customer routes and keep administrator sign-in separate.

`/billing` remains one dedicated page; do not add a second subscription-management
page. Keep the current subscription/access panel and pending payment or scheduled
changes visible. Plans, credit purchasing, and purchase/invoice history use
separate native `details`/`summary` groups that users can expand when needed.

An incoming purchase intent opens its matching plans or credits group. Without
an incoming intent, open plans only after subscription data reaches `ready` and
confirms no ongoing recurring subscription or lifetime access. Keep loading,
unavailable and failed subscription states distinct from that confirmed state.
Supported section anchors open the relevant group, and action feedback keeps its
own group visible. Opening a group or returning from sign-in never starts payment.

Amount and timing precede plan-change confirmation. Cancellation explains the
remaining access period before action. Only backend/checkout data is authoritative
for money and entitlements. Local preview and automated frontend checks validate
interface behavior; they do not verify live payments or production checkout.

## References

- [Geist colors](https://vercel.com/geist/colors) and [typography](https://vercel.com/geist/typography)
- [Stripe customer portal](https://docs.stripe.com/customer-management)
- [WCAG contrast](https://www.w3.org/WAI/WCAG22/Understanding/contrast-minimum.html)

## Public content architecture

Public navigation defaults to features and use cases; administrators can add other sections. Feature pages explain capabilities; use cases connect them to a task; examples show actual input and output; comparisons state tradeoffs. Resources collect practical articles and instructions. Keep account and billing management in the customer workspace.

Use the marketing index and detail templates for consistent spacing, facts, steps, boundaries, FAQ, and source links. Use short category labels in navigation and descriptive titles in page content. Avoid invented social proof. English uses original URLs and Chinese uses `/zh`; public links preserve language, with crawlable language alternatives. Account pages remain unindexed.

Language selection uses one native disclosure with the current language and a globe icon. List language names in their own language, mark the active choice, and keep public alternatives as links in initial HTML. Both public links and private choices save the selected language. English is the default; browser/OS language is never used as an automatic fallback. Public URL language stays authoritative for SEO and carries into the account/admin area. Private refreshes read the same saved preference, and storage events synchronize other tabs. The list scrolls when it grows; only expose languages with published translations and corresponding locale routes. Escape, focus leaving the disclosure, and outside clicks close the list.

New purchases belong on the public pricing page: select billing interval, choose a plan, sign in if needed, then continue directly to Stripe. State recurring billing, full yearly charge, cancellation, and one-time product rules beneath the offers. Do not route new customers through the subscription management form. Existing subscriptions use the workspace for reviewed changes and invoices.

### Customer account interactions

The account menu shows one read-only identity summary, account/billing/referral links and direct sign-out. The account page also offers direct sign-out; do not make customers open a duplicate information dialog. Do not expose avatar URLs, profile editing, session refresh, or WebSocket tickets in the customer interface. `/account` contains common account services: subscriptions, credits and referrals. Notes and uploads have no default frontend pages; `/examples` links directly to the three common customer services. Subscription, invoices, payment methods, and referral history retain dedicated pages; new purchases start on `/pricing`.

## Customer UI finish

Use a neutral sidebar, restrained green selection, compact 28px workspace titles, and consistent 44px menu targets. Account triggers show initials and one identity label; the email belongs in the disclosure and account summary. Avoid repeating identity cards in the sidebar. Menu arrows/Home/End navigate options; Tab remains native and Escape returns focus. Use shared line icons and semantic colors. Keep pricing heading and section spacing compact enough to expose the purchasing controls earlier. Preserve existing catalog, checkout, entitlement, and demonstration behavior.

## Responsive customer surfaces

`CustomerShell` owns the customer account layout: a shared brand header, contained settings navigation on desktop, and a left navigation drawer below 761px. It replaces the fixed workspace sidebar; the administrator console remains separate. Page content retains the workspace content primitives.

`ResponsiveDialog` is the common modal primitive for sign-in, administrator operations and mobile navigation. Native dialogs provide modal semantics; shared code handles focus boundaries, Escape, backdrop dismissal, focus restoration, and body scroll locking. Sheet dialogs become bottom sheets below 641px. Closed dialogs do not mount their interactive content or load authentication capabilities.

The shared sign-in/register dialog uses the neutral title “Sign in”. Authentication method discovery has a one-minute, tab-local memory cache and shares in-flight requests, so reopening a warm dialog immediately shows its methods. Expiry, page reload or an explicit error retry refreshes discovery. Billing capabilities and OAuth authorization URLs are never cached by this helper; OAuth starts only after the customer selects a provider.

Pricing keeps a global request lock but renders progress and errors only in the selected product card. Other product labels remain stable. Monetary values, OAuth handling, available methods, checkout payloads, and demonstration behavior are unchanged.

## Extension UI primitives

Use `DataTable<T>` for repeated records and server pagination; include a caption, loading, empty, and error states. Keep horizontal overflow within the table region. Use `FormField` to associate one native control with its label, description, and inline error; use `FormActions` for pending submissions. Credit history uses the shared table. Product features can use the form primitives when they add their own workflows.

## Public homepage

### Public content discovery

The header and footer are configured in Admin → Site settings → Navigation & footer; by default the header has Features and Use cases, and the brand mark is the home link. Product archives and product details offer visible peer navigation to Features, Use cases, Comparisons and the demo. Standalone product/support pages do not select an unrelated primary link. Breadcrumbs retain their shallow category hierarchy; footer group headings are not artificial parent pages.

The homepage links directly to published use cases, recent guides, demo, updates and project information. Resources opens with crawlable documentation/article/update directory links; its existing category buttons only filter the list. Details recommend at most three published entries, prioritizing explicit references, backlinks and same-category pages. The server provides card metadata for the current language; article bodies are not included in the discovery payload. Empty managed catalogs stay empty, and service failures do not resurrect starter articles. Only unconfigured/legacy APIs use the static catalog.

The footer uses the shared brand mark, the configured columns (by default Product / Resources / About & support) and a separate legal row. Public build acceptance verifies directory anchors and that all published pages can be reached from the same-language homepage within three clicks through the header and main content, without relying on the global footer. This is an information architecture check, not a ranking guarantee.

The homepage leads with one short value statement and a real, locale-specific account-center screenshot. Keep screenshot accounts and content fictional and label previews. The website/workspace/admin directory sits below the hero. Database-owned page copy explains workflows and boundaries; do not invent customer counts, testimonials or revenue. With an API configured, editorial home copy, section introductions and articles come exclusively from published CMS snapshots. Feature/scenario cards use published record metadata. Website settings supply brand identity and footer introduction; homepage SEO is not overwritten by legacy settings. `content/site.ts` supplies visual assets and unconfigured offline-preview defaults. The generated sharing artwork remains build-time.

Optional analytics and support UI only appear with valid configured provider IDs. AI is a server-side stateless integration, without a default chat page, chat API, conversation storage or charging policy. Public copy explains that products own these features.


### Account navigation

Customer routes live in the pathless `(private)/(customer)` group. Its shared layout owns `CustomerArea` and the account header/navigation; `SiteShell` supplies only the customer page heading, actions and body. Keep customer loading boundaries inside this group so administrator and sign-in layouts remain independent. Route paths do not include the group name.

Navigation uses Next.js `useLinkStatus` for a small pending indicator and accessible destination announcement. Keep normal link prefetching and modifier-key behavior. Do not simulate route progress with a timer or update `aria-current` before the destination commits. The mobile drawer stays available while navigation waits, then closes on commit; selecting the current destination closes it immediately. Respect reduced motion.


## Administrator interface

The administrator console uses the same brand mark, white surfaces, neutral borders, typography and semantic colors as the personal account area. Keep a wider 1440px frame for operational tables, a compact 72px header, and 28px page titles (24px on mobile). A grouped desktop sidebar becomes a navigation drawer below 901px. Administrator identity is read-only in a small disclosure with website and sign-out actions.

`lib/admin-navigation.ts` owns section names, descriptions, icons and three navigation groups: Workspace (overview/users), Business records (payments/subscriptions/referrals/credit history), and System. `AdminPageHeader` keeps page hierarchy and refresh feedback consistent. Router-owned pending labels provide feedback until navigation commits; loading replaces only the page content. Keep authorization in `AdminShell` and the API.

Use `DataTable` for lists: labeled search controls, a record count, scoped overflow, loading/empty/error states and server pagination. Right-align numeric columns. Display account email with a secondary ID where the API provides both; keep identifiers available in full. Payment amounts remain in minor units and the UI must state that unit.

Use `StatusBadge` with `adminTag` to label enums within their section and field. A pending payment, invitation and audit action have different meanings. Never translate arbitrary reasons, IDs or unknown states into a different business outcome. Include text in every status; color alone is insufficient.

Credit grants, consumption returns and referral reconciliation use `ResponsiveDialog` for focused operations. Keep reasons, validation, idempotency, session ownership and existing API behavior intact. Website settings groups public identity, support contacts and non-secret sender details. Provider status is read-only; credentials belong to deployment configuration. Configured credentials do not imply a tested connection.


## Customer area scope

Keep exactly three primary destinations: account, subscription/billing and referrals. Credit history remains an auxiliary account page. Product-specific content, file and AI interfaces belong to the product built from this template. The frontend has no default notes/export or file-upload page. Retired `/notes` and `/files` addresses redirect to `/account`; their old public example articles redirect to the localized `/examples` directory, which links directly to the three common services. Preserve the shared infrastructure, backend reference code and existing data.


Operational lists use a compact toolbar with accessible labels, search, filters, reset and count. The toolbar sticks beneath the admin header; rows scroll within a bounded region with sticky column headings. Pagination stays outside row scrolling. Keep customer email primary, truncate auxiliary IDs visually while preserving their full title, and align numeric values to the right.

The user list shows membership level rather than security role. Membership and credit adjustments open `OperatorUserDialog`, retain list filters/page, use stable retry keys and refresh after save. Manual access is distinct from Stripe charges. The overview uses real 7/30-day data, an explicit UTC+8/UTC reporting zone and live/test payment selector; never infer payments from webhook counts or trial subscriptions.

Public navigation defaults to Features / Use cases (功能介绍 / 使用场景), followed by language and the account menu. Administrators choose header sections, their order, names and menus (published pages listed automatically, chosen in order, or none) and the footer columns; `content/navigation.ts` shares labels and destinations with footers, breadcrumbs, index headings and metadata. The Guides label uses the existing `/resources` route. Features is a six-capability overview; use cases remain secondary. Account navigation stays at three services, with Help & support / Privacy / Terms in the compact footer on all screen sizes. `/contact` is customer self-service plus real configured support channels; operator setup guides belong in documentation.

Each header item is a real link to its section. A menu is one level of pages, revealed on hover or keyboard focus: plain links in the initial HTML, no buttons, scripts or `aria-expanded`. Entries come from pages published in the current language (`navigationIndex`), so the header never links to an unpublished page, and a section without published pages is hidden; small screens hide menus and arrows and rely on the directory pages. Give each link its own complete rounded hover/active target, separated by at least 8px. The footer links sections only (by default Product, Resources, About & support), without repeating the group name as a link; privacy/terms belong to its bottom row. Guides show documentation/articles only, category filters and a compact search, with three/two/one-column cards. Product capability/use-case/comparison pages retain their own destinations. Language selection uses a checkmark and stronger text; hover uses a separate fill with gaps between rows.

The five primary administration destinations are Overview, Users, Articles, Website settings and Audit log. Global payments and invitations are tabs inside Users. The customer dialog has exactly three tabs: Overview (registration time, effective membership, credits, compact subscription facts and explicit edit actions), Payment records, and Credit changes. Records are read-only; referral rewards appear in the shared credit ledger. Use distinct editor/history reconciliation keys to prevent retained forms after tab switches. Legacy section URLs redirect to their new home. Articles use a dedicated bilingual editor with draft/live separation and explicit per-language publication. Filters stay fixed above the bounded scrolling records; horizontal scrolling remains inside tables.

`Select` preserves native keyboard, option and label behavior with shared height, padding, chevron, focus and disabled states. Use it for administrator report filters, list filters and editors. Payment history shows created checkouts and verified outcomes; creation does not grant access or count as a payment. The ledger displays existing source/reason/actor/reference data; supporting a source label does not introduce an automatic reward policy.

Published content archives own their list/search styles in `published-content.module.css`. Keep the title and summary in one flexible column, with the reading link beside it on desktop and below it on mobile. Do not reuse the marketing index's numbered three-column grid: a lone content child would occupy its 48px number track. Offline Chrome layout acceptance in `scripts/check-published-layout.mjs` checks all twelve EN/ZH CMS archive pages at actual 1440/768/390px CSS viewports, including heading/summary widths and horizontal overflow.
