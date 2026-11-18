-- Moves legacy users.credits balances into the template credit ledger for a
-- database created by stacks/saascore (go-modules d6a572f or earlier).
-- See docs/MIGRATE_FROM_SAASCORE.md.
--
-- Use this file instead of templates/quickstart/migrations/20260906_credit_ledger.sql:
-- it does not require user_credit_grants (saascore never had it) and does not
-- create the template's note_exports example table. It works whether or not the
-- new application has already started once and AutoMigrated these tables.
--
-- Each positive balance becomes one non-expiring opening lot. Every legacy row
-- is marked credits_version = 1; until then the template rejects credit
-- operations for that user with credit_ledger_migration_required.
--
-- Run explicitly, after a backup, while application writes are stopped.
-- Rerunning after success changes nothing.
BEGIN;
LOCK TABLE users IN SHARE ROW EXCLUSIVE MODE;
SET LOCAL client_min_messages = warning; -- hide "already exists, skipping"

ALTER TABLE users ADD COLUMN IF NOT EXISTS credits_version integer NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS user_credit_transactions (
  id bigserial PRIMARY KEY,
  user_id varchar(36) NOT NULL REFERENCES users(id),
  kind varchar(16) NOT NULL,
  amount bigint NOT NULL,
  balance_after bigint NOT NULL,
  source varchar(32) NOT NULL,
  source_id varchar(255) NOT NULL,
  reason varchar(500) NOT NULL,
  actor_id varchar(36) NOT NULL DEFAULT '',
  expires_at timestamptz,
  related_transaction_id bigint,
  created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS uniq_credit_transaction_source ON user_credit_transactions(source, source_id);
CREATE UNIQUE INDEX IF NOT EXISTS uniq_credit_refund_transaction ON user_credit_transactions(related_transaction_id);
CREATE INDEX IF NOT EXISTS idx_user_credit_transactions_user_id ON user_credit_transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_credit_transactions_kind ON user_credit_transactions(kind);

CREATE TABLE IF NOT EXISTS user_credit_lots (
  id bigserial PRIMARY KEY,
  user_id varchar(36) NOT NULL REFERENCES users(id),
  transaction_id bigint NOT NULL REFERENCES user_credit_transactions(id),
  amount bigint NOT NULL CHECK (amount > 0),
  remaining bigint NOT NULL CHECK (remaining >= 0 AND remaining <= amount),
  expires_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_credit_lots_transaction_id ON user_credit_lots(transaction_id);
CREATE INDEX IF NOT EXISTS idx_credit_lot_user_expiry ON user_credit_lots(user_id, expires_at);

CREATE TABLE IF NOT EXISTS user_credit_allocations (
  id bigserial PRIMARY KEY,
  transaction_id bigint NOT NULL REFERENCES user_credit_transactions(id),
  lot_id bigint NOT NULL REFERENCES user_credit_lots(id),
  amount bigint NOT NULL CHECK (amount > 0)
);
CREATE INDEX IF NOT EXISTS idx_user_credit_allocations_transaction_id ON user_credit_allocations(transaction_id);
CREATE INDEX IF NOT EXISTS idx_user_credit_allocations_lot_id ON user_credit_allocations(lot_id);
SET LOCAL client_min_messages = notice;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM users WHERE credits_version = 0 AND credits < 0) THEN
    RAISE EXCEPTION 'Negative legacy credit balances require manual reconciliation; no balances were changed';
  END IF;
  IF EXISTS (
    SELECT 1 FROM users u JOIN user_credit_transactions t ON t.user_id = u.id
    WHERE u.credits_version = 0
  ) THEN
    RAISE EXCEPTION 'Unmigrated users already have ledger entries; review before migrating';
  END IF;
END $$;

-- created_at is explicit: tables created by AutoMigrate have no column default.
INSERT INTO user_credit_transactions(user_id, kind, amount, balance_after, source, source_id, reason, actor_id, created_at)
SELECT id, 'opening', credits, credits, 'opening', id, 'Opening balance migrated from saascore users.credits', '', CURRENT_TIMESTAMP
FROM users WHERE credits_version = 0 AND credits > 0;

INSERT INTO user_credit_lots(user_id, transaction_id, amount, remaining, created_at)
SELECT u.id, t.id, u.credits, u.credits, CURRENT_TIMESTAMP
FROM users u JOIN user_credit_transactions t ON t.source = 'opening' AND t.source_id = u.id
WHERE u.credits_version = 0 AND u.credits > 0;

DO $$
DECLARE
  migrated bigint;
  total bigint;
BEGIN
  SELECT count(*), COALESCE(sum(credits), 0) INTO migrated, total FROM users WHERE credits_version = 0;
  RAISE NOTICE 'users migrated: %, opening credits: %', migrated, total;
END $$;

UPDATE users SET credits_version = 1 WHERE credits_version = 0;
COMMIT;
