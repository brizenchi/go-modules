# quickstart-nextjs

Runnable Next.js SaaS frontend template for the shared `go-modules`
backend contract.

Use it with [`templates/quickstart`](../quickstart/) when you want a
copyable frontend that already includes public marketing pages, docs,
pricing, login, account settings, billing management, and referral
surfaces.

## What this template already includes

- top navigation
- breadcrumbs
- article table of contents
- EN / 中文 language switch
- login button and signed-in avatar menu
- pricing page
- docs page
- account settings page
- billing and subscription management page
- referral center

## Backend contract assumed by this template

- backend routes mounted under `/api/v1`
- JSON envelope from `foundation/httpresp`
- auth routes compatible with `stacks/saascore`
- billing routes compatible with the shared Stripe HTTP contract
- referral routes compatible with the shared referral HTTP contract
- referral attribution supported through `referral_code` on login or
  signup flows

## Main routes

- `/`: public product home
- `/pricing`: public pricing page
- `/docs`: docs-style article page
- `/login`: email-code login and Google OAuth entry
- `/account`: settings, session refresh, logout, WS ticket
- `/billing`: checkout, subscription, invoices, cancel, reactivate
- `/billing`: checkout, subscription change, Stripe Billing Portal, invoices, cancel, reactivate
- `/billing`: checkout, subscription preview, subscription change rules for monthly/yearly switches, Stripe Billing Portal, invoices, cancel, reactivate
- `/referrals`: referral link, stats, history
- `/invite`: capture `?ref=...` before signup

## Copy and run

```bash
cp -R templates/quickstart-nextjs ~/code/your-new-frontend
cd ~/code/your-new-frontend
cp .env.example .env.local
npm ci
npm run dev
```

Default local pairing:

- frontend: `http://localhost:3000`
- backend: `http://localhost:8080/api/v1`

## Frontend env

```dotenv
NEXT_PUBLIC_APP_URL=http://localhost:3000
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080/api/v1
NEXT_PUBLIC_APP_NAME=Clawmesh Quickstart Frontend
NEXT_PUBLIC_DEFAULT_PLAN=pro
NEXT_PUBLIC_DEFAULT_INTERVAL=monthly
NEXT_PUBLIC_DEFAULT_CREDITS_QUANTITY=1
NEXT_PUBLIC_CREDITS_PRICE_ID=
NEXT_PUBLIC_STRIPE_SUCCESS_PATH=/billing?checkout=success
NEXT_PUBLIC_STRIPE_CANCEL_PATH=/billing?checkout=cancelled
```

`NEXT_PUBLIC_DEFAULT_PLAN` should be one of `starter`, `pro`, `premium`, or `lifetime`.

## Required backend mapping

When frontend origin or route paths change, keep these backend values
aligned:

```dotenv
APP_AUTH_FRONTEND_REDIRECT=http://localhost:3000/login
APP_AUTH_GOOGLE_REDIRECT_URL=http://localhost:8080/api/v1/auth/google/callback
APP_REFERRAL_BASE_LINK=http://localhost:3000/invite?ref=
```

Rules:

- Google authorized redirect URI points to the backend callback
- Stripe webhook points to the backend, never to Next.js
- Stripe success and cancel URLs point back to frontend pages

## What you usually change

- `NEXT_PUBLIC_APP_NAME`
- brand copy, product copy, pricing copy
- page content under `/`, `/pricing`, `/docs`
- plan names and credits configuration
- account menu extensions such as workspace switch or profile settings

## Verification

```bash
npm ci
npm test
npm run verify
```

## Manual verification

Before treating the frontend template as ready, confirm:

1. `/invite?ref=CODE` stores the referral code
2. `/login` completes email-code login
3. Google OAuth round-trip works when configured
4. `/pricing` routes users into `/billing`
5. `/billing` can start first subscription checkout when user has no paid plan
6. `/billing` can start lifetime buyout checkout when configured
7. `/billing` can change active subscription plan in place
8. `/billing` can open Stripe Billing Portal
9. `/referrals` loads live referral code, stats, and history
10. signed-in avatar menu shows settings, subscription, referral, and logout entry points
