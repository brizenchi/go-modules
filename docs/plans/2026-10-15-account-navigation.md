# Account navigation responsiveness

The user approved immediate navigation feedback, a persistent account frame and independent content loading. Preserve the existing visual style, URL paths, public/admin layouts and billing/auth behavior.

Use a shared `(customer)` layout and Next.js `useLinkStatus`. A spinner alone leaves the frame mounting again; manually tracking clicks can get stuck after canceled navigation. Move the six customer route directories intact into the pathless group, extract existing account/language controls, and keep headings and actions owned by each page. Use a content-only loading boundary. Close the mobile drawer when the destination commits, or immediately for the current destination. Preserve native links, modifier keys, history and default production prefetching.

1. Add accessible pending/idle navigation regression coverage before implementation.
2. Extract controls and move the account frame into the shared route layout.
3. Add localized pending feedback and content placeholders with reduced-motion support.
4. Run isolated frontend verify and check route metadata, continuous navigation, history and mobile behavior where the UI connection is available.

No dependencies, API/database/payment changes, deployment or push are required.

Validation completed: 172 frontend tests, lint, production build and all 76 public content checks passed in an isolated copy. Customer HTML additionally verifies one header, main, H1 and account navigation per page, retaining noindex metadata. All six routes compile under the common group; local GET checks for those routes, login and admin returned 200. No production or backend state was changed.

The browser connector and native UI service were unavailable even after resetting the session. Actual click/history/mobile verification and a fresh screenshot could not be completed in this turn; no claim of measured end-to-end navigation speed is made.
