-- +goose Up

-- Auth flows: admin-defined pipelines that run at specific auth trigger points
-- (signup, login, password_reset, magic_link, oauth_callback). Each flow is an
-- ordered list of steps encoded as JSON. Higher priority runs first.
CREATE TABLE IF NOT EXISTS auth_flows (
    id          TEXT PRIMARY KEY,                         -- flow_<hex>
    name        TEXT NOT NULL,
    trigger     TEXT NOT NULL,                            -- signup | login | password_reset | magic_link | oauth_callback
    steps       TEXT NOT NULL DEFAULT '[]',               -- JSON array of step definitions
    enabled     INTEGER NOT NULL DEFAULT 1,
    priority    INTEGER NOT NULL DEFAULT 0,               -- higher = checked first
    conditions  TEXT NOT NULL DEFAULT '{}',               -- JSON: when this flow applies
    created_at  TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at  TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
CREATE INDEX IF NOT EXISTS idx_auth_flows_trigger  ON auth_flows(trigger);
CREATE INDEX IF NOT EXISTS idx_auth_flows_priority ON auth_flows(priority DESC);

-- Auth flow runs: history of every flow evaluation, with outcome + timing so
-- the Flow Builder dashboard "History" tab can show recent activity.
CREATE TABLE IF NOT EXISTS auth_flow_runs (
    id              TEXT PRIMARY KEY,                     -- fr_<hex>
    flow_id         TEXT NOT NULL REFERENCES auth_flows(id) ON DELETE CASCADE,
    user_id         TEXT,                                 -- nullable; pre-signup may not have a user yet
    trigger         TEXT NOT NULL,
    outcome         TEXT NOT NULL,                        -- continue | block | redirect | error
    blocked_at_step INTEGER,                              -- step index if blocked; NULL otherwise
    reason          TEXT,                                 -- human-readable note (block/error message)
    metadata        TEXT NOT NULL DEFAULT '{}',           -- JSON: step timeline, inputs, etc
    started_at      TIMESTAMP NOT NULL,
    finished_at     TIMESTAMP NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_auth_flow_runs_flow ON auth_flow_runs(flow_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_auth_flow_runs_user ON auth_flow_runs(user_id, started_at DESC);

-- +goose Down

DROP INDEX IF EXISTS idx_auth_flow_runs_user;
DROP INDEX IF EXISTS idx_auth_flow_runs_flow;
DROP TABLE IF EXISTS auth_flow_runs;
DROP INDEX IF EXISTS idx_auth_flows_priority;
DROP INDEX IF EXISTS idx_auth_flows_trigger;
DROP TABLE IF EXISTS auth_flows;
