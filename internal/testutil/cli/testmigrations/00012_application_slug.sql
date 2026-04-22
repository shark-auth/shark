-- +goose Up
-- B1: slug column for hosted page routing.
ALTER TABLE applications ADD COLUMN slug TEXT;
UPDATE applications SET slug = lower(replace(replace(name, ' ', '-'), '_', '-')) WHERE slug IS NULL;
CREATE UNIQUE INDEX applications_slug_unique ON applications(slug);

-- +goose Down
DROP INDEX applications_slug_unique;
