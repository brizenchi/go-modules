-- PostgreSQL upgrade for an EXISTING quickstart database. Review and back up
-- first; run explicitly while application writes are stopped. This file is
-- never executed automatically by the application or the development agent.
-- Existing balances become non-expiring opening lots. Historical grant keys
-- remain in user_credit_grants so webhook retries cannot issue credits again.
BEGIN;
LOCK TABLE users, user_credit_grants IN SHARE ROW EXCLUSIVE MODE;

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

CREATE TABLE IF NOT EXISTS note_exports (
  id bigserial PRIMARY KEY,
  user_id varchar(36) NOT NULL REFERENCES users(id),
  idempotency_key varchar(128) NOT NULL,
  note_id bigint NOT NULL,
  transaction_id bigint NOT NULL REFERENCES user_credit_transactions(id),
  filename varchar(255) NOT NULL,
  content text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS uniq_note_export_request ON note_exports(user_id, idempotency_key);
CREATE UNIQUE INDEX IF NOT EXISTS idx_note_exports_transaction_id ON note_exports(transaction_id);
CREATE INDEX IF NOT EXISTS idx_note_exports_note_id ON note_exports(note_id);

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

INSERT INTO user_credit_transactions(user_id, kind, amount, balance_after, source, source_id, reason)
SELECT id, 'opening', credits, credits, 'opening', id, 'Existing balance migrated; historical grant keys preserved'
FROM users WHERE credits_version = 0 AND credits > 0;

INSERT INTO user_credit_lots(user_id, transaction_id, amount, remaining)
SELECT u.id, t.id, u.credits, u.credits
FROM users u JOIN user_credit_transactions t ON t.source = 'opening' AND t.source_id = u.id
WHERE u.credits_version = 0 AND u.credits > 0;

UPDATE users SET credits_version = 1 WHERE credits_version = 0;
COMMIT;
