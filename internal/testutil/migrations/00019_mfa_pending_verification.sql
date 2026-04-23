-- +goose Up
-- F3.2: MFA pending-verification flag.
ALTER TABLE users ADD COLUMN mfa_verified_at TEXT;
UPDATE users SET mfa_verified_at = updated_at WHERE mfa_verified = 1;

-- +goose Down
