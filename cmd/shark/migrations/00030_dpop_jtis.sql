-- +goose Up
CREATE TABLE dpop_jtis (
    jti        TEXT PRIMARY KEY,
    expires_at TEXT NOT NULL
);

CREATE INDEX idx_dpop_jtis_expires ON dpop_jtis(expires_at);

-- +goose Down
DROP TABLE IF EXISTS dpop_jtis;
