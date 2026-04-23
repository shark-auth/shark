-- +goose Up

-- proxy_rules: runtime-editable override layer for the reverse proxy rule
-- engine. YAML rules stay the bootstrap source of truth; DB rows layer on top
-- and take precedence when the proxy engine is rebuilt (Wave D). Admins
-- manage these via the dashboard's Proxy Rules table.
--
-- name       : human label shown in the dashboard list
-- pattern    : chi-style path pattern (e.g. /api/orgs/{id}, /v1/public/*)
-- methods    : JSON array of HTTP verbs; empty = any
-- require    : requirement string ("authenticated" | "agent" | "role:admin" | ...)
-- allow      : alternative to require; currently only "anonymous"
-- scopes     : JSON array of additional AND'd scopes
-- enabled    : 0 disables the rule without deleting it
-- priority   : higher = evaluated first; YAML rules keep priority 0 by default
CREATE TABLE IF NOT EXISTS proxy_rules (
    id          TEXT PRIMARY KEY,                         -- pxr_<hex>
    name        TEXT NOT NULL,
    pattern     TEXT NOT NULL,
    methods     TEXT NOT NULL DEFAULT '[]',               -- JSON array
    require     TEXT NOT NULL DEFAULT '',
    allow       TEXT NOT NULL DEFAULT '',
    scopes      TEXT NOT NULL DEFAULT '[]',               -- JSON array
    enabled     INTEGER NOT NULL DEFAULT 1,
    priority    INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at  TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
CREATE INDEX IF NOT EXISTS idx_proxy_rules_enabled_priority
    ON proxy_rules(enabled, priority DESC);

-- +goose Down

DROP INDEX IF EXISTS idx_proxy_rules_enabled_priority;
DROP TABLE IF EXISTS proxy_rules;
