# Database-owned public content

Scope: replace virtual starter publications with persisted admin drafts and live
snapshots; manage home and archive introductions through the same CMS; replace
starter copy with verified English and Chinese content; render publications and
metadata on the server.

1. Test empty-database behavior, explicit starter import, independent locale
   publication and persistence. Remove read-time seed overlays.
2. Extend content with an allowlisted `page` kind. Add an admin-only import preview
   and explicit apply operation that skips existing paths and uses audited writes.
3. Author the starter pack from implemented features, with contextual links and
   explicit limitations. No invented traffic, customers, revenue or launch claims.
4. Render managed home, archive introductions and article bodies from published
   snapshots. Keep drafts private, preserve translated links, metadata and sitemap.
5. Verify backend tests and frontend checks, then inspect initial HTML and layouts
   against a disposable database. Never write a configured database implicitly.

Deployment/data: production remains read-only. The target for the requested real
data import must be identified before applying to an existing environment. The
import is preview-first and never replaces an operator's existing content.
