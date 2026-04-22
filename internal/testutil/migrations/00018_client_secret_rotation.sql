-- +goose Up
-- F4.3: OAuth client secret rotation with 1-hour grace window.
ALTER TABLE agents ADD COLUMN old_secret_hash TEXT;
ALTER TABLE agents ADD COLUMN old_secret_expires_at TEXT;

-- +goose Down
