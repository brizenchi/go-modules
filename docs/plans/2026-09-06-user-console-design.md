# Quickstart User Console Design

## Product decision

Settings, billing, and referrals are account-level workflows, so they belong in a persistent signed-in console rather than a collection of popups. Popups remain useful for short, interruptible tasks such as sign-in, copying an invitation, confirming a plan change, or cancellation. A separate administrator product is out of scope: this console is for the current customer and must not mix customer actions with operator controls.

## Visual direction

The console is a calm editorial operations desk: a warm paper canvas, a graphite navigation rail, signal-green status accents, precise hairlines, and large Newsreader numerals for real account data. It keeps the existing Manrope/Newsreader type pairing but gives it a more purposeful hierarchy. The memorable element is the dark navigation rail beside quiet, high-contrast data surfaces—not decorative gradients or generic rounded-card grids.

## Information architecture

Public routes keep the existing marketing header. Signed-in routes use a dedicated account shell with only three persistent destinations: Settings, Subscription, and Referrals. There is no overview item or intermediate dashboard; sign-in opens `/account` directly, and the legacy `/dashboard` URL redirects there for compatibility. Each destination retains its working forms, real API data, and explicit error states while using compact account-center page headers.

## Interaction and responsive behavior

Desktop uses a fixed-width sidebar and a compact utility bar. Mobile replaces the sidebar with a horizontally scrollable console navigation so every feature remains one tap away without hiding it behind an avatar. Loading, disabled, authentication, and network states remain explicit and retryable. Focus indicators, semantic navigation labels, keyboard operation, reduced-motion preferences, and horizontal table overflow are preserved.

## Data and safety

The account center does not invent analytics or add a redundant summary layer. Each page reads only the account, capability, subscription, invoice, or referral data it needs. Authentication continues to use the browser-bound OAuth and email flows already implemented. Billing mutations and referral attribution stay in their existing pages and backend services.
