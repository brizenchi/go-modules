# Editorial content ownership

`seed.json` is a reviewed bilingual **import pack**, not a public read fallback.
It contains 34 records: 9 pages/introductions, 12 documentation entries, 3 product
articles, 6 feature details, 3 use cases and 1 comparison. Import timestamps are
real publication times; no dates, customers, traffic or revenue are invented.

Initialize from Admin → Content → Initialize content. Preview is read-only. Apply
creates missing records and publishes EN/中文 through the audited content service,
atomically. Existing paths are skipped, including deliberately unpublished
records. Normal application startup and reads never import it. The `cmd/preview`
acceptance fixture explicitly imports into its own disposable temporary database.

## Evidence for the copy

- Authentication: `internal/platform/auth_provider.go`, auth HTTP adapters and frontend
  sign-in methods; email codes and configured Google/GitHub are distinct from the
  administrator password flow.
- Payments: `modules/billing`, template membership/checkout history and frontend
  pricing/billing flows. The example prices are demonstration offers. No claim of
  automatic refund/dispute reversal or a complete metered invoice system.
- Credits: `internal/credit`, `CREDITS_SERVICE.md` and the note export reference.
  Atomic local database work is distinct from external AI task compensation.
- Referrals: `internal/bootstrap/host_hooks.go`, `subscriptions.go` and
  `REFERRAL_HOOKS.md`: 20/10 at attributed registration, another 20/10 at qualifying
  paid activation, each stage once. Rules belong to host-product code.
- AI: `foundation/aix`, `internal/ai`: stateless text adapter, no default chat UI,
  conversation store, public endpoint or product charging policy.
- Content/admin: content/operations services and the corresponding Next.js admin
  and public renderers. Public text and SEO read live snapshots, not drafts.

## Ongoing editing

Operators maintain deployed content in admin. Editing this pack does not change
an existing database. `scripts/sync-content-seed.cjs` validates pack completeness;
it no longer copies frontend defaults over editorial work. Feature and use-case
claims must distinguish implemented foundations from proposed product examples.
Update both languages when code changes, without silently overwriting a site's
custom content on upgrade.

Structural homepage and directory routes remain available before editorial
initialization: they show saved site identity/navigation and empty lists, with
`noindex`. They never fabricate article content or publication dates. Missing or
unpublished article details still return 404. Home metadata belongs to the home
page, not the shared public layout, so its temporary `noindex` cannot leak into
pricing or other complete pages.

Legal policies, prices/catalogue, artwork and functional application screens are
separate product configuration/code. Server rendering and crawlable links help
discovery; neither the template nor the copy guarantees indexing or rankings.
