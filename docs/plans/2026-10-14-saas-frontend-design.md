# SaaS frontend design refresh

Approved direction: retain the free starter's warm white, dark neutral and green
identity, with clearer information hierarchy across public, customer and admin
surfaces. Use the existing Next.js/CSS stack without a new component dependency.

## Scope

- Extend the existing semantic design tokens and normalize controls, typography,
  Chinese heading spacing, data alignment, and responsive account layouts.
- Keep the account navigation limited to settings, subscription and referrals;
  keep administrator routes independent. Preserve language switching on mobile.
- Simplify account settings and progressively disclose developer credentials.
- Compare recurring plans under one billing-interval selector; separate lifetime
  access and credit packages. Label the public catalog as example pricing.
- Keep `/billing` as the single subscription page, without an additional
  subscription-management route. Keep current access and pending actions visible;
  use native `details` groups for plans, credits, and purchase/invoice history.
- Open the matching plans or credits group for an incoming purchase intent. With
  no incoming intent, open plans once subscription data is ready and confirms
  there is no ongoing recurring subscription or lifetime access. Loading and
  failed requests must not be treated as a confirmed absence of a subscription.
- Preserve the selected product, plan and interval across navigation and sign-in,
  including OAuth. Retain query parameters and anchors only on validated customer
  return paths, and validate the selection against enabled backend offers before
  purchase; unavailable selections require an explicit new choice.
- Place charge previews before confirmation and explain cancellation before
  committing. Opening a disclosure or returning from sign-in never starts payment.
- Simplify the public home into value, product flow, capabilities and setup guides;
  share controls and visual tokens with the independent admin console.

## Boundaries

No backend contract, migration, payment rule, dependency or deployment changes.
The capabilities response does not supply credit-package price/quantity metadata;
the page must not substitute marketing examples for real checkout amounts.
Existing concurrency guards, billing lifecycle handling and checkout refreshes stay.

## Verification

- Cover purchase intent parsing, capability validation, initial disclosure state,
  and retaining the validated customer destination after sign-in, including OAuth.
- Run `npm run verify` under Node 22 (tests, lint, build, public content checks).
- Inspect local public/account/billing pages in English and Chinese, including
  keyboard interactions and a narrow viewport where available. Use only the
  isolated `cmd/preview` backend with temporary data and external payments disabled.

These checks cover local UI behavior and automated frontend logic. They do not
constitute acceptance of a live Stripe purchase or production payment flow.

The reusable visual rules are documented in
`templates/quickstart-nextjs/DESIGN.md`.
