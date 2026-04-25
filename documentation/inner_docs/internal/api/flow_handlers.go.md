# flow_handlers.go

**Path:** `internal/api/flow_handlers.go`
**Package:** `api`
**LOC:** 555
**Tests:** likely integration-tested

## Purpose
CRUD + dry-run + run-history for AuthFlow definitions. Flows attach to a trigger (signup/login/password-reset/magic-link/oauth-callback) and run an ordered list of steps (require_email_verification, require_mfa_*, require_password_strength, redirect, webhook, set_metadata, assign_role, add_to_org, custom_check, delay, conditional). Validation rejects unknown trigger / step types up front so the dashboard never persists flows that always fail.

## Handlers exposed
- `handleCreateFlow` (line 184) — POST `/admin/flows`. Validates trigger + step types.
- `handleListFlows` (line 227) — GET.
- `handleGetFlow` (line 253) — GET `/{id}`.
- `handleUpdateFlow` (line 269) — PATCH (all fields optional pointers).
- `handleDeleteFlow` (line 336) — DELETE.
- `handleTestFlow` (line 360) — POST `/{id}/test`. Dry-run with caller-supplied `mockFlowUser`.
- `handleListFlowRuns` (line 409) — GET `/{id}/runs`.

## Internal entry point
- `runAuthFlow` (line 449) — invoked by signup/login/etc. handlers to actually execute matching flows; returns `handled=true` when a step blocked the request.

## Key types
- `flowResponse` (line 52), `flowRunResponse` (line 90)
- `createFlowRequest` (line 124), `updateFlowRequest` (line 133), `testFlowRequest` (line 142)
- `mockFlowUser` (line 151) — minimal User stand-in for dry-run.

## Helpers
- `validAuthFlowTriggers` (line 23, set), `supportedFlowStepTypes` (line 34, set).
- `validateFlowPayload` (line 513), `validateSteps` (line 527), `validateStepsRecursive` (line 534).
- `newAuthFlowID` (line 503), `flowToResponse`/`flowRunToResponse` (lines 64, 103).

## Imports of note
- `internal/authflow` — engine
- `internal/storage` — `AuthFlow`, `AuthFlowRun`, `FlowStep`

## Wired by
- `internal/api/router.go:711-717`

## Notes
- Recursive step validation handles `conditional` step's nested branches.
- Dry-run uses `usr_dryrun` when caller omits user.id.
