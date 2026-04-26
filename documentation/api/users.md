# Users API

## DELETE /api/v1/users/{id}

Permanently deletes a user account.

### Token and session revocation on deletion (Wave 1.5)

Before the user record is removed, the server now performs the following revocation steps to prevent orphaned tokens from passing introspection:

1. **List all agents** created by the user (`created_by = user_id`).
2. **Revoke all OAuth tokens** for each agent via `RevokeOAuthTokensByClientID`.
3. **Delete all active sessions** for the user via `DeleteSessionsByUserID`.
4. **Delete the user** (ON DELETE CASCADE handles FK-linked rows in the schema).

**Audit event**: `user.deleted_with_token_revocation` is written with metadata:

```json
{
  "revoked_token_count": <number>,
  "revoked_session_count": <number>
}
```

Previously issued tokens will return `active: false` on introspection immediately after the DELETE completes.

### Response

Returns `200 OK` with `{ "message": "User deleted" }` on success, or `404` if the user does not exist.
