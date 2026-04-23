-- +goose Up
ALTER TABLE applications ADD COLUMN slug TEXT;
-- Backfill: lowercase name, spaces+underscores → hyphens. Leaves alnum + hyphens.
UPDATE applications SET slug = lower(replace(replace(name, ' ', '-'), '_', '-')) WHERE slug IS NULL;
CREATE UNIQUE INDEX applications_slug_unique ON applications(slug);

-- +goose Down
DROP INDEX applications_slug_unique;
