# Navigation and support consistency

Keep the customer area's three primary destinations. Unify labels and destinations in shared navigation metadata; add a compact customer footer for help/privacy/terms so support stays visible on mobile. Use semantic breadcrumbs without public schema on private routes. Align customer headings with navigation names.

Public navigation becomes Home, Features, Pricing and Resources. The existing /features route remains the single feature introduction; present six shipped capability summaries with links to their details and the account demo. Use cases remain reachable within the product area and footer. Preserve published routes, metadata and locale behavior.

The existing /contact route becomes the consistently named Help & support page. Customer self-service and common account/billing/credit/referral questions replace operator setup guidance. Preserve the real support settings API and its loading/unavailable states; do not invent contact details or response promises. Provide a clear account return link.

Verify route ownership, shared link destinations, private breadcrumb hierarchy and published content. Run isolated frontend verify and local route/link checks. Browser acceptance depends on the available computer-use service. The navigation work itself has no backend/API changes. Later admin requests are implemented separately in the admin operations design. No commit, push or production action.


Validation completed locally:
- Frontend `npm run verify`: 184 tests, lint, production build and 72 public-page content checks passed (isolated copy; the development `.next` output was not overwritten).
- Backend `go test -race ./...` and `go vet ./...` passed. Focused regressions cover administrator access, member/authorization separation, payment subscription preservation and automatic fallback, credit idempotency/overdraw, calendar boundaries and payment deduplication/environment separation.
- Root format, race, vet, purity and CI-version lint passed. The detached backend release-copy verification (tests/vet/native and Linux builds) also passed.
- Local HTTP checks returned 200 for EN/ZH features/help, the four account routes and admin routes; private noindex remained present. Shared navigation links and translated help/footer contents matched.
- Browser control was unavailable, so authenticated visual and mobile interaction acceptance remains unverified. The running local backend is older; new metrics/member operations require a local backend restart. Frontend authorization falls back to the old protected overview endpoint only on a 404, preserving existing admin access during a staggered update. No production change, service restart, SQL operation against a configured database, commit or push was performed.
