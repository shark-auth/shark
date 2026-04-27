-- +goose Up
-- may_act_grants: operator-issued grants letting subject `from_id` act on behalf
-- of `to_id`. Distinct from the JWT-claim path (subjectClaims["may_act"]) — this
-- table is the auditable, revocable record. token-exchange writes the matched
-- grant_id into the audit row metadata so the dashboard can correlate hops.
--
-- scopes stored as JSON array string. expires_at + revoked_at NULL means
-- live-forever / not-revoked.
CREATE TABLE may_act_grants (
    id          TEXT PRIMARY KEY,
    from_id     TEXT NOT NULL,
    to_id       TEXT NOT NULL,
    max_hops    INTEGER NOT NULL DEFAULT 1,
    scopes      TEXT NOT NULL DEFAULT '[]',
    expires_at  TIMESTAMP,
    revoked_at  TIMESTAMP,
    created_by  TEXT,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_may_act_grants_from ON may_act_grants(from_id);
CREATE INDEX idx_may_act_grants_to   ON may_act_grants(to_id);

-- +goose Down
DROP INDEX IF EXISTS idx_may_act_grants_to;
DROP INDEX IF EXISTS idx_may_act_grants_from;
DROP TABLE IF EXISTS may_act_grants;
