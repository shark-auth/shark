-- +goose Up

-- PKCE request session storage. Required because fosite calls
-- CreateAuthorizeCodeSession with a Sanitize()-stripped Requester (no
-- code_challenge in the form), so the PKCE handler relies on a separate
-- CreatePKCERequestSession call to persist the unsanitized challenge.
-- Without this table the token endpoint cannot validate code_verifier
-- and Auth Code + PKCE flow returns 400 at exchange time.
CREATE TABLE IF NOT EXISTS oauth_pkce_sessions (
    signature_hash         TEXT PRIMARY KEY,                    -- SHA-256 of fosite signature
    code_challenge         TEXT NOT NULL,
    code_challenge_method  TEXT NOT NULL DEFAULT 'S256',
    client_id              TEXT NOT NULL,
    expires_at             TIMESTAMP NOT NULL,
    created_at             TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX idx_oauth_pkce_sessions_expires_at ON oauth_pkce_sessions(expires_at);

-- +goose Down

DROP INDEX IF EXISTS idx_oauth_pkce_sessions_expires_at;
DROP TABLE IF EXISTS oauth_pkce_sessions;
