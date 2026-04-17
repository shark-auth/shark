-- +goose Up
CREATE TABLE IF NOT EXISTS applications (
  id                    TEXT NOT NULL PRIMARY KEY,         -- app_<nanoid>
  name                  TEXT NOT NULL,
  client_id             TEXT NOT NULL UNIQUE,              -- shark_app_<nanoid>
  client_secret_hash    TEXT NOT NULL,                     -- SHA-256 hex
  client_secret_prefix  TEXT NOT NULL,                     -- first 8 chars (UX display)
  allowed_callback_urls TEXT NOT NULL DEFAULT '[]',        -- JSON array
  allowed_logout_urls   TEXT NOT NULL DEFAULT '[]',        -- JSON array
  allowed_origins       TEXT NOT NULL DEFAULT '[]',        -- JSON array
  is_default            BOOLEAN NOT NULL DEFAULT 0,
  metadata              TEXT NOT NULL DEFAULT '{}',
  created_at            TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at            TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX idx_applications_one_default
  ON applications(is_default) WHERE is_default = 1;

-- +goose Down
DROP INDEX IF EXISTS idx_applications_one_default;
DROP TABLE IF EXISTS applications;
