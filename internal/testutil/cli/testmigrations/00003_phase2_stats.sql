-- +goose Up
-- +goose StatementBegin

-- Dev-mode email inbox (populated only when `shark serve --dev` runs).
-- Using `to_addr` instead of `to` because `to` is a SQLite reserved keyword.
CREATE TABLE IF NOT EXISTS dev_emails (
    id         TEXT PRIMARY KEY,
    to_addr    TEXT NOT NULL,
    subject    TEXT NOT NULL,
    html       TEXT NOT NULL DEFAULT '',
    text       TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_dev_emails_created ON dev_emails(created_at DESC);

-- Indexes backing admin session filtering + keyset pagination.
CREATE INDEX IF NOT EXISTS idx_sessions_auth_method ON sessions(auth_method);
CREATE INDEX IF NOT EXISTS idx_sessions_created_id  ON sessions(created_at DESC, id DESC);

-- Index backing failed-login counts in /admin/stats.
CREATE INDEX IF NOT EXISTS idx_audit_logs_action_status_created
    ON audit_logs(action, status, created_at);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_audit_logs_action_status_created;
DROP INDEX IF EXISTS idx_sessions_created_id;
DROP INDEX IF EXISTS idx_sessions_auth_method;
DROP INDEX IF EXISTS idx_dev_emails_created;
DROP TABLE IF EXISTS dev_emails;
-- +goose StatementEnd
