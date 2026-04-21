-- +goose Up
-- Branding config (global row for V1, extensible to org scope later)
CREATE TABLE branding (
  id TEXT PRIMARY KEY,
  scope TEXT NOT NULL DEFAULT 'global',
  logo_url TEXT,
  logo_sha TEXT,
  primary_color TEXT DEFAULT '#7c3aed',
  secondary_color TEXT DEFAULT '#1a1a1a',
  font_family TEXT DEFAULT 'manrope',
  footer_text TEXT DEFAULT '',
  email_from_name TEXT DEFAULT 'SharkAuth',
  email_from_address TEXT DEFAULT 'noreply@example.com',
  updated_at TEXT NOT NULL
);

INSERT INTO branding (id, scope, updated_at) VALUES ('global', 'global', CURRENT_TIMESTAMP);

-- Email templates (editable copy)
CREATE TABLE email_templates (
  id TEXT PRIMARY KEY,
  subject TEXT NOT NULL,
  preheader TEXT NOT NULL DEFAULT '',
  header_text TEXT NOT NULL,
  body_paragraphs TEXT NOT NULL DEFAULT '[]',
  cta_text TEXT NOT NULL DEFAULT '',
  cta_url_template TEXT NOT NULL DEFAULT '',
  footer_text TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL
);

-- Applications integration mode + branding override + proxy fallback
ALTER TABLE applications ADD COLUMN integration_mode TEXT NOT NULL DEFAULT 'custom';
ALTER TABLE applications ADD COLUMN branding_override TEXT;
ALTER TABLE applications ADD COLUMN proxy_login_fallback TEXT NOT NULL DEFAULT 'hosted';
ALTER TABLE applications ADD COLUMN proxy_login_fallback_url TEXT;

-- Users welcome-email-sent flag
ALTER TABLE users ADD COLUMN welcome_email_sent INTEGER NOT NULL DEFAULT 0;

-- +goose Down
DROP TABLE email_templates;
DROP TABLE branding;
-- NOTE: SQLite does not support ALTER TABLE DROP COLUMN pre 3.35.
-- Rollback leaves applications + users columns in place; acceptable for dev.
