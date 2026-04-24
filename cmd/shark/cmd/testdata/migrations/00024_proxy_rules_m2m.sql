-- +goose Up
ALTER TABLE proxy_rules ADD COLUMN m2m INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE proxy_rules DROP COLUMN m2m;
