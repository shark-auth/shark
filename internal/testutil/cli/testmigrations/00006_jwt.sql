-- +goose Up

CREATE TABLE jwt_signing_keys (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    kid             TEXT    NOT NULL UNIQUE,
    algorithm       TEXT    NOT NULL DEFAULT 'RS256',
    public_key_pem  TEXT    NOT NULL,
    private_key_pem TEXT    NOT NULL,   -- AES-GCM encrypted, base64-encoded ciphertext
    created_at      DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    rotated_at      DATETIME,           -- NULL while active
    status          TEXT    NOT NULL DEFAULT 'active'
                    CHECK (status IN ('active', 'retired'))
);

CREATE INDEX idx_jwt_signing_keys_status ON jwt_signing_keys(status);

CREATE TABLE revoked_jti (
    jti         TEXT     PRIMARY KEY,
    revoked_at  DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    expires_at  DATETIME NOT NULL
);

CREATE INDEX idx_revoked_jti_expires_at ON revoked_jti(expires_at);

-- +goose Down

DROP INDEX IF EXISTS idx_revoked_jti_expires_at;
DROP TABLE IF EXISTS revoked_jti;
DROP INDEX IF EXISTS idx_jwt_signing_keys_status;
DROP TABLE IF EXISTS jwt_signing_keys;
