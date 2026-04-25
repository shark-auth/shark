# steps.go

**Path:** `internal/authflow/steps.go`  
**Package:** `authflow`  
**LOC:** 439  
**Tests:** `steps_test.go`

## Purpose
Step executor implementations. Dispatches individual FlowStep types (require_email_verification, require_mfa_enrollment, webhook, redirect, assign_role, etc.) and returns StepResult.

## Key types / functions
- `executeStep(ctx, step, fc)` (func, line 26) — dispatcher; routes step.Type to typed executor
- `requireEmailVerification(ctx, step, fc)` (func, line 66) — blocks if user.EmailVerified=false; upgradable to Redirect
- `requireMFAEnrollment(ctx, step, fc)` (func, line 83) — blocks if no MFA secret; skip_if_enrolled config
- `requirePasswordStrength(ctx, step, fc)` (func, line 100) — validates Password against min_length + require_special
- `executeRedirect(ctx, step, fc)` (func, line 120) — redirect step; extracts url config
- `executeWebhook(ctx, step, fc)` (func, line 135) — POST JSON to configured URL; sanitizes User (no PasswordHash/MFASecret)
- `executeSetMetadata(ctx, step, fc)` (func) — merges step.Config JSON into Context.Metadata
- `executeAssignRole(ctx, step, fc)` (func) — queues role assignment via MetadataPatch
- `executeAddToOrg(ctx, step, fc)` (func) — queues org membership assignment
- `executeCustomCheck(ctx, step, fc)` (func) — invokes custom JS/WASM logic
- `executeDelay(ctx, step, fc)` (func) — sleep (non-blocking; returned as delay config)
- `executeConditional(ctx, step, fc)` (func) — nested flow logic (if-then)
- `requireMFAChallenge(ctx, step, fc)` (func) — issue challenge, return AwaitMFA outcome

## Imports of note
- `net/http` — webhook transport
- `encoding/json` — payload marshaling
- `internal/storage` — FlowStep, User types
- `context` — request timeouts

## Config extractors
- `strFromConfig(step.Config, key, default)` — JSON → string
- `boolFromConfig(step.Config, key, default)` — JSON → bool
- `intFromConfig(step.Config, key, default)` — JSON → int
- `secsFromConfig(step.Config, key, default, max)` — JSON seconds → duration with bounds

## Notes
- Error outcome short-circuits entire flow (individual steps don't swallow failures)
- Webhook sanitization: PasswordHash + MFASecret cleared before POST
- Webhook timeout configurable per step; hard cap 30s on Engine client
- Config handling: all extractors graceful on missing keys (use default)
- Nested conditions: conditional step recursively calls Evaluate + Execute

