-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS organizations (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    metadata    TEXT NOT NULL DEFAULT '{}',
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_organizations_slug ON organizations(slug);

-- role is an OrgRole enum validated at the handler: owner | admin | member.
-- Enforced at the SQL level via a CHECK to refuse unknown values even if
-- a buggy caller bypasses the handler.
CREATE TABLE IF NOT EXISTS organization_members (
    organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role            TEXT NOT NULL CHECK (role IN ('owner', 'admin', 'member')),
    joined_at       TEXT NOT NULL,
    PRIMARY KEY (organization_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_organization_members_user ON organization_members(user_id);
CREATE INDEX IF NOT EXISTS idx_organization_members_org  ON organization_members(organization_id);

CREATE TABLE IF NOT EXISTS organization_invitations (
    id               TEXT PRIMARY KEY,
    organization_id  TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email            TEXT NOT NULL,
    role             TEXT NOT NULL CHECK (role IN ('owner', 'admin', 'member')),
    token_hash       TEXT NOT NULL UNIQUE,
    invited_by       TEXT REFERENCES users(id) ON DELETE SET NULL,
    accepted_at      TEXT,
    expires_at       TEXT NOT NULL,
    created_at       TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_org_invitations_org   ON organization_invitations(organization_id);
CREATE INDEX IF NOT EXISTS idx_org_invitations_email ON organization_invitations(email);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_org_invitations_email;
DROP INDEX IF EXISTS idx_org_invitations_org;
DROP TABLE IF EXISTS organization_invitations;
DROP INDEX IF EXISTS idx_organization_members_org;
DROP INDEX IF EXISTS idx_organization_members_user;
DROP TABLE IF EXISTS organization_members;
DROP INDEX IF EXISTS idx_organizations_slug;
DROP TABLE IF EXISTS organizations;
-- +goose StatementEnd
