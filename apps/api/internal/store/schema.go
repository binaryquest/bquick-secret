package store

const schemaSQL = `
CREATE TABLE IF NOT EXISTS secrets (
    id BIGSERIAL PRIMARY KEY,
    public_id TEXT NOT NULL UNIQUE,
    encrypted_payload BYTEA NOT NULL,
    iv BYTEA NOT NULL,
    algorithm TEXT NOT NULL DEFAULT 'AES-256-GCM',
    version INT NOT NULL DEFAULT 1,
    expires_at TIMESTAMPTZ NOT NULL,
    one_time BOOLEAN NOT NULL DEFAULT TRUE,
    consumed_at TIMESTAMPTZ NULL,
    deleted_at TIMESTAMPTZ NULL,
    sender_email_hash TEXT NOT NULL,
    recipient_email_provided BOOLEAN NOT NULL DEFAULT FALSE,
    manual_link_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    passphrase_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    delete_token_hash TEXT NULL,
    payload_size_bytes INT NOT NULL DEFAULT 0,
    notify_sender_on_reveal BOOLEAN NOT NULL DEFAULT FALSE,
    sender_notify_email TEXT NULL,
    sender_notified_at TIMESTAMPTZ NULL,
    wrapped_key BYTEA NULL,
    wrapping_iv BYTEA NULL,
    kdf_salt BYTEA NULL,
    kdf_iterations INT NULL,
    kdf_algorithm TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE secrets ADD COLUMN IF NOT EXISTS wrapped_key BYTEA NULL;
ALTER TABLE secrets ADD COLUMN IF NOT EXISTS wrapping_iv BYTEA NULL;
ALTER TABLE secrets ADD COLUMN IF NOT EXISTS kdf_salt BYTEA NULL;
ALTER TABLE secrets ADD COLUMN IF NOT EXISTS kdf_iterations INT NULL;
ALTER TABLE secrets ADD COLUMN IF NOT EXISTS kdf_algorithm TEXT NULL;
ALTER TABLE secrets ADD COLUMN IF NOT EXISTS notify_sender_on_reveal BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE secrets ADD COLUMN IF NOT EXISTS sender_notify_email TEXT NULL;
ALTER TABLE secrets ADD COLUMN IF NOT EXISTS sender_notified_at TIMESTAMPTZ NULL;

CREATE INDEX IF NOT EXISTS idx_secrets_public_id ON secrets(public_id);
CREATE INDEX IF NOT EXISTS idx_secrets_expires_at ON secrets(expires_at);
CREATE INDEX IF NOT EXISTS idx_secrets_cleanup ON secrets(expires_at, deleted_at, consumed_at);

CREATE TABLE IF NOT EXISTS daily_stats (
    stat_date DATE PRIMARY KEY,
    secrets_created_count BIGINT NOT NULL DEFAULT 0,
    secrets_opened_count BIGINT NOT NULL DEFAULT 0,
    secrets_expired_count BIGINT NOT NULL DEFAULT 0,
    secrets_deleted_count BIGINT NOT NULL DEFAULT 0,
    emails_sent_count BIGINT NOT NULL DEFAULT 0,
    manual_links_created_count BIGINT NOT NULL DEFAULT 0,
    passphrase_enabled_count BIGINT NOT NULL DEFAULT 0,
    one_time_enabled_count BIGINT NOT NULL DEFAULT 0,
    files_uploaded_count BIGINT NOT NULL DEFAULT 0,
    total_encrypted_file_bytes BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS rate_limit_buckets (
    bucket_key TEXT PRIMARY KEY,
    count INT NOT NULL DEFAULT 0,
    resets_at TIMESTAMPTZ NOT NULL
);
`
