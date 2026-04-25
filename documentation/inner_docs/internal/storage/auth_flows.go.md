# auth_flows.go

**Path:** `internal/storage/auth_flows.go`
**Package:** `storage`
**LOC:** 73
**Tests:** indirectly via `auth_flows_sqlite_test.go`.

## Purpose
Type-only file: declares the entities for the Phase 6 Visual Flow Builder — admin-composed pipelines that run at auth trigger points (signup, login, password reset, magic link, OAuth callback).

## Types defined
- `AuthFlow` (line 29) — ordered pipeline of `FlowStep`s tied to a trigger, with `Priority` (higher first), `Enabled` flag, and JSON-encoded `Conditions` map (when this flow applies, e.g. `{"user.org_id": "org_123"}`).
- `FlowStep` (line 46) — one node. Either a leaf (Type = `require_email_verification`/`require_mfa_enrollment`/`redirect`/`webhook`/etc.) or a branch (Type = `conditional`) with `Condition` expression plus recursive `ThenSteps`/`ElseSteps`.
- `AuthFlowRun` (line 62) — append-only audit row of a single flow evaluation; `Outcome` ∈ {`continue`, `block`, `redirect`, `error`}; `BlockedAtStep` index when halted.

## Constants
- Triggers (lines 6-12): `signup`, `login`, `password_reset`, `magic_link`, `oauth_callback`
- Outcomes (lines 15-20): `continue`, `block`, `redirect`, `error`

## Used by
- `internal/storage/auth_flows_sqlite.go` — implementation.
- `internal/flows` (Phase 6 engine) — reads flows by trigger and executes Steps in order.
- `internal/api/auth_flows.go` admin CRUD.

## Notes
- Steps + Conditions are stored as JSON; callers always work with the typed Go representation here.
- Tie-breaking: ties on Priority resolve by `CreatedAt ASC` so behavior is deterministic.
