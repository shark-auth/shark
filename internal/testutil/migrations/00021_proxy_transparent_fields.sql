-- +goose Up
ALTER TABLE applications ADD COLUMN proxy_public_domain TEXT;
ALTER TABLE applications ADD COLUMN proxy_protected_url TEXT;

-- +goose Down
ALTER TABLE applications DROP COLUMN proxy_public_domain;
ALTER TABLE applications DROP COLUMN proxy_protected_url;
