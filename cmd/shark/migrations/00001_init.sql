-- +goose Up

-- Core tables

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
    updated_at    TEXT NOT NULL
);

CREATE TABLE sessions (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ip         TEXT,
    user_agent TEXT,
    mfa_passed INTEGER DEFAULT 0,
    auth_method TEXT DEFAULT 'password',
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE TABLE oauth_accounts (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider    TEXT NOT NULL,
    provider_id TEXT NOT NULL,
    email       TEXT,
    access_token  TEXT,
    refresh_token TEXT,
    created_at  TEXT NOT NULL,
    UNIQUE(provider, provider_id)
);

CREATE TABLE migrations (
    id          TEXT PRIMARY KEY,
    source      TEXT NOT NULL,
    status      TEXT NOT NULL,
    users_total INTEGER DEFAULT 0,
    users_imported INTEGER DEFAULT 0,
    errors      TEXT DEFAULT '[]',
    created_at  TEXT NOT NULL,
    completed_at TEXT
);

-- Passkey / WebAuthn tables

CREATE TABLE passkey_credentials (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credential_id   BLOB NOT NULL UNIQUE,
    public_key      BLOB NOT NULL,
    aaguid          TEXT,
    sign_count      INTEGER DEFAULT 0,
    name            TEXT,
    transports      TEXT DEFAULT '[]',
    backed_up       INTEGER DEFAULT 0,
    created_at      TEXT NOT NULL,
    last_used_at    TEXT
);

-- Magic link tables

CREATE TABLE magic_link_tokens (
    id         TEXT PRIMARY KEY,
    email      TEXT NOT NULL,
    token_hash TEXT NOT NULL,
    used       INTEGER DEFAULT 0,
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL
);

-- MFA tables

CREATE TABLE mfa_recovery_codes (
    id      TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code    TEXT NOT NULL,
    used    INTEGER DEFAULT 0,
    created_at TEXT NOT NULL
);

-- RBAC tables

CREATE TABLE roles (
    id          TEXT PRIMARY KEY,
    name        TEXT UNIQUE NOT NULL,
    description TEXT,
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL
);

CREATE TABLE permissions (
    id       TEXT PRIMARY KEY,
    action   TEXT NOT NULL,
    resource TEXT NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE(action, resource)
);

CREATE TABLE role_permissions (
    role_id       TEXT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id TEXT NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE user_roles (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id TEXT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

-- SSO tables

CREATE TABLE sso_connections (
    id           TEXT PRIMARY KEY,
    type         TEXT NOT NULL,
    name         TEXT NOT NULL,
    domain       TEXT,
    saml_idp_url       TEXT,
    saml_idp_cert      TEXT,
    saml_sp_entity_id  TEXT,
    saml_sp_acs_url    TEXT,
    oidc_issuer        TEXT,
    oidc_client_id     TEXT,
    oidc_client_secret TEXT,
    enabled    INTEGER DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE sso_identities (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    connection_id   TEXT NOT NULL REFERENCES sso_connections(id) ON DELETE CASCADE,
    provider_sub    TEXT NOT NULL,
    created_at      TEXT NOT NULL,
    UNIQUE(connection_id, provider_sub)
);

-- M2M API key tables

CREATE TABLE api_keys (
    id           TEXT PRIMARY KEY,
    name         TEXT NOT NULL,
    key_hash     TEXT NOT NULL UNIQUE,
    key_prefix   TEXT NOT NULL,
    scopes       TEXT DEFAULT '[]',
    rate_limit   INTEGER DEFAULT 1000,
    expires_at   TEXT,
    last_used_at TEXT,
    created_at   TEXT NOT NULL,
    revoked_at   TEXT
);

-- Audit log tables

CREATE TABLE audit_logs (
    id         TEXT PRIMARY KEY,
    actor_id   TEXT,
    actor_type TEXT DEFAULT 'user',
    action     TEXT NOT NULL,
    target_type TEXT,
    target_id  TEXT,
    ip         TEXT,
    user_agent TEXT,
    metadata   TEXT DEFAULT '{}',
    status     TEXT DEFAULT 'success',
    created_at TEXT NOT NULL
);

CREATE INDEX idx_audit_logs_actor   ON audit_logs(actor_id, created_at);
CREATE INDEX idx_audit_logs_action  ON audit_logs(action, created_at);
CREATE INDEX idx_audit_logs_target  ON audit_logs(target_id, created_at);
CREATE INDEX idx_audit_logs_created ON audit_logs(created_at);

-- +goose Down

DROP INDEX IF EXISTS idx_audit_logs_created;
DROP INDEX IF EXISTS idx_audit_logs_target;
DROP INDEX IF EXISTS idx_audit_logs_action;
DROP INDEX IF EXISTS idx_audit_logs_actor;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS sso_identities;
DROP TABLE IF EXISTS sso_connections;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS mfa_recovery_codes;
DROP TABLE IF EXISTS magic_link_tokens;
DROP TABLE IF EXISTS passkey_credentials;
DROP TABLE IF EXISTS migrations;
DROP TABLE IF EXISTS oauth_accounts;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
