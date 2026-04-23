-- +goose Up

-- fosite calls RotateRefreshToken / RevokeAccessToken with its internal request
-- ID, and reuses the same request ID across rotation chains (refresh.go:86 sets
-- new request.ID = originalRequest.ID). With jti tied 1:1 to that request ID
-- the unique constraint blocks re-insertion of a new (rotated) token under the
-- same chain. Decouple: add a separate request_id column for fosite lookups,
-- let jti be globally unique per token, free of request scoping.
ALTER TABLE oauth_tokens ADD COLUMN request_id TEXT;
CREATE INDEX idx_oauth_tokens_request_id_type ON oauth_tokens(request_id, token_type);

-- +goose Down

DROP INDEX IF EXISTS idx_oauth_tokens_request_id_type;
ALTER TABLE oauth_tokens DROP COLUMN request_id;
