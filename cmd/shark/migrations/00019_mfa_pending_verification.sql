-- +goose Up
-- F3.2: MFA pending-verification flag.
-- Adds mfa_verified_at to users so we can distinguish "enrolled but not yet
-- verified" (NULL) from "verified" (non-NULL timestamp).
-- Backfills existing verified rows from the mfa_verified boolean.
ALTER TABLE users ADD COLUMN mfa_verified_at TEXT;
UPDATE users SET mfa_verified_at = updated_at WHERE mfa_verified = 1;

-- +goose Down
-- SQLite does not support DROP COLUMN on older versions; omit for safety.
