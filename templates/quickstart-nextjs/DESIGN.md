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

Public navigation groups features, use cases, pricing, and resources. Feature pages explain capabilities; use cases connect them to a task; examples show actual input and output; comparisons state tradeoffs. Guides remain the home of setup instructions. Keep account and billing management in the customer workspace.

Use the marketing index and detail templates for consistent spacing, facts, steps, boundaries, FAQ, and source links. Use short category labels in navigation and descriptive titles in page content. Avoid invented social proof. English uses original URLs and Chinese uses `/zh`; public links preserve language, with crawlable language alternatives. Account pages remain unindexed.

Language selection uses one native disclosure with the current language and a globe icon. List language names in their own language, mark the active choice, and keep public alternatives as links in initial HTML. Private workspace choices update the saved preference. The list scrolls when it grows; only expose languages with published translations and corresponding locale routes. Escape, focus leaving the disclosure, and outside clicks close the list.

New purchases belong on the public pricing page: select billing interval, choose a plan, sign in if needed, then continue directly to Stripe. State recurring billing, full yearly charge, cancellation, and one-time product rules beneath the offers. Do not route new customers through the subscription management form. Existing subscriptions use the workspace for reviewed changes and invoices.

### Customer account interactions

Account information opens in a compact native dialog from the account menu or workspace overview. Show the sign-in email as read-only and offer sign-out; do not expose avatar URLs, profile editing, session refresh, or WebSocket tickets in the customer interface. `/account` contains common account services: subscriptions, credits and referrals. Notes and uploads have no default frontend pages; `/examples` links directly to the three common customer services. Subscription, invoices, payment methods, and referral history retain dedicated pages; new purchases start on `/pricing`.

## Customer UI finish

Use a neutral sidebar, restrained green selection, compact 28px workspace titles, and consistent 44px menu targets. Account triggers show initials and one identity label; the email belongs in the disclosure and read-only account dialog. Avoid repeating identity cards in the sidebar. Menu arrows/Home/End navigate options; Tab remains native and Escape returns focus. Use shared line icons and semantic colors. Keep pricing heading and section spacing compact enough to expose the purchasing controls earlier. Preserve existing catalog, checkout, entitlement, and demonstration behavior.

## Responsive customer surfaces

`CustomerShell` owns the customer account layout: a shared brand header, contained settings navigation on desktop, and a left navigation drawer below 761px. It replaces the fixed workspace sidebar; the administrator console remains separate. Page content retains the workspace content primitives.

`ResponsiveDialog` is the common modal primitive for sign-in, read-only account information, and navigation. Native dialogs provide modal semantics; shared code handles focus boundaries, Escape, backdrop dismissal, focus restoration, and body scroll locking. Sign-in and account dialogs become bottom sheets below 641px. Closed dialogs do not mount their interactive content or load authentication capabilities.

Pricing keeps a global request lock but renders progress and errors only in the selected product card. Other product labels remain stable. Monetary values, OAuth handling, available methods, checkout payloads, and demonstration behavior are unchanged.

## Extension UI primitives

Use `DataTable<T>` for repeated records and server pagination; include a caption, loading, empty, and error states. Keep horizontal overflow within the table region. Use `FormField` to associate one native control with its label, description, and inline error; use `FormActions` for pending submissions. Credit history uses the shared table. Product features can use the form primitives when they add their own workflows.

## Public homepage

The homepage leads with one short value statement and a real, locale-specific account-center screenshot. Keep screenshot accounts and content fictional and label previews. The website/workspace/admin directory sits below the hero. Native FAQ disclosures answer practical setup questions; do not invent customer counts, testimonials or revenue. Main presentation defaults live in `content/site.ts`. Brand overrides are build-time only so navigation, initial HTML and sharing metadata stay consistent; runtime support configuration remains separate.

Optional analytics and support UI only appear with valid configured provider IDs. AI is a server-side stateless integration, without a default chat page, chat API, conversation storage or charging policy. Public copy explains that products own these features.


### Account navigation

Customer routes live in the pathless `(private)/(customer)` group. Its shared layout owns `CustomerArea` and the account header/navigation; `SiteShell` supplies only the customer page heading, actions and body. Keep customer loading boundaries inside this group so administrator and sign-in layouts remain independent. Route paths do not include the group name.

Navigation uses Next.js `useLinkStatus` for a small pending indicator and accessible destination announcement. Keep normal link prefetching and modifier-key behavior. Do not simulate route progress with a timer or update `aria-current` before the destination commits. The mobile drawer stays available while navigation waits, then closes on commit; selecting the current destination closes it immediately. Respect reduced motion.


## Administrator interface

The administrator console uses the same brand mark, white surfaces, neutral borders, typography and semantic colors as the personal account area. Keep a wider 1440px frame for operational tables, a compact 72px header, and 28px page titles (24px on mobile). A grouped desktop sidebar becomes a navigation drawer below 901px. Administrator identity is read-only in a small disclosure with website and sign-out actions.

`lib/admin-navigation.ts` owns section names, descriptions, icons and three navigation groups: Operations, Billing, and Website & system. `AdminPageHeader` keeps page hierarchy and refresh feedback consistent. Router-owned pending labels provide feedback until navigation commits; loading replaces only the page content. Keep authorization in `AdminShell` and the API.

Use `DataTable` for lists: labeled search controls, a record count, scoped overflow, loading/empty/error states and server pagination. Right-align numeric columns. Display account email with a secondary ID where the API provides both; keep identifiers available in full. Payment amounts remain in minor units and the UI must state that unit.

Use `StatusBadge` with `adminTag` to label enums within their section and field. A pending payment, invitation and audit action have different meanings. Never translate arbitrary reasons, IDs or unknown states into a different business outcome. Include text in every status; color alone is insufficient.

Credit grants, consumption returns and referral reconciliation use `ResponsiveDialog` for focused operations. Keep reasons, validation, idempotency, session ownership and existing API behavior intact. Site settings separate API service details, support channels and developer-example parameters. Service integration forms distinguish saved configuration from the running API process; configured credentials do not imply a tested connection.


## Customer area scope

Keep exactly three primary destinations: account, subscription/billing and referrals. Credit history remains an auxiliary account page. Product-specific content, file and AI interfaces belong to the product built from this template. The frontend has no default notes/export or file-upload page. Retired `/notes` and `/files` addresses redirect to `/account`; their old public example articles redirect to the localized `/examples` directory, which links directly to the three common services. Preserve the shared infrastructure, backend reference code and existing data.
