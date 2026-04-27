-- +goose Up
ALTER TABLE vault_providers ADD COLUMN extra_auth_params TEXT NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE vault_providers DROP COLUMN extra_auth_params;
