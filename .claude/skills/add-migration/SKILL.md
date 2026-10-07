---
name: add-migration
description: Change the database schema or existing data safely (new tables/columns, backfills, renames, drops, indexes) following docs/standards/DATABASE.md. Use when a task touches GORM models, table structure, or requires a SQL migration file.
---

# Database change

Standard: `docs/standards/DATABASE.md`.

## 1. Classify the change

| Change | How |
| --- | --- |
| New table, new nullable column, new index on a small table | Edit the GORM model; `AutoMigrate` applies it at boot |
| Backfill / transform existing data, change a type, drop or rename a column, `NOT NULL` on existing column, index on a large table | New SQL file in `templates/quickstart/migrations/`, run manually after backup |

Never edit an existing file in `migrations/`. Never drop or rename something the currently
deployed code still reads (expand → migrate → contract across separate releases).

## 2. Model conventions
- Table/column names snake_case; tables plural; module tables prefixed (`billing_…`).
- Main entities: `varchar(36)` UUID ids; child/ledger rows may use `bigint` auto-increment.
- Money: `bigint` minor units + currency column. Never floats.
- `created_at` (+ `updated_at` for mutable rows), stored in UTC.
- Index every foreign key and every column used in `WHERE`/`ORDER BY` of list queries.

## 3. SQL migration file
- Name: `YYYYMMDD_<description>.sql` (today's date).
- Header comment: what it does, prerequisites (backup, stop writes), how to roll back.
- Wrap in `BEGIN; … COMMIT;`; make it re-runnable (`IF NOT EXISTS`, `ON CONFLICT DO NOTHING`).
- `CREATE INDEX CONCURRENTLY` goes in its own file without a transaction.

```sql
-- Adds notes.archived_at so notes can be archived.
-- Prerequisite: none (additive). Rollback: ALTER TABLE notes DROP COLUMN archived_at;
BEGIN;
ALTER TABLE notes ADD COLUMN IF NOT EXISTS archived_at timestamptz;
COMMIT;
```

## 4. Code
- Queries use `db.WithContext(ctx)`, parameter binding, stable order + `LIMIT`.
- Balance/credit updates use conditional updates or a version column.

## 5. Tests and docs
- Repository tests on SQLite in-memory; note PostgreSQL-only SQL in the PR and test it on PostgreSQL.
- PR description states the release order and the rollback.

Commit as `feat(<scope>): …` or `fix(<scope>): …`; never mix a destructive migration with the
code change that stops using the column.
