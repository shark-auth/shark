-- +goose Up

-- Minimal schema for fosite storage adapter tests.

CREATE TABLE users (
    id            TEXT PRIMARY KEY,
    email         TEXT UNIQUE NOT NULL,
    email_verified INTEGER DEFAULT 0,
    password_hash TEXT,
    hash_type     TEXT DEFAULT 'argon2id',
    name          TEXT,
    avatar_url    TEXT,
    mfa_enabled   INTEGER DEFAULT 0,
    mfa_secret    TEXT,
    mfa_verified  INTEGER DEFAULT 0,
    metadata      TEXT DEFAULT '{}',
    created_at    TEXT NOT NULL,
    updated_at    TEXT NOT NULL,
    last_login_at TEXT
);

CREATE TABLE agents (
    id                    TEXT PRIMARY KEY,
    name                  TEXT NOT NULL,
    description           TEXT DEFAULT '',
    client_id             TEXT UNIQUE NOT NULL,
    client_secret_hash    TEXT,
    client_type           TEXT NOT NULL DEFAULT 'confidential'
                              CHECK (client_type IN ('confidential', 'public')),
    auth_method           TEXT NOT NULL DEFAULT 'client_secret_basic'
                              CHECK (auth_method IN ('client_secret_basic', 'client_secret_post', 'private_key_jwt', 'none')),
    jwks                  TEXT,
    jwks_uri              TEXT,
    redirect_uris         TEXT NOT NULL DEFAULT '[]',
    grant_types           TEXT NOT NULL DEFAULT '["client_credentials"]',
    response_types        TEXT NOT NULL DEFAULT '["code"]',
    scopes                TEXT NOT NULL DEFAULT '[]',
    token_lifetime        INTEGER DEFAULT 900,
    metadata              TEXT DEFAULT '{}',
    logo_uri              TEXT,
    homepage_uri          TEXT,
    active                INTEGER NOT NULL DEFAULT 1,
    created_by            TEXT,
    created_at            TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at            TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    old_secret_hash       TEXT,
    old_secret_expires_at TEXT
);

CREATE TABLE oauth_authorization_codes (
    code_hash               TEXT PRIMARY KEY,
    client_id               TEXT NOT NULL,
    user_id                 TEXT NOT NULL REFERENCES users(id),
    redirect_uri            TEXT NOT NULL,
    scope                   TEXT NOT NULL DEFAULT '',
    code_challenge          TEXT NOT NULL,
    code_challenge_method   TEXT NOT NULL DEFAULT 'S256',
    resource                TEXT,
    authorization_details   TEXT,
    nonce                   TEXT,
    expires_at              TIMESTAMP NOT NULL,
    created_at              TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE TABLE oauth_pkce_sessions (
    signature_hash         TEXT PRIMARY KEY,
    code_challenge         TEXT NOT NULL,
    code_challenge_method  TEXT NOT NULL DEFAULT 'S256',
    client_id              TEXT NOT NULL,
    expires_at             TIMESTAMP NOT NULL,
    created_at             TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE TABLE oauth_tokens (
    id                      TEXT PRIMARY KEY,
    jti                     TEXT UNIQUE NOT NULL,
    client_id               TEXT NOT NULL,
    agent_id                TEXT,
    user_id                 TEXT,
    token_type              TEXT NOT NULL
                                CHECK (token_type IN ('access', 'refresh')),
    token_hash              TEXT UNIQUE NOT NULL,
    scope                   TEXT NOT NULL DEFAULT '',
    audience                TEXT,
    authorization_details   TEXT,
    dpop_jkt                TEXT,
    delegation_subject      TEXT,
    delegation_actor        TEXT,
    family_id               TEXT,
    expires_at              TIMESTAMP NOT NULL,
    created_at              TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    revoked_at              TIMESTAMP,
    request_id              TEXT
);
CREATE INDEX idx_oauth_tokens_request_id_type ON oauth_tokens(request_id, token_type);

CREATE TABLE revoked_jti (
    jti         TEXT     PRIMARY KEY,
    revoked_at  DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    expires_at  DATETIME NOT NULL
);

CREATE TABLE jwt_signing_keys (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    kid             TEXT    NOT NULL UNIQUE,
    algorithm       TEXT    NOT NULL DEFAULT 'RS256',
    public_key_pem  TEXT    NOT NULL,
    private_key_pem TEXT    NOT NULL,
    created_at      DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    rotated_at      DATETIME,
    status          TEXT    NOT NULL DEFAULT 'active'
                    CHECK (status IN ('active', 'retired'))
);

CREATE INDEX idx_jwt_signing_keys_status ON jwt_signing_keys(status);

CREATE TABLE oauth_consents (
    id                      TEXT PRIMARY KEY,
    user_id                 TEXT NOT NULL REFERENCES users(id),
    client_id               TEXT NOT NULL,
    scope                   TEXT NOT NULL,
    authorization_details   TEXT,
    granted_at              TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    expires_at              TIMESTAMP,
    revoked_at              TIMESTAMP
);
CREATE UNIQUE INDEX idx_oauth_consents_user_client
    ON oauth_consents(user_id, client_id) WHERE revoked_at IS NULL;

CREATE TABLE oauth_dcr_clients (
    client_id               TEXT PRIMARY KEY,
    registration_token_hash TEXT UNIQUE NOT NULL,
    client_metadata         TEXT NOT NULL,
    created_at              TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    expires_at              TIMESTAMP
);

CREATE TABLE oauth_device_codes (
    device_code_hash    TEXT PRIMARY KEY,
    user_code           TEXT UNIQUE NOT NULL,
    client_id           TEXT NOT NULL,
    scope               TEXT NOT NULL DEFAULT '',
    resource            TEXT,
    user_id             TEXT,
    status              TEXT NOT NULL DEFAULT 'pending'
                            CHECK (status IN ('pending', 'approved', 'denied', 'expired')),
    last_polled_at      TIMESTAMP,
    poll_interval       INTEGER NOT NULL DEFAULT 5,
    expires_at          TIMESTAMP NOT NULL,
    created_at          TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX idx_oauth_device_codes_user_code ON oauth_device_codes(user_code);
CREATE INDEX idx_oauth_device_codes_expires_at ON oauth_device_codes(expires_at);

CREATE TABLE audit_logs (
    id          TEXT PRIMARY KEY,
    actor_id    TEXT,
    actor_type  TEXT DEFAULT 'user',
    action      TEXT NOT NULL,
    target_type TEXT,
    target_id   TEXT,
    ip          TEXT,
    user_agent  TEXT,
    metadata    TEXT DEFAULT '{}',
    status      TEXT DEFAULT 'success',
    created_at  TEXT NOT NULL
);

CREATE TABLE api_keys (
    id           TEXT PRIMARY KEY,
    name         TEXT NOT NULL,
    key_hash     TEXT UNIQUE NOT NULL,
    key_prefix   TEXT NOT NULL,
    key_suffix   TEXT NOT NULL DEFAULT '',
    scopes       TEXT NOT NULL DEFAULT '[]',
    rate_limit   INTEGER NOT NULL DEFAULT 1000,
    expires_at   TEXT,
    last_used_at TEXT,
    created_at   TEXT NOT NULL,
    revoked_at   TEXT
);

-- +goose Down
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS audit_logs;
DROP INDEX IF EXISTS idx_oauth_device_codes_expires_at;
DROP INDEX IF EXISTS idx_oauth_device_codes_user_code;
DROP TABLE IF EXISTS oauth_device_codes;
DROP TABLE IF EXISTS oauth_dcr_clients;
DROP INDEX IF EXISTS idx_oauth_consents_user_client;
DROP TABLE IF EXISTS oauth_consents;
DROP INDEX IF EXISTS idx_jwt_signing_keys_status;
DROP TABLE IF EXISTS jwt_signing_keys;
DROP TABLE IF EXISTS revoked_jti;
DROP TABLE IF EXISTS oauth_tokens;
DROP TABLE IF EXISTS oauth_authorization_codes;
DROP TABLE IF EXISTS agents;
DROP TABLE IF EXISTS users;
