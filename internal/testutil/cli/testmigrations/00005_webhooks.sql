-- +goose Up
-- +goose StatementBegin

-- Webhooks: admin-registered HTTP endpoints that receive event deliveries.
-- `events` is a JSON array of event names (e.g. ["user.created","session.revoked"]).
-- `secret` is the signing secret (plaintext — stored field-encrypted by the
-- handler layer; hashing here would prevent HMAC computation at dispatch time).
CREATE TABLE IF NOT EXISTS webhooks (
    id         TEXT PRIMARY KEY,
    url        TEXT NOT NULL,
    secret     TEXT NOT NULL,
    events     TEXT NOT NULL DEFAULT '[]',
    enabled    INTEGER NOT NULL DEFAULT 1,
    description TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_webhooks_enabled ON webhooks(enabled);

-- Webhook deliveries: one row per attempt. status one of: pending | delivered
-- | retrying | failed. `attempt` starts at 1. next_retry_at drives the
-- retry scheduler. payload is the JSON body sent; signature_header is the
-- exact X-Shark-Signature value so debuggers can match it against the log.
CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id               TEXT PRIMARY KEY,
    webhook_id       TEXT NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
    event            TEXT NOT NULL,
    payload          TEXT NOT NULL,
    signature_header TEXT NOT NULL DEFAULT '',
    status           TEXT NOT NULL CHECK (status IN ('pending','delivered','retrying','failed')),
    status_code      INTEGER,
    response_body    TEXT NOT NULL DEFAULT '',
    error            TEXT NOT NULL DEFAULT '',
    attempt          INTEGER NOT NULL DEFAULT 1,
    next_retry_at    TEXT,
    delivered_at     TEXT,
    created_at       TEXT NOT NULL,
    updated_at       TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_webhook ON webhook_deliveries(webhook_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_status  ON webhook_deliveries(status, next_retry_at);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_created ON webhook_deliveries(created_at);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_webhook_deliveries_created;
DROP INDEX IF EXISTS idx_webhook_deliveries_status;
DROP INDEX IF EXISTS idx_webhook_deliveries_webhook;
DROP TABLE IF EXISTS webhook_deliveries;
DROP INDEX IF EXISTS idx_webhooks_enabled;
DROP TABLE IF EXISTS webhooks;
-- +goose StatementEnd
