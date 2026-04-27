-- +goose Up
CREATE TABLE may_act_grants (
    id          TEXT PRIMARY KEY,
    from_id     TEXT NOT NULL,
    to_id       TEXT NOT NULL,
    max_hops    INTEGER NOT NULL DEFAULT 1,
    scopes      TEXT NOT NULL DEFAULT '[]',
    expires_at  TIMESTAMP,
    revoked_at  TIMESTAMP,
    created_by  TEXT,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_may_act_grants_from ON may_act_grants(from_id);
CREATE INDEX idx_may_act_grants_to   ON may_act_grants(to_id);

-- +goose Down
DROP INDEX IF EXISTS idx_may_act_grants_to;
DROP INDEX IF EXISTS idx_may_act_grants_from;
DROP TABLE IF EXISTS may_act_grants;
