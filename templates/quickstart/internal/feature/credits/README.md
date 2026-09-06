# Credits and paid note exports

This host-owned feature supplies an immutable credit ledger, expiring grants,
earliest-expiry-first spending, and one full refund per consumption. Credit lots,
ledger entries, allocations, and `users.credits` change in one transaction under
the user's database row lock. Updating a profile cannot overwrite the balance.

Register `credits.New(deps.Users).Register(groups)` and add `credits.Models()` to
host model registration. `user.Models()` includes the ledger tables. Inject a
validated setting using `credits.WithExportCost(func(context.Context) (int64,
error))`; the default note export costs **1 credit**, with an accepted configured
range of 1–1,000,000. Prices are never supplied by the requesting user.

## Upgrade existing databases

Review and explicitly apply `migrations/20260906_credit_ledger.sql` to PostgreSQL
with application writes stopped and a backup available. The application does
not run this migration automatically. AutoMigrate creates new schema but does
**not** migrate existing balances. Existing accounts keep their balance readable;
credit operations return `503 credit_ledger_migration_required` until migrated.

The migration preserves each current balance as a non-expiring opening lot and
keeps the old `user_credit_grants` table intact for webhook deduplication. It does
not recalculate balances from historical grants, which could restore credits
already consumed. Negative old balances or partial ledger state stop the whole
migration for review. Running the completed migration again is safe.

New users created through `user.Repository.Create` are initialized automatically;
an explicitly supplied positive starting balance creates an opening transaction.

## HTTP contract

All paths below have the `/api/v1` prefix and return the normal `{code,msg,data}`
envelope. User routes require authentication; admin routes additionally check
the authenticated admin role. JSON body fields not in the contract are rejected.

| Method and path | Input | `data` |
| --- | --- | --- |
| `GET /credits` | — | `{balance,expiring_credits,next_expiry_at?}` |
| `GET /credits/transactions` | `page=1&limit=20` | `{list,total,page,limit}` |
| `GET /admin/credits/transactions` | optional `user_id`, pagination | `{list,total,page,limit}` |
| `POST /admin/credits/grants` | `{user_id,amount,expires_at?,reason,idempotency_key}` | `{transaction,balance}` |
| `POST /admin/credits/refunds` | `{transaction_id,reason,idempotency_key}` | `{transaction,balance}` |
| `POST /notes/:id/export` | `{idempotency_key,expected_cost}` | `{filename,content,transaction_id,balance}` |

An idempotency key must be 8–128 printable non-space ASCII characters, normally a
UUID. Reuse the same key when retrying the same operation; changed inputs with the
same key return `409 idempotency_conflict`. Grant/refund keys are scoped to the
authenticated administrator; export keys are scoped to the authenticated user.
Pagination limits are 1–100, pages 1–1,000,000. Expiry timestamps use RFC3339;
new grants require a future expiry. Grant amount must be positive and no more
than 1,000,000,000,000. Empty reasons are rejected for administrator mutations.

Each transaction contains `id,user_id,kind,amount,balance_after,source,source_id,
reason,actor_id,created_at`, plus optional `expires_at,related_transaction_id`.
Kinds are `opening`, `grant`, `consume`, `refund`, and `expire`. Amount is signed;
`balance_after` is historical. `expiring_credits` counts the remaining credits
expiring in the next 30 days; `next_expiry_at` is the earliest remaining expiry.
Expiry is applied whenever the account balance/ledger is read or mutated.

Refunds restore the full original consumption to its original owner, once.
Refunded credits do not expire. Partial refunds, refunding a grant or refund, and
a second refund with a new key are rejected. Actor IDs come from authentication
and are recorded with the operator's reason in the immutable transaction.

The export checks note ownership, saves a Markdown snapshot in the same database
transaction as its charge, and returns it to the client for download. A failed
database operation rolls back the charge. Repeating a successful key returns the
same file even if the note or configured price changed; it never charges twice.
A new export must include the displayed `expected_cost`; a changed server price
returns `409 price_changed` without charging, so the user can review the new cost.
A new key intentionally creates a new paid export. No emails, payment requests,
or other external services are called by this example.

Errors use `400` for invalid input, `404` for missing/inaccessible notes or
transactions, `409 insufficient_credits`, `409 idempotency_conflict`, and
`409 already_refunded`. A general database error returns a generic `500` without
exposing SQL or connection details.

## Using credits in another feature

Use `GrantCredits(ctx,userID,source,sourceID,amount)` for existing signup, referral,
and Stripe events. For new grants with expiry or audit fields, use
`GrantCreditsWithExpiry`. Use `ConsumeCreditsAndDo` to store a business result in
the same transaction as its charge. Its callback runs once and must perform only
database work: callback failure rolls back the charge, and a retry may succeed.
An external job should store its outbox/job row in that callback, then process it
outside the transaction. `AddCredits` is retained for old hosts but cannot make a
caller's retries idempotent; prefer operations with a stable source key.
