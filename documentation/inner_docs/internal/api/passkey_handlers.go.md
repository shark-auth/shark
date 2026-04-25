# passkey_handlers.go

**Path:** `internal/api/passkey_handlers.go`
**Package:** `api`
**LOC:** 330
**Tests:** `passkey_handlers_test.go`

## Purpose
WebAuthn / passkey HTTP surface — registration begin/finish, login begin/finish (no session required), and per-credential management (list, delete, rename) for the authenticated user.

## Handlers exposed
- `handlePasskeyRegisterBegin` (func, line 33) — `POST /api/v1/auth/passkey/register/begin`; requires session; calls `PasskeyManager.BeginRegistration`, returns `{publicKey, challengeKey}`
- `handlePasskeyRegisterFinish` (func, line 67) — `POST /api/v1/auth/passkey/register/finish`; requires `X-Challenge-Key` header; persists credential via `FinishRegistration`
- `handlePasskeyLoginBegin` (func, line 110) — `POST /api/v1/auth/passkey/login/begin`; public; optional `{email}` body for username-bound flow; `auth.ErrNoPasskeys` → 400 `no_passkeys`
- `handlePasskeyLoginFinish` (func, line 144) — `POST /api/v1/auth/passkey/login/finish`; public; verifies assertion via challenge key, sets session cookie, returns `userToResponse`
- `handlePasskeyCredentialsList` (func, line 168) — `GET /api/v1/auth/passkey/credentials`; lists current user's credentials
- `handlePasskeyCredentialDelete` (func, line 204) — `DELETE /api/v1/auth/passkey/credentials/{id}`; ownership-checked
- `handlePasskeyCredentialRename` (func, line 260) — `PATCH /api/v1/auth/passkey/credentials/{id}`; `{name}` body; ownership-checked

## Key types
- `passkeyLoginBeginRequest` (struct, line 14)
- `passkeyRenameRequest` (struct, line 19)
- `passkeyCredentialResponse` (struct, line 24) — id, name, transports, backed_up, created_at, last_used_at

## Imports of note
- `internal/auth` — `PasskeyManager` (Begin/Finish Registration + Login), `ErrNoPasskeys`
- `internal/api/middleware` (mw) — context user ID

## Wired by / used by
- Routes registered in `internal/api/router.go:273–292` — register + credential management gated by session + verified email; login/begin and login/finish are public

## Notes
- Challenge state is keyed by an opaque `challengeKey` returned in begin and replayed via `X-Challenge-Key` header on finish — server-side challenge stash, not a JWT.
- `PasskeyManager` is `nil` when config is invalid (e.g. missing RPID) — these endpoints return 500s in that case (no panic).
- Ownership for delete/rename is verified by enumerating the user's credentials rather than a single `GetByID(uid, credID)` lookup.
