# Quickstart SaaS Visual Redesign

## Direction

The Next.js quickstart will use an Attio-inspired editorial hierarchy with Clerk-inspired product framing: a quiet off-white canvas, precise hairlines, a single green accent, generous negative space, and an interface preview that makes the starter feel like a real product.

The result should feel calm and exact rather than decorative. It avoids gradients, glass panels, oversized rounded cards, and dense “component gallery” layouts.

## Visual system

- Palette: paper white, ink black, cool gray borders, muted gray copy, and one signal-green accent.
- Typography: a crisp grotesk-style UI stack with tight display tracking and compact monospace labels.
- Geometry: 12–16px surfaces, 8–10px controls, square product framing, and restrained pill usage only for status chips.
- Depth: one subtle product-window shadow; regular content surfaces rely on borders and spacing.
- Motion: short entrance stagger and small hover movement, disabled for reduced-motion users.

## Information architecture

- The shared header becomes a compact floating navigation bar with a recognizable symbol mark, three public routes, locale control, and account entry.
- The shared hero becomes more editorial: breadcrumbs and eyebrow above a large, readable headline; environment details sit in a quiet utility column.
- The home page adds a convincing application preview immediately after the hero, then explains capabilities, routes, and backend contracts with fewer visual containers.
- Pricing, docs, login, account, billing, invite, and referral pages keep their current behavior and content while inheriting the new system.

## Responsive behavior

- Desktop keeps the split hero and full product preview.
- Tablet collapses hero/supporting panels while preserving horizontal navigation where practical.
- Mobile uses a two-row header, single-column grids, scrollable data tables, and compact preview details.

## Accessibility

- Interactive elements retain visible focus states and adequate contrast.
- Decorative preview graphics are hidden from assistive technology where appropriate.
- Animation respects `prefers-reduced-motion`.

