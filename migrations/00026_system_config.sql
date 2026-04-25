-- +goose Up
-- system_config: single-row table holding all runtime config as a JSON blob.
-- id is constrained to 1 so only one row can ever exist.
CREATE TABLE system_config (
    id         INTEGER PRIMARY KEY CHECK (id = 1),
    payload    TEXT    NOT NULL DEFAULT '{}',
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
INSERT OR IGNORE INTO system_config (id, payload) VALUES (1, '{}');

-- secrets: named key-value store for session secret, JWT signing keys,
-- admin API key, etc. Populated by Phase B first-boot flow.
CREATE TABLE secrets (
    name       TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    rotated_at TIMESTAMP
);

-- +goose Down
DROP TABLE IF EXISTS secrets;
DROP TABLE IF EXISTS system_config;
