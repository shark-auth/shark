-- +goose Up
ALTER TABLE audit_logs ADD COLUMN org_id TEXT;
ALTER TABLE audit_logs ADD COLUMN session_id TEXT;
ALTER TABLE audit_logs ADD COLUMN resource_type TEXT;
ALTER TABLE audit_logs ADD COLUMN resource_id TEXT;

CREATE INDEX idx_audit_logs_org     ON audit_logs(org_id, created_at);
CREATE INDEX idx_audit_logs_session ON audit_logs(session_id, created_at);

-- +goose Down
DROP INDEX IF EXISTS idx_audit_logs_session;
DROP INDEX IF EXISTS idx_audit_logs_org;
