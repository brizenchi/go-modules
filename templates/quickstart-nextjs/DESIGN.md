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

- Customer navigation remains account, subscription and referrals. Notes, files
  and credit history remain auxiliary personal features; admin never joins this nav.
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
