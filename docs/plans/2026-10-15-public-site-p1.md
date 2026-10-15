# Public site P1 implementation plan

**Goal:** Finish the three approved P1 items for a launchable SaaS template: accurate capability copy, a clearer homepage with a real product preview, and centralized public branding/homepage configuration.

**Architecture:** Keep the existing Next.js/Go boundaries and payment flows. Public presentation uses build-time configuration so initial HTML, navigation, metadata and sharing cards agree. Runtime support channels and business settings remain API-backed. Homepage sections stay explicit; no page builder, analytics expansion or new dependencies.

**Design:** Retain the warm light surfaces, restrained green accent and existing typography. Replace the homepage hero's entry directory with a screenshot of the actual customer workspace. Move the three entry links below the hero, shorten the heading and add a compact bilingual FAQ. Use real local demonstration data and label screenshots as previews.

**Implementation:**
1. Add failing regressions for stale account/AI descriptions and the neutral default brand.
2. Add `content/site.ts` for default brand, bilingual homepage/SEO/sharing copy, CTA destinations and preview assets. Keep environment access in `lib/env.ts`.
3. Wire public/customer presentation, metadata, sharing card and build-time fallback branding to that configuration. Explain the distinction from runtime service details in admin help and publishing docs.
4. Update capability and setup guides to describe the read-only account dialog and optional chat accurately, including its daily request limit and separate credit billing boundary.
5. Capture the existing workspace in both languages; update the hero, entry section and FAQ with responsive styles.
6. Run `npm run verify` in an isolated copy, leaving the user's running `.next` directory intact. Inspect English/Chinese desktop and mobile pages, FAQ, CTA links and account interactions. Save a review screenshot.

**Boundaries:** Do not change prices, checkout eligibility, login providers, backend behavior, schema, packages or deployments. Existing unrelated working-tree changes remain in place. No commit or push is part of this task.

**Validation completed:** The two baseline regressions failed before implementation. Final `npm run verify` passed in an isolated copy: 171 tests, ESLint without warnings, production build, and 74 public-page/30 article checks. Browser checks covered English/Chinese hero copy and real preview images, matching visible/metadata brands, 320px and 390px layouts without document overflow, native FAQ expansion, the setup anchor, and the new AI guide. Public branding is deliberately build-time; API support details and export cost retain their runtime behavior.
