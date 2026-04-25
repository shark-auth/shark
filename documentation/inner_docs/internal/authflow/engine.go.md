# engine.go

**Path:** `internal/authflow/engine.go`  
**Package:** `authflow`  
**LOC:** 443  
**Tests:** `engine_test.go`

## Purpose
Phase 6 Auth Flow execution engine. Selects flows by trigger, evaluates conditions, walks steps in order, persists run records, and returns aggregate Result. Wire once, configure many via DB.

## Key types / functions
- `Outcome` (type, line 38) — Continue, Block, Redirect, Error, AwaitMFA
- `Context` (struct, line 55) — per-execution state: Trigger, User, Password, Request, Metadata, UserRoles
- `StepResult` (struct, line 67) — Outcome, Reason, RedirectURL, MetadataPatch, ChallengeID
- `Result` (struct, line 76) — aggregate flow outcome: Outcome, Reason, RedirectURL, BlockedAtStep, Timeline
- `StepTimelineEntry` (struct, line 91) — step history entry (index, type, outcome, duration)
- `Engine` (struct, line 104) — store, logger, http.Client (webhook/custom_check), clock
- `NewEngine(store, logger)` (func, line 116) — constructor
- `Execute(ctx, triggerName, user, password, request)` (func) — main entry point
- `ExecuteDryRun(ctx, triggerName, user, password, request)` (func) — no persistence

## Imports of note
- `net/http` — HTTP client for webhooks
- `time` — timeline entry duration tracking
- `internal/storage` — AuthFlow, FlowStep, AuthFlowRun types

## Wired by
- `internal/api` handlers create Engine and call Execute/ExecuteDryRun
- Admin dashboard "Test this flow" button calls ExecuteDryRun

## Notes
- Steps read Context, return StepResult; MUST NOT mutate DB directly
- Side effects queued via StepResult.MetadataPatch (caller decides persistence)
- Engine never panics on nil inputs: defaults to safe values
- Timeline: every step (including Block/Error) gets a timeline entry
- Run records persisted on Execute; ExecuteDryRun skips persistence
- Highest-priority matching flow selected (conditions evaluated top-down)
- HTTP client: 30s hard cap; per-step timeouts via request contexts

