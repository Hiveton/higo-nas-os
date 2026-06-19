-- server-cloud Postgres schema. Applied idempotently on Connect.

CREATE TABLE IF NOT EXISTS cloud_users (
    id           TEXT PRIMARY KEY,
    display_name TEXT NOT NULL DEFAULT '',
    avatar       TEXT NOT NULL DEFAULT '',
    status       TEXT NOT NULL DEFAULT 'active',
    created_at   TIMESTAMPTZ NOT NULL,
    updated_at   TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS cloud_identities (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES cloud_users(id) ON DELETE CASCADE,
    type        TEXT NOT NULL,
    principal   TEXT NOT NULL,
    secret_hash TEXT NOT NULL DEFAULT '',
    verified_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL,
    UNIQUE (type, principal)
);
CREATE INDEX IF NOT EXISTS idx_identities_user ON cloud_identities(user_id);

CREATE TABLE IF NOT EXISTS cloud_refresh_tokens (
    token_hash TEXT PRIMARY KEY,
    id         TEXT NOT NULL,
    user_id    TEXT NOT NULL REFERENCES cloud_users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_refresh_user ON cloud_refresh_tokens(user_id);

CREATE TABLE IF NOT EXISTS cloud_devices (
    id               TEXT PRIMARY KEY,
    serial           TEXT NOT NULL UNIQUE,
    secret_hash      TEXT NOT NULL,
    model            TEXT NOT NULL DEFAULT '',
    version          TEXT NOT NULL DEFAULT '',
    agent_public_key TEXT NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL,
    last_seen_at     TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS cloud_bindings (
    user_id       TEXT NOT NULL REFERENCES cloud_users(id) ON DELETE CASCADE,
    device_id     TEXT NOT NULL REFERENCES cloud_devices(id) ON DELETE CASCADE,
    id            TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'user',
    local_user_id TEXT NOT NULL DEFAULT '',
    label         TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (user_id, device_id)
);
CREATE INDEX IF NOT EXISTS idx_bindings_device ON cloud_bindings(device_id);

CREATE TABLE IF NOT EXISTS cloud_pairing_challenges (
    kind       TEXT NOT NULL,
    secret     TEXT NOT NULL,
    device_id  TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    claimed    BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (kind, secret)
);

CREATE TABLE IF NOT EXISTS cloud_push_tokens (
    user_id    TEXT NOT NULL REFERENCES cloud_users(id) ON DELETE CASCADE,
    token      TEXT NOT NULL,
    id         TEXT NOT NULL,
    platform   TEXT NOT NULL DEFAULT 'ios',
    created_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (user_id, token)
);

CREATE TABLE IF NOT EXISTS cloud_audit (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL DEFAULT '',
    device_id  TEXT NOT NULL DEFAULT '',
    action     TEXT NOT NULL,
    detail     TEXT NOT NULL DEFAULT '',
    ip         TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_audit_user ON cloud_audit(user_id, created_at DESC);
