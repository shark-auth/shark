# user_handlers.go

**Path:** `internal/api/user_handlers.go`
**Package:** `api`
**LOC:** 315
**Tests:** `user_handlers_test.go`

## Purpose
Admin user CRUD (`/api/v1/users/*`) plus the user's self-deletion endpoint (`/auth/me`). Linked OAuth account list / unlink also live here.

## Handlers exposed
- `handleListUsers` (func, line 47) — `GET /api/v1/users`; supports `limit`/`offset` AND dashboard's `page`/`per_page`; filters: `search`, `role_id`, `auth_method`, `org_id`, `mfa_enabled`, `email_verified`; returns `{users, total}` (separate count query with limit=1M)
- `handleGetUser` (func, line 110) — `GET /api/v1/users/{id}`
- `handleDeleteUser` (func, line 133) — `DELETE /api/v1/users/{id}`; ON DELETE CASCADE handles related rows; emits `WebhookEventUserDeleted`
- `handleUpdateUser` (func, line 178) — `PATCH /api/v1/users/{id}`; partial update of email, name, email_verified, metadata
- `handleListUserOAuthAccounts` (func, line 233) — `GET /api/v1/users/{id}/oauth-accounts`
- `handleDeleteUserOAuthAccount` (func, line 256) — `DELETE /api/v1/users/{id}/oauth-accounts/{oauthId}`; 204 on success
- `handleDeleteMe` (func, line 278) — `DELETE /api/v1/auth/me`; requires session, deletes account, clears cookie, emits webhook

## Key types
- `adminUserResponse` (struct, line 18) — admin view (snake_case + metadata + last_login_at)
- `updateUserRequest` (struct, line 170) — pointer fields for partial PATCH

## Helpers
- `adminUserToResponse(u *storage.User)` (line 31)

## Imports of note
- `internal/api/middleware` (mw) — context user ID
- `internal/storage` — `ListUsersOpts`, `OAuthAccount`, `WebhookEventUserDeleted`

## Wired by / used by
- Admin routes registered in `internal/api/router.go:423–443` (admin-key gated)
- `handleDeleteMe` mounted at `router.go:251–256` (session + verified email gate)

## Notes
- The "total" count is a second `ListUsers` call with `Limit=1000000` — naive but sufficient for the current scale.
- `updateUserRequest.Metadata` is accepted as raw string (caller must supply valid JSON); no schema validation here.
- Admin response uses snake_case (`email_verified`, `mfa_enabled`); user-self response uses camelCase (`emailVerified`) — historical compat.
