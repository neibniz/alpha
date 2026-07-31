CREATE TABLE IF NOT EXISTS forwarder_instances (
    id BIGSERIAL PRIMARY KEY,
    instance_id UUID NOT NULL UNIQUE,
    boot_id UUID NOT NULL,
    name VARCHAR(100) NOT NULL DEFAULT '',
    hostname VARCHAR(255) NOT NULL DEFAULT '',
    public_ip VARCHAR(64) NOT NULL DEFAULT '',
    version VARCHAR(64) NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ NOT NULL,
    first_seen_at TIMESTAMPTZ NOT NULL,
    last_seen_at TIMESTAMPTZ NOT NULL,
    desired_version VARCHAR(64) NOT NULL DEFAULT '',
    desired_hash VARCHAR(128) NOT NULL DEFAULT '',
    applied_version VARCHAR(64) NOT NULL DEFAULT '',
    applied_hash VARCHAR(128) NOT NULL DEFAULT '',
    apply_status VARCHAR(32) NOT NULL DEFAULT 'pending',
    apply_error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_forwarder_instances_public_ip
    ON forwarder_instances (public_ip);

CREATE INDEX IF NOT EXISTS idx_forwarder_instances_last_seen
    ON forwarder_instances (last_seen_at DESC);
