# Databases

Written for relational databases (PostgreSQL and similar). This project's ORM, migration tool and directories are recorded in [PROJECT.md](./PROJECT.md).

## Ownership

- Each table is owned by one module; other modules go through its interface or events and never read or write its tables directly.

## Naming

| Object | Rule | Example |
| --- | --- | --- |
| Tables | snake_case, plural; a module prefix is allowed | `invoices`, `billing_subscriptions` |
| Columns | snake_case | `user_id`, `current_period_end` |
| Primary key | `id` | |
| Foreign keys | `<referenced table, singular>_id` | `user_id` |
| Timestamps | `<action>_at` | `created_at`, `deleted_at` |
| Booleans | `is_<adjective>` or a state | `is_active`, `email_verified` |
| Indexes | `idx_<table>_<columns>`; unique ones `uniq_` | `idx_invoices_user_id` |

## Column types

| Data | Type | Notes |
| --- | --- | --- |
| Main entity IDs | UUID (or string) | entities referenced widely or appearing in public URLs |
| Child records, ledger IDs | auto-increment `bigint` is fine | when counts need not be hidden |
| Times | timezone-aware type, stored in **UTC** | |
| **Money** | integer in the **minor unit**, plus a currency column | **never floating point** |
| Enums | `varchar(n)`, values validated in code | |
| Text | `varchar(n)` when bounded, otherwise `text` | |
| Loosely structured extras | `jsonb` | small data not used in queries or constraints; promote to columns or tables once stable |

Tables that are updated have `created_at` and `updated_at`; append-only ledgers need only `created_at`.

## Migrations

- Name migrations by date (for example `YYYYMMDD_<description>`) and start each with: what it does, prerequisites (backup, write freeze), how to roll back.
- Make statements re-runnable where possible (`IF NOT EXISTS`, `ON CONFLICT DO NOTHING`).
- **Never edit a migration that has run in any environment**; add a new one to correct it.

### Zero-downtime changes: expand, then contract

Changing or removing a column takes several releases so that old and new code both keep working:

```text
release 1: add the new column (nullable); code writes both columns
release 2: backfill existing rows; code reads the new column
release 3: code stops using the old column
release 4: drop the old column
```

- Index large tables without locking them (PostgreSQL: `CREATE INDEX CONCURRENTLY`, outside a transaction).
- Adding a `NOT NULL` column: add it nullable, backfill, then add the constraint.

## Queries

- Queries carry the request context (cancellable, visible in traces).
- Parameterised queries only — **never** concatenate user input into SQL; dynamic sort columns come from an allow-list.
- List queries always have a `LIMIT` and a stable order.
- Avoid N+1 queries; select only the columns you need and keep large columns out of list queries.
- Record and fix slow queries; SQL in logs and traces **omits parameter values**.

## Transactions and concurrency

- Multi-step writes that must be atomic share one transaction.
- **No outbound HTTP calls inside a transaction** (they hold locks for too long); commit first, or use the outbox pattern.
- Decrement balances, credits and stock with conditional updates (`UPDATE … WHERE balance >= ?`) or optimistic locking (a version column).

## Deletion and retention

- Delete rows by default; use soft deletes (`deleted_at`) only for data that must be audited or restored, with partial unique indexes.
- When a user deletes their account, delete or anonymise their personal data according to the privacy policy.

## Environments

- Tests use their own database (in-memory or a test container); statements relying on database-specific features are verified against the real database.
- Connection details come only from environment variables; **never copy production data to a local machine**.
