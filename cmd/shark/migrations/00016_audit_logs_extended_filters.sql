-- +goose Up
-- Add org_id, session_id, resource_type, resource_id columns to audit_logs.
-- Existing rows get NULL for these columns (no backfill needed).
ALTER TABLE audit_logs ADD COLUMN org_id TEXT;
ALTER TABLE audit_logs ADD COLUMN session_id TEXT;
ALTER TABLE audit_logs ADD COLUMN resource_type TEXT;
ALTER TABLE audit_logs ADD COLUMN resource_id TEXT;

CREATE INDEX idx_audit_logs_org ON audit_logs(org_id, created_at);
CREATE INDEX idx_audit_logs_session ON audit_logs(session_id, created_at);

-- +goose Down
DROP INDEX IF EXISTS idx_audit_logs_session;
DROP INDEX IF EXISTS idx_audit_logs_org;
-- SQLite does not support DROP COLUMN before 3.35.0; recreate to roll back cleanly.
CREATE TABLE audit_logs_backup AS SELECT id, actor_id, actor_type, action, target_type, target_id, ip, user_agent, metadata, status, created_at FROM audit_logs;
DROP TABLE audit_logs;
ALTER TABLE audit_logs_backup RENAME TO audit_logs;
CREATE INDEX idx_audit_logs_actor   ON audit_logs(actor_id, created_at);
CREATE INDEX idx_audit_logs_action  ON audit_logs(action, created_at);
CREATE INDEX idx_audit_logs_target  ON audit_logs(target_id, created_at);
CREATE INDEX idx_audit_logs_created ON audit_logs(created_at);
