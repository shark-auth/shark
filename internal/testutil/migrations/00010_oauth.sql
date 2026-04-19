-- +goose Up
-- 00010: Add OAuth 2.1 tables (agents, tokens, consents, auth codes, device codes, DCR clients).
-- Mirrors the production cmd/shark/migrations/00010_oauth.sql schema and the
-- internal/oauth/testmigrations schema used by the fosite adapter tests.

CREATE TABLE IF NOT EXISTS agents (
    id                  TEXT PRIMARY KEY,
    name                TEXT NOT NULL,
    description         TEXT DEFAULT '',
    client_id           TEXT UNIQUE NOT NULL,
    client_secret_hash  TEXT,
    client_type         TEXT NOT NULL DEFAULT 'confidential'
                            CHECK (client_type IN ('confidential', 'public')),
    auth_method         TEXT NOT NULL DEFAULT 'client_secret_basic'
                            CHECK (auth_method IN ('client_secret_basic', 'client_secret_post', 'private_key_jwt', 'none')),
    jwks                TEXT,
    jwks_uri            TEXT,
    redirect_uris       TEXT NOT NULL DEFAULT '[]',
    grant_types         TEXT NOT NULL DEFAULT '["client_credentials"]',
    response_types      TEXT NOT NULL DEFAULT '["code"]',
    scopes              TEXT NOT NULL DEFAULT '[]',
    token_lifetime      INTEGER DEFAULT 900,
    metadata            TEXT DEFAULT '{}',
    logo_uri            TEXT,
    homepage_uri        TEXT,
    active              INTEGER NOT NULL DEFAULT 1,
    created_by          TEXT REFERENCES users(id),
    created_at          TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at          TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
CREATE INDEX IF NOT EXISTS idx_agents_client_id ON agents(client_id);
CREATE INDEX IF NOT EXISTS idx_agents_active ON agents(active);

CREATE TABLE IF NOT EXISTS oauth_authorization_codes (
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

CREATE TABLE IF NOT EXISTS oauth_tokens (
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
    revoked_at              TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_oauth_tokens_agent_id ON oauth_tokens(agent_id);
CREATE INDEX IF NOT EXISTS idx_oauth_tokens_client_id ON oauth_tokens(client_id);

CREATE TABLE IF NOT EXISTS oauth_consents (
    id                      TEXT PRIMARY KEY,
    user_id                 TEXT NOT NULL REFERENCES users(id),
    client_id               TEXT NOT NULL,
    scope                   TEXT NOT NULL,
    authorization_details   TEXT,
    granted_at              TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    expires_at              TIMESTAMP,
    revoked_at              TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_oauth_consents_user_client
    ON oauth_consents(user_id, client_id) WHERE revoked_at IS NULL;

CREATE TABLE IF NOT EXISTS oauth_device_codes (
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
CREATE INDEX IF NOT EXISTS idx_oauth_device_codes_user_code ON oauth_device_codes(user_code);
CREATE INDEX IF NOT EXISTS idx_oauth_device_codes_expires_at ON oauth_device_codes(expires_at);

CREATE TABLE IF NOT EXISTS oauth_dcr_clients (
    client_id               TEXT PRIMARY KEY,
    registration_token_hash TEXT UNIQUE NOT NULL,
    client_metadata         TEXT NOT NULL,
    created_at              TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    expires_at              TIMESTAMP
);

-- +goose Down
DROP TABLE IF EXISTS oauth_dcr_clients;
DROP INDEX IF EXISTS idx_oauth_device_codes_expires_at;
DROP INDEX IF EXISTS idx_oauth_device_codes_user_code;
DROP TABLE IF EXISTS oauth_device_codes;
DROP INDEX IF EXISTS idx_oauth_consents_user_client;
DROP TABLE IF EXISTS oauth_consents;
DROP INDEX IF EXISTS idx_oauth_tokens_client_id;
DROP INDEX IF EXISTS idx_oauth_tokens_agent_id;
DROP TABLE IF EXISTS oauth_tokens;
DROP TABLE IF EXISTS oauth_authorization_codes;
DROP INDEX IF EXISTS idx_agents_active;
DROP INDEX IF EXISTS idx_agents_client_id;
DROP TABLE IF EXISTS agents;
