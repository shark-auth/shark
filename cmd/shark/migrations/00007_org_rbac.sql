-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS org_roles (
  id TEXT PRIMARY KEY,                        -- orgrole_<nanoid>
  organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  description TEXT,
  is_builtin BOOLEAN NOT NULL DEFAULT 0,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(organization_id, name)
);

CREATE TABLE IF NOT EXISTS org_role_permissions (
  org_role_id TEXT NOT NULL REFERENCES org_roles(id) ON DELETE CASCADE,
  action TEXT NOT NULL,
  resource TEXT NOT NULL,
  PRIMARY KEY (org_role_id, action, resource)
);

CREATE TABLE IF NOT EXISTS org_user_roles (
  organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  org_role_id TEXT NOT NULL REFERENCES org_roles(id) ON DELETE CASCADE,
  granted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  granted_by TEXT REFERENCES users(id) ON DELETE SET NULL,
  PRIMARY KEY (organization_id, user_id, org_role_id)
);

CREATE INDEX IF NOT EXISTS idx_org_roles_org ON org_roles(organization_id);
CREATE INDEX IF NOT EXISTS idx_org_user_roles_user ON org_user_roles(user_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_org_user_roles_user;
DROP INDEX IF EXISTS idx_org_roles_org;
DROP TABLE IF EXISTS org_user_roles;
DROP TABLE IF EXISTS org_role_permissions;
DROP TABLE IF EXISTS org_roles;

-- +goose StatementEnd
