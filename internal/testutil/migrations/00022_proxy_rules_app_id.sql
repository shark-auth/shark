-- +goose Up
-- Migration: Add app_id to proxy_rules for per-app route control
ALTER TABLE proxy_rules ADD COLUMN app_id TEXT;
CREATE INDEX idx_proxy_rules_app_id ON proxy_rules(app_id);

-- Backfill: Assign existing rules to the default app if it exists.
UPDATE proxy_rules SET app_id = (SELECT id FROM applications WHERE slug = 'default' LIMIT 1) WHERE app_id IS NULL;

-- +goose Down
DROP INDEX idx_proxy_rules_app_id;
ALTER TABLE proxy_rules DROP COLUMN app_id;
