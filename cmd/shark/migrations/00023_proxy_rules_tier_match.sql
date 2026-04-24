-- +goose Up
ALTER TABLE proxy_rules ADD COLUMN tier_match TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE proxy_rules DROP COLUMN tier_match;
