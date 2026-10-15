# Consistent administrator interface

The user requested consistent styling with the customer area and clearer sections/labels. Keep the existing API routes, authorization, payment and credit operation behavior. Retain real data only and the current service configuration lifecycle.

Use the same brand mark, neutral surfaces, type scale, border/radius tokens and compact header. Group nine destinations into Operations, Billing, and Website & system. Give each page a specific description. Use line icons, router-owned navigation feedback, a mobile drawer, and a compact administrator disclosure with read-only identity and sign-out.

Use the existing shared DataTable for lists and consistent status badges. Scope enum labels to the appropriate column/domain so payment, invitation and audit states are not mislabeled. Keep payment amounts in existing minor units, explicitly labeling that unit. Separate credit/reward operations into the shared dialog and organize site settings into form sections. Replace integration-page hardcoded colors and spacing with design tokens; keep all validation, versioning and idempotency behavior.

Validate navigation coverage and scoped enum labels, then run isolated frontend verify. Check public/private metadata and local route health. Attempt browser acceptance without changing production or existing service configuration. No new dependencies, database changes, deployment or push.

## Validation

Implemented the shared shell, grouped navigation, page header, scoped badges, shared list tables, focused operation dialogs and sectioned settings/integration forms. Existing service save lifecycle, request ownership, validation and idempotency remain in place.

- Frontend `npm run verify` passed in an isolated working copy: 176 tests, ESLint, production build and 76-page content validation. An additional enum-prototype edge case failed first and passed after the fallback correction.
- Local GET checks returned HTTP 200 and noindex for all nine administrator destinations, plus the personal account route.
- `git diff --check` passed.
- Browser acceptance remains incomplete: the computer-use service reported a native-pipe startup failure and exposed no browser/app surfaces. No authenticated click-through or mobile visual acceptance is claimed.
