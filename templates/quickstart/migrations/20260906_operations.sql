-- Host-owned operator settings, audit and private uploads.
-- Review and apply to YOUR deployment explicitly; this file never touches
-- provider tables, roles or user balances. Safe to rerun.
BEGIN;

CREATE TABLE IF NOT EXISTS site_settings (
    id bigint PRIMARY KEY CHECK (id = 1),
    brand_name varchar(100) NOT NULL,
    description varchar(500) NOT NULL,
    support_email varchar(255) NOT NULL DEFAULT '',
    support_url varchar(1024) NOT NULL DEFAULT '',
    export_credit_cost bigint NOT NULL CHECK (export_credit_cost BETWEEN 1 AND 1000000),
    updated_at timestamptz
);

CREATE TABLE IF NOT EXISTS operations_audit_events (
    id bigserial PRIMARY KEY,
    actor_id varchar(36) NOT NULL,
    idempotency_key varchar(128) NOT NULL,
    action varchar(64) NOT NULL,
    target_id varchar(255) NOT NULL,
    reason varchar(500) NOT NULL,
    request_hash varchar(64) NOT NULL,
    status varchar(32) NOT NULL,
    details text,
    created_at timestamptz,
    updated_at timestamptz,
    CONSTRAINT uniq_operations_actor_key UNIQUE(actor_id, idempotency_key)
);
CREATE INDEX IF NOT EXISTS idx_operations_audit_events_actor_id ON operations_audit_events(actor_id);
CREATE INDEX IF NOT EXISTS idx_operations_audit_events_action ON operations_audit_events(action);

CREATE TABLE IF NOT EXISTS private_image_uploads (
    id varchar(36) PRIMARY KEY,
    user_id varchar(36) NOT NULL,
    storage_key varchar(255) NOT NULL,
    provider varchar(16) NOT NULL,
    content_type varchar(32) NOT NULL,
    size bigint NOT NULL CHECK (size > 0 AND size <= 5242880),
    filename varchar(100) NOT NULL,
    created_at timestamptz
);
CREATE INDEX IF NOT EXISTS idx_private_image_uploads_user_id ON private_image_uploads(user_id);

COMMIT;
