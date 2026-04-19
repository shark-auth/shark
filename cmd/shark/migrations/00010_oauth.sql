-- +goose Up

-- OAuth 2.1 agents (clients with agent identity)
CREATE TABLE IF NOT EXISTS agents (
    id                  TEXT PRIMARY KEY,                         -- agent_<nanoid>
    name                TEXT NOT NULL,
    description         TEXT DEFAULT '',
    client_id           TEXT UNIQUE NOT NULL,                    -- public identifier
    client_secret_hash  TEXT,                                    -- SHA-256, NULL for public clients
    client_type         TEXT NOT NULL DEFAULT 'confidential'
                            CHECK (client_type IN ('confidential', 'public')),
    auth_method         TEXT NOT NULL DEFAULT 'client_secret_basic'
                            CHECK (auth_method IN ('client_secret_basic', 'client_secret_post', 'private_key_jwt', 'none')),
    jwks                TEXT,                                    -- JSON: public keys for private_key_jwt
    jwks_uri            TEXT,
    redirect_uris       TEXT NOT NULL DEFAULT '[]',              -- JSON array
    grant_types         TEXT NOT NULL DEFAULT '["client_credentials"]', -- JSON array
    response_types      TEXT NOT NULL DEFAULT '["code"]',        -- JSON array
    scopes              TEXT NOT NULL DEFAULT '[]',              -- JSON array
    token_lifetime      INTEGER DEFAULT 900,                     -- seconds, default 15min
    metadata            TEXT DEFAULT '{}',
    logo_uri            TEXT,
    homepage_uri        TEXT,
    active              INTEGER NOT NULL DEFAULT 1,
    created_by          TEXT REFERENCES users(id),
    created_at          TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at          TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
CREATE INDEX idx_agents_client_id ON agents(client_id);
CREATE INDEX idx_agents_active    ON agents(active);

-- Short-lived, single-use authorization codes
CREATE TABLE IF NOT EXISTS oauth_authorization_codes (
    code_hash               TEXT PRIMARY KEY,                    -- SHA-256
    client_id               TEXT NOT NULL,
    user_id                 TEXT NOT NULL REFERENCES users(id),
    redirect_uri            TEXT NOT NULL,
    scope                   TEXT NOT NULL DEFAULT '',
    code_challenge          TEXT NOT NULL,                       -- PKCE
    code_challenge_method   TEXT NOT NULL DEFAULT 'S256',
    resource                TEXT,                                -- RFC 8707
    authorization_details   TEXT,                                -- RFC 9396 RAR, JSON
    nonce                   TEXT,
    expires_at              TIMESTAMP NOT NULL,
    created_at              TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

-- Access + refresh tokens tracked for revocation/introspection
CREATE TABLE IF NOT EXISTS oauth_tokens (
    id                      TEXT PRIMARY KEY,
    jti                     TEXT UNIQUE NOT NULL,
    client_id               TEXT NOT NULL,
    agent_id                TEXT REFERENCES agents(id),
    user_id                 TEXT REFERENCES users(id),           -- NULL for client_credentials
    token_type              TEXT NOT NULL
                                CHECK (token_type IN ('access', 'refresh')),
    token_hash              TEXT UNIQUE NOT NULL,                -- SHA-256
    scope                   TEXT NOT NULL DEFAULT '',
    audience                TEXT,
    authorization_details   TEXT,
    dpop_jkt                TEXT,                                -- DPoP key thumbprint
    delegation_subject      TEXT,
    delegation_actor        TEXT,
    family_id               TEXT,                                -- refresh token family for rotation
    expires_at              TIMESTAMP NOT NULL,
    created_at              TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    revoked_at              TIMESTAMP
);
CREATE INDEX idx_oauth_tokens_family_id  ON oauth_tokens(family_id);
CREATE INDEX idx_oauth_tokens_client_id  ON oauth_tokens(client_id);
CREATE INDEX idx_oauth_tokens_jti        ON oauth_tokens(jti);
CREATE INDEX idx_oauth_tokens_user_id    ON oauth_tokens(user_id);

-- User consent records per-agent
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
CREATE UNIQUE INDEX idx_oauth_consents_user_client
    ON oauth_consents(user_id, client_id) WHERE revoked_at IS NULL;

-- RFC 8628 device authorization codes
CREATE TABLE IF NOT EXISTS oauth_device_codes (
    device_code_hash    TEXT PRIMARY KEY,                        -- SHA-256
    user_code           TEXT UNIQUE NOT NULL,                    -- SHARK-XXXX format
    client_id           TEXT NOT NULL,
    scope               TEXT NOT NULL DEFAULT '',
    resource            TEXT,
    user_id             TEXT REFERENCES users(id),               -- set when approved
    status              TEXT NOT NULL DEFAULT 'pending'
                            CHECK (status IN ('pending', 'approved', 'denied', 'expired')),
    last_polled_at      TIMESTAMP,
    poll_interval       INTEGER NOT NULL DEFAULT 5,
    expires_at          TIMESTAMP NOT NULL,
    created_at          TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

-- Dynamic Client Registration (RFC 7591)
CREATE TABLE IF NOT EXISTS oauth_dcr_clients (
    client_id               TEXT PRIMARY KEY,
    registration_token_hash TEXT UNIQUE NOT NULL,
    client_metadata         TEXT NOT NULL,                       -- full JSON
    created_at              TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    expires_at              TIMESTAMP
);

-- +goose Down

DROP TABLE IF EXISTS oauth_dcr_clients;
DROP TABLE IF EXISTS oauth_device_codes;
DROP INDEX IF EXISTS idx_oauth_consents_user_client;
DROP TABLE IF EXISTS oauth_consents;
DROP INDEX IF EXISTS idx_oauth_tokens_user_id;
DROP INDEX IF EXISTS idx_oauth_tokens_jti;
DROP INDEX IF EXISTS idx_oauth_tokens_client_id;
DROP INDEX IF EXISTS idx_oauth_tokens_family_id;
DROP TABLE IF EXISTS oauth_tokens;
DROP TABLE IF EXISTS oauth_authorization_codes;
DROP INDEX IF EXISTS idx_agents_active;
DROP INDEX IF EXISTS idx_agents_client_id;
DROP TABLE IF EXISTS agents;
