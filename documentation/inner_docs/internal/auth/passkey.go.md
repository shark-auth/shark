# passkey.go

**Path:** `internal/auth/passkey.go`
**Package:** `auth`
**LOC:** 478
**Tests:** `passkey_test.go`

## Purpose
WebAuthn/passkey ceremony orchestration: registration + authentication, challenge-session storage with TTL, and adapter from storage credentials to `webauthn.Credential`.

## Key types / functions
- Constants (line 22-26): `passkeyPrefix="pk_"`, `challengeTTL=5min`, `cleanupInterval=1min`.
- `ErrPasskeyNotFound`, `ErrChallengeNotFound`, `ErrChallengeExpired`, `ErrNoPasskeys` (vars, line 28-33).
- `challengeStore` (type, line 42) — RWMutex map of challenge keys → `*webauthn.SessionData` with expiry; `put` (line 57), `get` (line 66; one-shot delete), `cleanupLoop` (line 84), `cleanup` (line 97), `stop` (line 108).
- `webauthnUser` (type, line 113) — adapter implementing the `webauthn.User` interface (`WebAuthnID`, `WebAuthnName`, `WebAuthnDisplayName`, `WebAuthnCredentials`).
- `PasskeyManager` (type, line 138) — wraps `webauthn.WebAuthn`, store, sessions, challenge store.
- `NewPasskeyManager` (func, line 146) — applies attestation/residentKey/userVerification defaults; builds `webauthn.New`.
- `Stop` (func, line 187) — stops the cleanup goroutine.
- `BeginRegistration` (func, line 193) — loads existing creds, builds exclusion list, calls `webauthn.BeginRegistration`, stashes session data under generated `wac_` challenge key.
- `FinishRegistration` (func, line 226) — pops challenge, verifies attestation, persists `storage.PasskeyCredential` (with transports JSON, AAGUID hex, sign count).
- `BeginLogin` (func, line 286) — branched: email-scoped (allowCredentials) or discoverable (resident-key) flow.
- `FinishLogin` (func, line 331) — verifies assertion, looks up user, updates sign count, creates session with `auth_method="passkey"` and `mfa_passed=true`.
- `loadWebAuthnCredentials` (func, line 405) — converts stored creds to library type (transports, AAGUID, flags, sign count).
- `updateCredentialAfterLogin` (func, line 447) — bumps sign count + LastUsedAt; warns on non-monotonic counter (clone detection signal).
- `generateChallengeKey` / `withExclusions` (funcs, line 469-478) — helpers.

## Imports of note
- `github.com/go-webauthn/webauthn` + `protocol` — WebAuthn FIDO2 ceremony.
- `github.com/matoous/go-nanoid/v2` — credential and challenge IDs.

## Used by
- `internal/api/passkey_handlers.go` — REST endpoints for begin/finish register/login.

## Notes
- Challenge store is **PROCESS-LOCAL** — registration and login must hit the same instance unless sticky sessions are configured.
- Sign-count regression triggers a `slog.Warn` rather than rejecting the login (line 456); tighten if cloned-authenticator detection becomes a requirement.
- Discoverable login uses `FinishPasskeyLogin` with a handler that resolves the user via the userHandle (line 364).
- `mfa_passed=true` is forced on passkey login because the authenticator already enforces user verification.
