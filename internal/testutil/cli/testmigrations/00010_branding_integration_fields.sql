-- +goose Up
-- Mirror of production migration 00017 — adds the integration_mode +
-- branding_override + proxy_login_fallback columns the applications
-- storage layer unconditionally writes since A8. Kept trimmed to just
-- the columns CreateApplication / UpdateApplication touch; the branding
-- + email_templates tables live in main migrations only.
ALTER TABLE applications ADD COLUMN integration_mode TEXT NOT NULL DEFAULT 'custom';
ALTER TABLE applications ADD COLUMN branding_override TEXT;
ALTER TABLE applications ADD COLUMN proxy_login_fallback TEXT NOT NULL DEFAULT 'hosted';
ALTER TABLE applications ADD COLUMN proxy_login_fallback_url TEXT;

-- +goose Down
-- SQLite pre-3.35 lacks DROP COLUMN; leave columns in place.
