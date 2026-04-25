# flow_mfa_handlers.go

**Path:** `internal/api/flow_mfa_handlers.go`
**Package:** `api`
**LOC:** 87
**Tests:** none directly (covered by `flow_handlers_test.go` + integration)

## Purpose
Single endpoint that lets the SDK satisfy a `require_mfa_challenge` step inside a paused auth flow — at the point where the user has no session yet (the challenge ID is the proof of prior auth). Used by Phase 6 F3 auth flows.

## Handlers exposed
- `handleFlowMFAVerify` (func, line 43) — `POST /api/v1/auth/flow/mfa/verify`; unauthenticated by design; consumes the challenge from `authflow.GlobalChallengeStore` (single-use, atomic), looks up the user, validates TOTP, returns `{verified: true, user_id}` on success or 401 on failure
  - Body: `{flow_run_id, challenge_id, code, user_id}` — `flow_run_id` is informational/forward-compat only

## Key types
- `flowMFAVerifyRequest` (struct, line 13)

## Imports of note
- `internal/authflow` — `GlobalChallengeStore.Consume(challengeID, userID)`
- `internal/auth` — `MFAManager.ValidateTOTP`
- `internal/config` — empty `MFAConfig{}` for `MFAManager` ctor

## Wired by / used by
- Route registered in `internal/api/router.go:335–337` (`/auth/flow/mfa/verify`)
- Originator: handlers that ran a flow returning `awaiting_mfa` (login, signup, magic-link)

## Notes
- Unauthenticated — challenge ID is the credential. Validation: challenge must exist, not be expired, and `user_id` must match the recorded subject.
- Returns 200 success contract is a no-op for session minting; the SDK is expected to re-POST the original action (e.g. login) to actually create the session.
- `MFAManager` is constructed with empty `MFAConfig{}` — only `ValidateTOTP` is called, which doesn't read config fields.
