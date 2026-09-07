BEGIN;

CREATE TABLE IF NOT EXISTS auth_token_revocations (
    token_hash char(64) PRIMARY KEY,
    expires_at timestamptz NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_auth_token_revocations_expires_at
    ON auth_token_revocations (expires_at);

COMMIT;
