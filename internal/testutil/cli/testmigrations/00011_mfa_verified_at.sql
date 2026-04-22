-- +goose Up
-- F3.2: adds mfa_verified_at to users so the storage layer can read/write it.
ALTER TABLE users ADD COLUMN mfa_verified_at TEXT;
UPDATE users SET mfa_verified_at = updated_at WHERE mfa_verified = 1;

-- +goose Down
