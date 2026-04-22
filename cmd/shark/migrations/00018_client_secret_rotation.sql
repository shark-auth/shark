-- +goose Up
-- F4.3: OAuth client secret rotation with 1-hour grace window.
-- Adds old_secret_hash and old_secret_expires_at to agents so the previous
-- secret remains valid until the grace window expires.
ALTER TABLE agents ADD COLUMN old_secret_hash TEXT;
ALTER TABLE agents ADD COLUMN old_secret_expires_at TEXT;

-- +goose Down
-- SQLite does not support DROP COLUMN on older versions; omit for safety.
-- The columns are nullable and unused when not populated.
