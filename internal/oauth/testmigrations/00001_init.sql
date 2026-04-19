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
    created_by          TEXT,
    created_at          TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at          TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
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
    revoked_at              TIMESTAMP
);

CREATE TABLE revoked_jti (
    jti         TEXT     PRIMARY KEY,
    revoked_at  DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    expires_at  DATETIME NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS revoked_jti;
DROP TABLE IF EXISTS oauth_tokens;
DROP TABLE IF EXISTS oauth_authorization_codes;
DROP TABLE IF EXISTS agents;
DROP TABLE IF EXISTS users;
