-- +goose Up

-- Token Vault: OAuth providers users can connect (Google, Slack, GitHub, etc.)
CREATE TABLE IF NOT EXISTS vault_providers (
    id                  TEXT PRIMARY KEY,                         -- vp_<nanoid>
    name                TEXT NOT NULL UNIQUE,                     -- "google_calendar", "slack", "github"
    display_name        TEXT NOT NULL,                            -- "Google Calendar"
    auth_url            TEXT NOT NULL,
    token_url           TEXT NOT NULL,
    client_id           TEXT NOT NULL,
    client_secret_enc   TEXT NOT NULL,                            -- AES-256-GCM via FieldEncryptor (enc::<b64>)
    scopes              TEXT NOT NULL DEFAULT '[]',               -- JSON array: default scopes
    icon_url            TEXT,
    active              INTEGER NOT NULL DEFAULT 1,
    created_at          TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at          TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

-- Token Vault: per-user OAuth connections (access/refresh tokens for a given provider)
CREATE TABLE IF NOT EXISTS vault_connections (
    id                  TEXT PRIMARY KEY,                         -- vc_<nanoid>
    provider_id         TEXT NOT NULL REFERENCES vault_providers(id) ON DELETE CASCADE,
    user_id             TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    access_token_enc    TEXT NOT NULL,
    refresh_token_enc   TEXT,                                     -- may be NULL
    token_type          TEXT NOT NULL DEFAULT 'Bearer',
    scopes              TEXT NOT NULL DEFAULT '[]',               -- JSON array
    expires_at          TIMESTAMP,
    metadata            TEXT NOT NULL DEFAULT '{}',               -- JSON object
    needs_reauth        INTEGER NOT NULL DEFAULT 0,
    last_refreshed_at   TIMESTAMP,
    created_at          TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at          TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    UNIQUE(provider_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_vault_connections_user     ON vault_connections(user_id);
CREATE INDEX IF NOT EXISTS idx_vault_connections_provider ON vault_connections(provider_id);

-- +goose Down

DROP INDEX IF EXISTS idx_vault_connections_provider;
DROP INDEX IF EXISTS idx_vault_connections_user;
DROP TABLE IF EXISTS vault_connections;
DROP TABLE IF EXISTS vault_providers;
