-- +goose Up
-- +goose NO TRANSACTION
-- SQLite does not support ALTER TABLE ADD CONSTRAINT. We must recreate the tables
-- with ON DELETE CASCADE / ON DELETE SET NULL to prevent 500 errors during user deletion.

PRAGMA foreign_keys = OFF;

-- 1. agents (created_by -> ON DELETE SET NULL)
CREATE TABLE agents_new (
    id                  TEXT PRIMARY KEY,
    name                TEXT NOT NULL,
    description         TEXT DEFAULT '',
    client_id           TEXT UNIQUE NOT NULL,
    client_secret_hash  TEXT,
    client_type         TEXT NOT NULL DEFAULT 'confidential' CHECK (client_type IN ('confidential', 'public')),
    auth_method         TEXT NOT NULL DEFAULT 'client_secret_basic' CHECK (auth_method IN ('client_secret_basic', 'client_secret_post', 'private_key_jwt', 'none')),
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
    created_by          TEXT REFERENCES users(id) ON DELETE SET NULL,
    created_at          TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at          TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    old_secret_hash     TEXT,
    old_secret_expires_at TEXT
);
INSERT INTO agents_new (id, name, description, client_id, client_secret_hash, client_type, auth_method, jwks, jwks_uri, redirect_uris, grant_types, response_types, scopes, token_lifetime, metadata, logo_uri, homepage_uri, active, created_by, created_at, updated_at, old_secret_hash, old_secret_expires_at)
SELECT id, name, description, client_id, client_secret_hash, client_type, auth_method, jwks, jwks_uri, redirect_uris, grant_types, response_types, scopes, token_lifetime, metadata, logo_uri, homepage_uri, active, created_by, created_at, updated_at, old_secret_hash, old_secret_expires_at FROM agents;
DROP TABLE agents;
ALTER TABLE agents_new RENAME TO agents;
CREATE INDEX idx_agents_client_id ON agents(client_id);
CREATE INDEX idx_agents_active    ON agents(active);

-- 2. oauth_authorization_codes (user_id -> ON DELETE CASCADE)
CREATE TABLE oauth_authorization_codes_new (
    code_hash               TEXT PRIMARY KEY,
    client_id               TEXT NOT NULL,
    user_id                 TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
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
INSERT INTO oauth_authorization_codes_new SELECT * FROM oauth_authorization_codes;
DROP TABLE oauth_authorization_codes;
ALTER TABLE oauth_authorization_codes_new RENAME TO oauth_authorization_codes;

-- 3. oauth_tokens (user_id -> ON DELETE CASCADE)
CREATE TABLE oauth_tokens_new (
    id                      TEXT PRIMARY KEY,
    jti                     TEXT UNIQUE NOT NULL,
    client_id               TEXT NOT NULL,
    agent_id                TEXT REFERENCES agents(id),
    user_id                 TEXT REFERENCES users(id) ON DELETE CASCADE,
    token_type              TEXT NOT NULL CHECK (token_type IN ('access', 'refresh')),
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
INSERT INTO oauth_tokens_new (id, jti, client_id, agent_id, user_id, token_type, token_hash, scope, audience, authorization_details, dpop_jkt, delegation_subject, delegation_actor, family_id, expires_at, created_at, revoked_at, request_id)
SELECT id, jti, client_id, agent_id, user_id, token_type, token_hash, scope, audience, authorization_details, dpop_jkt, delegation_subject, delegation_actor, family_id, expires_at, created_at, revoked_at, request_id FROM oauth_tokens;
DROP TABLE oauth_tokens;
ALTER TABLE oauth_tokens_new RENAME TO oauth_tokens;
CREATE INDEX idx_oauth_tokens_family_id  ON oauth_tokens(family_id);
CREATE INDEX idx_oauth_tokens_client_id  ON oauth_tokens(client_id);
CREATE INDEX idx_oauth_tokens_jti        ON oauth_tokens(jti);
CREATE INDEX idx_oauth_tokens_user_id    ON oauth_tokens(user_id);
CREATE INDEX idx_oauth_tokens_request_id_type ON oauth_tokens(request_id, token_type);

-- 4. oauth_consents (user_id -> ON DELETE CASCADE)
CREATE TABLE oauth_consents_new (
    id                      TEXT PRIMARY KEY,
    user_id                 TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    client_id               TEXT NOT NULL,
    scope                   TEXT NOT NULL,
    authorization_details   TEXT,
    granted_at              TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    expires_at              TIMESTAMP,
    revoked_at              TIMESTAMP
);
INSERT INTO oauth_consents_new SELECT * FROM oauth_consents;
DROP TABLE oauth_consents;
ALTER TABLE oauth_consents_new RENAME TO oauth_consents;
CREATE UNIQUE INDEX idx_oauth_consents_user_client ON oauth_consents(user_id, client_id) WHERE revoked_at IS NULL;

-- 5. oauth_device_codes (user_id -> ON DELETE CASCADE)
CREATE TABLE oauth_device_codes_new (
    device_code_hash    TEXT PRIMARY KEY,
    user_code           TEXT UNIQUE NOT NULL,
    client_id           TEXT NOT NULL,
    scope               TEXT NOT NULL DEFAULT '',
    resource            TEXT,
    user_id             TEXT REFERENCES users(id) ON DELETE CASCADE,
    status              TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'denied', 'expired')),
    last_polled_at      TIMESTAMP,
    poll_interval       INTEGER NOT NULL DEFAULT 5,
    expires_at          TIMESTAMP NOT NULL,
    created_at          TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
INSERT INTO oauth_device_codes_new SELECT * FROM oauth_device_codes;
DROP TABLE oauth_device_codes;
ALTER TABLE oauth_device_codes_new RENAME TO oauth_device_codes;

PRAGMA foreign_keys = ON;

-- +goose Down
-- Reverting is omitted as it is a structural fix for SQLite's lack of ALTER TABLE ADD CONSTRAINT.
