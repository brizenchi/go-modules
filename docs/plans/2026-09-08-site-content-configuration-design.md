# Unified site configuration and publishing

Status: proposed design. The configuration source is awaiting a choice between admin-managed publishing and version-controlled configuration. This document does not imply that an editor or publishing API has been implemented.

## Objective

Make the SaaS template reusable without duplicating branding, copy, navigation, documentation, or article structures across pages. Keep its existing responsive visual language and SEO behavior. Layouts remain maintained components; configuration supplies their content and visibility, not arbitrary HTML or executable code.

## Current gaps

- Public site settings expose a brand name, description, contact channels, and export credit cost, but not a logo or navigation.
- Header branding and footer branding use different hard-coded marks. Footer groups are defined inside the component.
- Homepage copy lives in the page component. Documentation and blog entries have different structures and separate rendering paths.
- Article routes, metadata, and the sitemap currently depend on build-time content. Adding an admin editor alone would not make newly published articles accessible or update search metadata.

## Shared model

### 1. Brand

One brand record supplies the site name, logo, compact icon, favicon, tagline, support links, and default social-sharing image. Header, footer, sign-in, and administrative shell reuse a shared brand component with presentation variants. Attribution text is a configurable display field rather than being embedded in each component.

Logo variants have explicit display sizes and accessible text. Public branding assets must not reuse or expose the existing private user-upload namespace. A logo URL or separately authorized public asset upload can provide assets; arbitrary SVG/HTML must not be inserted into the page.

### 2. Page copy

Each supported page has a stable key, a heading, an introduction, actions, and supported content sections. Text uses the existing English/Chinese field shape. Repeated feature cards and links support addition, removal, ordering, and visibility within their predefined slots. Marketing descriptions remain separate from payment prices, product entitlements, and service credentials.

Global defaults and page-specific overrides provide SEO titles, descriptions, and sharing images. Rendered page content and server-generated metadata read the same published configuration.

### 3. Content

Use a shared hierarchy: collection -> category -> entry. Start with `docs` and `blog` collections; these share validation, editing, publication status, body rendering, and SEO generation.

- Collection: stable ID, route prefix, localized title and description, display mode.
- Category: stable ID, collection ID, localized title, ordering, and optional parent category. Reject cycles and excessive nesting.
- Entry: stable ID, collection/category reference, slug, localized title and summary, body, optional cover/author, ordering, draft or published status, and genuine publication/modification dates.

Documentation displays ordered navigation; the blog displays a date-ordered feed. Both use the same article reader. Collection navigation is distinct from the article's on-page table of contents: the latter comes from body headings. Breadcrumbs, next/previous entries, related entries, search, and the sitemap derive from these same records.

Only published entries appear publicly. Published URLs stay stable; any future slug changes need an explicit redirect policy. Existing `/docs/[slug]` and `/blog/[slug]` URLs and legacy guide fragment targets must be preserved during migration.

### 4. Navigation and footer slots

One menu-item shape is reused across locations: stable ID, localized label, destination, order, visibility, and external-link behavior. Internal destinations reference an existing page, collection, or entry rather than duplicating its URL in several components. External destinations are validated URLs.

Predefined locations:

- Header: primary navigation and a primary action.
- Footer brand area: shared logo/name, short introduction, optional action.
- Footer link area: configurable titled groups, defaulting to Product, Resources, and Company/Support; use a bounded responsive column count.
- Footer bottom row: copyright text and legal links.
- Footer social area: optional external profiles with accessible names; omit the area when empty.

Each location has a default configuration and allows adding, removing, sorting, and hiding links. Empty groups do not render. A disabled menu item does not unpublish its target article. A draft or removed content target must not leave a public dangling link.

## Configuration source decision

Recommended for no-code operation: an admin-managed published configuration backed by the Go application. Provide separate admin areas for Brand & Copy, Navigation & Footer, and Content. Draft/preview/publish prevents half-edited site-wide changes from appearing publicly. Admin writes retain authorization, validation, audit, and concurrency protection.

Lower-complexity alternative: the same model is edited in version-controlled configuration/content files and deployed through the existing build. This avoids a database-backed CMS but does not satisfy editing and publishing directly in the admin UI.

Do not introduce two equally authoritative sources. Starter configuration can initialize a site, but must not silently overwrite saved settings or resurrect intentionally hidden content.

## Publishing and SEO requirements

- Public HTML, metadata, favicon, structured data, navigation, and sitemap consume the same published configuration.
- Admin publishing requires runtime article resolution and a deliberate cache-refresh policy; the current build-only slug allowlist cannot remain the source of truth.
- Draft content is never included in anonymous APIs, public HTML, search, or the sitemap. Preview requires authorization and must not be indexed.
- Configuration failure must not turn a known published article into a false 404 or expose a draft as fallback content.
- Shared English/Chinese URLs remain the current limitation unless independently indexed locale routes are explicitly included in a later scope.
- Provider credentials and private user uploads never become part of the public site configuration.

## Implementation sequence

1. Agree the configuration source and define shared typed models, defaults, validation, and compatibility adapters for existing content.
2. Centralize the brand component, page copy, and menu/footer slots without changing existing URLs or business behavior.
3. Unify categories, entries, directory/search, and article rendering while preserving all existing starter content.
4. If admin publishing is selected, add authorized configuration/content editing and the server-side published-content loading/cache flow. Persist required schema migrations without running them against a configured database.
5. Verify configuration changes propagate across every surface; test invalid links, drafts, broken dependencies, old URLs, missing translations, logo failures, responsive layout, and SEO output.

## Non-goals

No arbitrary page-builder canvas, executable MDX, custom JavaScript fields, payment business changes, automatic production migration, or deployment. Search ranking and rich-result inclusion are not guaranteed by this design.
