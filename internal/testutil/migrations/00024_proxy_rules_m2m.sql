-- +goose Up
-- PROXYV1_5 §4.17: add m2m boolean so rules can be locked to agent
-- callers (service-to-service). Default 0 keeps legacy rules human-inclusive.
ALTER TABLE proxy_rules ADD COLUMN m2m INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE proxy_rules DROP COLUMN m2m;
