-- Host-owned Resend and Stripe overrides, loaded before providers are assembled.
-- Review and apply explicitly to your chosen deployment; safe to rerun.
-- Payload deliberately contains PLAINTEXT provider credentials. It is private
-- server data, never a public settings value, audit snapshot, or SQL log field.
BEGIN;

CREATE TABLE IF NOT EXISTS service_provider_settings (
    provider varchar(16) PRIMARY KEY CHECK (provider IN ('resend', 'stripe')),
    source varchar(16) NOT NULL CHECK (source IN ('environment', 'database')),
    version bigint NOT NULL CHECK (version >= 1),
    payload text NOT NULL,
    updated_at timestamptz
);

COMMIT;
