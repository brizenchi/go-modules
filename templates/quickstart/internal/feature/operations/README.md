# Operator features

Mount with `operations.New(deps).Register(groups)` and append `operations.Models()` to host models. Existing `/api/v1` auth groups supply the identity; this module independently requires the admin role for every operator route. Admin emails remain deployment configuration. Public settings never expose login, payment, mail or storage credentials.

## API

All JSON uses `{code, msg, data}`. List endpoints return `{items, total, page, limit}`; pages start at 1, default limit 20, maximum 100. `query` searches only documented identifiers and email/name fields as literal text, with a 200-character limit.

- `GET /admin/overview`: user, subscription and referral counts; **not DAU**. Disabled modules return zero counts.
- `GET /admin/users`: profiles and current effective credit balances. Legacy balances retain their existing value until the explicit credit ledger migration.
- `GET /admin/subscriptions`: stored provider subscription snapshots, without raw provider data.
- `GET /admin/orders`: invoice/checkout summaries derived from persisted verified payment events, with invoice IDs deduplicated across event kinds and checkout references. `amount` is nullable and in currency minor units; `livemode` indicates real/test provider records where present. Subscription lifecycle events are excluded. Source events are scanned in batches; for a large payment history, materialize this projection in the billing listener. Invoices themselves remain available through the provider's billing portal.
- `GET /admin/referrals`: invitation records, optional `status=pending|activated|expired`, referrer/referee emails, reward amount and deadline.
- `POST /admin/referrals/:id/retry-reward`: body `{reason}`, plus `Idempotency-Key`. Only an already activated record with a stored activation time can replay its original reward event. This does **not** qualify pending invitations or change rewards. A failed listener can be retried with the same key; the credit ledger prevents duplicate grants.
- `GET /admin/audit`: actor, reason, action, target, status and outcome for settings changes and referral reward reconciliation. Credit adjustments have their own actor/reason ledger.
- `GET /site/settings`, `GET /admin/settings`: `{brand_name, description, support_email, support_url, export_credit_cost}`. Defaults work before the first row is saved. Empty support fields mean support is not configured.
- `PATCH /admin/settings`: a subset of those fields plus a mandatory `reason`, and `Idempotency-Key`. Unknown fields are rejected. Export cost is an integer from 1 to 1,000,000. Support URLs must use HTTPS. Changes and audit reservations commit in one transaction. Reusing a key with different data returns 409.
- `POST /uploads/images`: authenticated multipart form with exactly one `file`. Maximum 5 MiB. Content is sniffed independently of supplied MIME/extension; PNG/JPEG/GIF headers are validated with a 40 megapixel limit, and WebP containers are checked. SVG is rejected. The response contains `id`, `url`, `content_type`, `size`, `filename`, `created_at`.
- `GET /uploads/images`: the signed-in user's metadata, paginated.
- `GET /uploads/images/:id`: image bytes for the same owner only. Use authenticated fetch and an object URL in the browser; this is not a public CDN URL. Files have generated names and private cache headers.

## Storage

Uploads are disabled until `host.uploads.enabled=true`. `provider=local` requires an explicit `directory`. Local development files use generated paths inside an owner directory with private permissions and filesystem-root confinement. Do not expose that directory as static web content. A local production deployment needs a persistent private volume, backups, and a single shared filesystem.

For multiple instances or ephemeral deployments, set `provider=s3`, `bucket`, `region`, and optionally `endpoint`, `use_path_style`, `access_key_id`, `secret_access_key`. R2 uses region `auto` and its account S3 endpoint. Credentials belong in deployment secrets, never in settings. AWS may use its default credential chain when explicit credentials are omitted. Use a private bucket and disable public access. The same authenticated read route proxies private S3/R2 objects.

Provider changes do not migrate existing objects. Metadata records the original provider and returns unavailable if it no longer matches configuration. Move files and metadata through an explicit migration before switching a deployed storage backend.

The supplied SQL migration creates only these host feature tables. No migration or external database command is run by tests; tests use isolated SQLite databases and temporary directories. The existing quickstart startup migration wiring also includes these models after the host registers them.
