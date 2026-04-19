package authflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sharkauth/sharkauth/internal/storage"
)

// executeStep dispatches a single FlowStep to its typed executor.
//
// Returning an Error outcome here short-circuits the entire flow at the
// engine loop — individual steps must not swallow a hard failure by
// returning Continue with a reason string. Use Continue only when the step
// truly completed, or (for stubs) when the wiring hasn't landed yet and
// surfacing an error would pointlessly break every flow that references it.
func (e *Engine) executeStep(ctx context.Context, step *storage.FlowStep, fc *Context) StepResult {
	if step == nil {
		return StepResult{Outcome: Error, Reason: "nil step"}
	}
	switch step.Type {
	case "require_email_verification":
		return e.requireEmailVerification(ctx, step, fc)
	case "require_mfa_enrollment":
		return e.requireMFAEnrollment(ctx, step, fc)
	case "require_mfa_challenge":
		return e.requireMFAChallenge(ctx, step, fc)
	case "require_password_strength":
		return e.requirePasswordStrength(ctx, step, fc)
	case "redirect":
		return e.executeRedirect(ctx, step, fc)
	case "webhook":
		return e.executeWebhook(ctx, step, fc)
	case "set_metadata":
		return e.executeSetMetadata(ctx, step, fc)
	case "assign_role":
		return e.executeAssignRole(ctx, step, fc)
	case "add_to_org":
		return e.executeAddToOrg(ctx, step, fc)
	case "custom_check":
		return e.executeCustomCheck(ctx, step, fc)
	case "delay":
		return e.executeDelay(ctx, step, fc)
	case "conditional":
		return e.executeConditional(ctx, step, fc)
	default:
		return StepResult{Outcome: Error, Reason: "unknown step type: " + step.Type}
	}
}

// --- Fully wired step executors ---

// requireEmailVerification blocks when the user's email is unverified. If
// the step config carries a "redirect" URL the block is upgraded to a
// Redirect outcome — handy for sending users to a verification page
// instead of a dead end.
func (e *Engine) requireEmailVerification(_ context.Context, step *storage.FlowStep, fc *Context) StepResult {
	if fc.User == nil {
		return StepResult{Outcome: Error, Reason: "email verification step needs a user context"}
	}
	if fc.User.EmailVerified {
		return StepResult{Outcome: Continue}
	}
	if redirect := strFromConfig(step.Config, "redirect", ""); redirect != "" {
		return StepResult{Outcome: Redirect, RedirectURL: redirect, Reason: "email verification required"}
	}
	return StepResult{Outcome: Block, Reason: "email verification required"}
}

// requireMFAEnrollment blocks when the user has no MFA secret. Config key
// "skip_if_enrolled" (default true) controls whether an already-enrolled
// user flows past — a value of false is useful for admin-only flows that
// want to insist the user re-confirm enrollment.
func (e *Engine) requireMFAEnrollment(_ context.Context, step *storage.FlowStep, fc *Context) StepResult {
	if fc.User == nil {
		return StepResult{Outcome: Error, Reason: "mfa enrollment step needs a user context"}
	}
	enrolled := fc.User.MFASecret != nil && *fc.User.MFASecret != ""
	if enrolled && boolFromConfig(step.Config, "skip_if_enrolled", true) {
		return StepResult{Outcome: Continue}
	}
	if enrolled {
		return StepResult{Outcome: Continue}
	}
	return StepResult{Outcome: Block, Reason: "mfa enrollment required"}
}

// requirePasswordStrength validates fc.Password against min_length and
// require_special rules. Blocks with a specific reason so dashboards can
// render helpful errors. Empty password also blocks.
func (e *Engine) requirePasswordStrength(_ context.Context, step *storage.FlowStep, fc *Context) StepResult {
	minLen := intFromConfig(step.Config, "min_length", 12)
	requireSpecial := boolFromConfig(step.Config, "require_special", true)

	pw := fc.Password
	if pw == "" {
		return StepResult{Outcome: Block, Reason: "password required"}
	}
	if len(pw) < minLen {
		return StepResult{Outcome: Block, Reason: fmt.Sprintf("password too short (min %d)", minLen)}
	}
	if requireSpecial && !containsSpecial(pw) {
		return StepResult{Outcome: Block, Reason: "password missing special character"}
	}
	return StepResult{Outcome: Continue}
}

// executeRedirect is the literal "send the user somewhere else" step. The
// "delay" config key is passed through untouched for the HTTP layer to
// honour — this engine doesn't sleep on it.
func (e *Engine) executeRedirect(_ context.Context, step *storage.FlowStep, _ *Context) StepResult {
	url := strFromConfig(step.Config, "url", "")
	if url == "" {
		return StepResult{Outcome: Error, Reason: "redirect step missing url"}
	}
	return StepResult{Outcome: Redirect, RedirectURL: url, Reason: "redirect"}
}

// executeWebhook POSTs (or other configured method) a JSON payload to the
// configured URL and continues on 2xx. 4xx/5xx and network/timeout errors
// produce an Error outcome — callers decide whether that should block auth
// (for critical audit hooks) or be tolerated.
//
// Payload sanitization: the User copy has PasswordHash and MFASecret
// cleared. A misbehaving endpoint must never see a credential.
func (e *Engine) executeWebhook(ctx context.Context, step *storage.FlowStep, fc *Context) StepResult {
	url := strFromConfig(step.Config, "url", "")
	if url == "" {
		return StepResult{Outcome: Error, Reason: "webhook step missing url"}
	}
	method := strFromConfig(step.Config, "method", http.MethodPost)
	timeout := secsFromConfig(step.Config, "timeout", 5, 30)

	body, err := json.Marshal(map[string]any{
		"trigger":  fc.Trigger,
		"user":     sanitizeUser(fc.User),
		"metadata": fc.Metadata,
	})
	if err != nil {
		return StepResult{Outcome: Error, Reason: "webhook marshal: " + err.Error()}
	}

	reqCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, method, url, bytes.NewReader(body))
	if err != nil {
		return StepResult{Outcome: Error, Reason: "webhook request build: " + err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	if hdrs, ok := step.Config["headers"].(map[string]any); ok {
		for k, v := range hdrs {
			if s, ok := v.(string); ok {
				req.Header.Set(k, s)
			}
		}
	}

	client := e.http
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return StepResult{Outcome: Error, Reason: "webhook: " + err.Error()}
	}
	defer resp.Body.Close()
	_ = readAndDiscard(resp.Body)

	if resp.StatusCode >= 400 {
		return StepResult{Outcome: Error, Reason: fmt.Sprintf("webhook returned %d", resp.StatusCode)}
	}
	return StepResult{Outcome: Continue}
}

// executeConditional evaluates step.Condition and runs either ThenSteps or
// ElseSteps in sequence. Matches the engine's outer loop: a Continue on
// each sub-step means "keep going"; anything else short-circuits the
// branch.
//
// An empty Condition string matches everything (ThenSteps always run).
func (e *Engine) executeConditional(ctx context.Context, step *storage.FlowStep, fc *Context) StepResult {
	cond, err := conditionMap(step.Condition)
	if err != nil {
		return StepResult{Outcome: Error, Reason: "bad conditional: " + err.Error()}
	}
	match, err := Evaluate(cond, fc)
	if err != nil {
		return StepResult{Outcome: Error, Reason: "bad conditional: " + err.Error()}
	}

	branch := step.ElseSteps
	if match {
		branch = step.ThenSteps
	}

	accumulated := map[string]any{}
	for i, inner := range branch {
		inner := inner
		sub := e.executeStep(ctx, &inner, fc)
		// Merge metadata patches between inner steps so later steps see earlier ones.
		for k, v := range sub.MetadataPatch {
			accumulated[k] = v
		}
		mergeMetadata(fc, sub.MetadataPatch)
		if sub.Outcome != Continue {
			sub.Reason = fmt.Sprintf("branch step %d: %s", i, sub.Reason)
			sub.MetadataPatch = accumulated
			return sub
		}
	}
	return StepResult{Outcome: Continue, MetadataPatch: accumulated}
}

// --- Stubbed step executors ---
//
// Each returns Continue + logs a warning so flows referencing a planned
// step type don't crash today. Removed once F2.1 lands the real behavior.

// TODO(F2.1): wire require_mfa_challenge (needs a session-bound challenge store
// the step can consult — MVP just continues so pre-existing flows don't break).
func (e *Engine) requireMFAChallenge(_ context.Context, _ *storage.FlowStep, fc *Context) StepResult {
	fc.Logger.Warn("require_mfa_challenge not yet fully implemented",
		"step", "require_mfa_challenge")
	return StepResult{Outcome: Continue}
}

// TODO(F2.1): wire set_metadata to persist via store.UpdateUser; today we
// only stage the mutation in fc.Metadata via MetadataPatch.
func (e *Engine) executeSetMetadata(_ context.Context, step *storage.FlowStep, fc *Context) StepResult {
	fc.Logger.Warn("set_metadata not yet fully implemented", "step", "set_metadata")
	key := strFromConfig(step.Config, "key", "")
	if key == "" {
		// Intentionally soft: a misconfigured stub should still Continue.
		return StepResult{Outcome: Continue}
	}
	value := step.Config["value"]
	return StepResult{Outcome: Continue, MetadataPatch: map[string]any{key: value}}
}

// TODO(F2.1): wire assign_role to call store.AssignRoleToUser once the
// engine owns a transactional boundary for queued effects.
func (e *Engine) executeAssignRole(_ context.Context, _ *storage.FlowStep, fc *Context) StepResult {
	fc.Logger.Warn("assign_role not yet fully implemented", "step", "assign_role")
	return StepResult{Outcome: Continue}
}

// TODO(F2.1): wire add_to_org to call store.CreateOrganizationMember once
// we have a story for invite-vs-direct-add.
func (e *Engine) executeAddToOrg(_ context.Context, _ *storage.FlowStep, fc *Context) StepResult {
	fc.Logger.Warn("add_to_org not yet fully implemented", "step", "add_to_org")
	return StepResult{Outcome: Continue}
}

// TODO(F2.1): wire custom_check — similar to webhook but with a
// configurable pass/fail contract (not just 2xx = pass).
func (e *Engine) executeCustomCheck(_ context.Context, _ *storage.FlowStep, fc *Context) StepResult {
	fc.Logger.Warn("custom_check not yet fully implemented", "step", "custom_check")
	return StepResult{Outcome: Continue}
}

// TODO(F2.1): wire delay — useful for rate-limit simulation and for
// staggered webhook fan-outs. MVP skips the sleep so tests stay fast.
func (e *Engine) executeDelay(_ context.Context, _ *storage.FlowStep, fc *Context) StepResult {
	fc.Logger.Warn("delay not yet fully implemented", "step", "delay")
	return StepResult{Outcome: Continue}
}

// containsSpecial reports whether s has at least one non-alphanumeric
// character. The rule here is intentionally simple — admins who need more
// can layer their own custom_check step on top.
func containsSpecial(s string) bool {
	const special = "!@#$%^&*()_+-=[]{}|;:,.<>?/~`\"'\\"
	return strings.ContainsAny(s, special)
}
