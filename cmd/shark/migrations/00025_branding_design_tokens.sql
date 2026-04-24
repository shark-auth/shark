-- +goose Up
-- PROXYV1_5 §4.11: design tokens live as a JSON blob so the dashboard
-- can evolve its token tree without schema migrations per field.
ALTER TABLE branding ADD COLUMN design_tokens TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE branding DROP COLUMN design_tokens;
