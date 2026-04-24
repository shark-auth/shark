-- +goose Up
ALTER TABLE branding ADD COLUMN design_tokens TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE branding DROP COLUMN design_tokens;
