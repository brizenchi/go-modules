# Customer Frontend Interactions Implementation Plan

**Goal:** Finish the customer UI as a cohesive SaaS template without changing backend contracts or commercial configuration.

**Architecture:** Share a native modal component for account and sign-in, with responsive bottom sheets and a navigation drawer. Compose an independent customer shell inside SiteShell. Derive product-specific checkout progress from existing purchase intent, preserving the global request lock.

**Tech Stack:** Next.js 15, React 18, TypeScript, CSS modules; no added dependencies.

### Task 1: Dialog primitives
- Create `components/responsive-dialog.tsx` and its CSS module.
- Add pure focus-boundary logic and tests in `lib/dialog-focus.ts` and `tests/dialog-focus.test.ts`.
- Migrate `components/account-dialog.tsx` and `components/sign-in-dialog.tsx`; preserve auth callbacks, OAuth destination and error behavior.
- Verify focus wrap, Escape, backdrop, body scroll restoration and mobile sizing in the browser.

### Task 2: Customer layout
- Create `components/customer-shell.tsx` and CSS module.
- Replace the workspace branch in `components/site-shell.tsx` with the customer shell, keeping account menu and locale controls.
- Retain route recognition, existing actions and optional context. Keep admin untouched.
- Verify desktop and mobile navigation, current route and return-home links.

### Task 3: Purchase progress
- Add pure selection comparison in `lib/pricing-checkout.ts`, with tests in `tests/pricing-checkout.test.ts`.
- Expose active intent from `components/use-pricing-checkout.ts`; render loading indicators only on the matching pricing button while preserving all write locks.
- Keep demonstration copy, price values, capabilities checks and checkout payloads unchanged.

### Task 4: Verification
- Run frontend `npm run verify` in a temporary copy to preserve the running dev server's `.next`.
- Check English and Chinese desktop/mobile views, keyboard use, authentication method visibility and checkout feedback.
- Save screenshots. Do not commit, push or deploy.
