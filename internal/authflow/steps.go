package authflow

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/shark-auth/shark/internal/storage"
)

// executeStep dispatches a single FlowStep to its typed executor.
//
// Returning an Error outcome here short-circuits the entire flow at the
// engine loop â€” individual steps must not swallow a hard failure by
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
// Redirect outcome â€” handy for sending users to a verification page
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
// user flows past â€” a value of false is useful for admin-only flows that
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
// honour â€” this engine doesn't sleep on it.
func (e *Engine) executeRedirect(_ context.Context, step *storage.FlowStep, _ *Context) StepResult {
	url := strFromConfig(step.Config, "url", "")
	if url == "" {
		return StepResult{Outcome: Error, Reason: "redirect step missing url"}
	}
	return StepResult{Outcome: Redirect, RedirectURL: url, Reason: "redirect"}
}

// executeWebhook POSTs (or other configured method) a JSON payload to the
// configured URL and continues on 2xx. 4xx/5xx and network/timeout errors
// produce an Error outcome â€” callers decide whether that should block auth
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

// requireMFAChallenge pauses the flow for an MFA TOTP challenge.
//
// If the user already has MFA enrolled (MFAEnabled && MFAVerified), a challenge
// entry is minted in GlobalChallengeStore and the step returns AwaitMFA with
// the challenge ID. The HTTP layer converts that outcome to a 401 with
// mfa_required: true so the client can call POST /api/v1/auth/flow/mfa/verify.
//
// If MFA is not enrolled: the step errors with "mfa_required_but_not_enrolled".
// Wrap the step in a conditional if you want a soft-fail path.
//
// Config:
//
//	allow_recovery (bool, default true) â€” no-op placeholder for v0.2 when
//	recovery-code bypass is wired; kept in the schema now so existing configs
//	don't break when the feature lands.
func (e *Engine) requireMFAChallenge(_ context.Context, step *storage.FlowStep, fc *Context) StepResult {
	if fc.User == nil {
		return StepResult{Outcome: Error, Reason: "require_mfa_challenge: no user context"}
	}

	enrolled := fc.User.MFAEnabled && fc.User.MFAVerified
	if !enrolled {
		return StepResult{Outcome: Error, Reason: "mfa_required_but_not_enrolled"}
	}

	// _ = boolFromConfig(step.Config, "allow_recovery", true) // reserved for v0.2
	_ = step // config read reserved for v0.2

	challengeID := GlobalChallengeStore.Issue(fc.User.ID, "")

	fc.Logger.Info("require_mfa_challenge: challenge issued",
		"user_id", fc.User.ID,
		"challenge_id", challengeID)

	return StepResult{
		Outcome:     AwaitMFA,
		Reason:      "mfa_challenge_required",
		ChallengeID: challengeID,
	}
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

// executeAssignRole attaches a global role to the running user.
//
// Config:
//
//	role_id  (string, required) â€” ID of the role to assign.
//	org_id   (string, optional) â€” reserved for v0.2 org-scoped assignment;
//	                               ignored today because AssignRoleToUserInOrg
//	                               is not yet in the storage interface.
//
// Idempotency: the underlying INSERT OR IGNORE means assigning an already-held
// role is a no-op at the DB level and returns Continue.
//
// Failure: role not found â†’ Error; store failure â†’ Error.
func (e *Engine) executeAssignRole(ctx context.Context, step *storage.FlowStep, fc *Context) StepResult {
	if fc.User == nil {
		return StepResult{Outcome: Error, Reason: "assign_role: no user context"}
	}

	roleID := strFromConfig(step.Config, "role_id", "")
	if roleID == "" {
		return StepResult{Outcome: Error, Reason: "assign_role: missing role_id in config"}
	}

	// Validate the role exists before assigning.
	if _, err := e.store.GetRoleByID(ctx, roleID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return StepResult{Outcome: Error, Reason: fmt.Sprintf("assign_role: role %q not found", roleID)}
		}
		return StepResult{Outcome: Error, Reason: "assign_role: store error: " + err.Error()}
	}

	if err := e.store.AssignRoleToUser(ctx, fc.User.ID, roleID); err != nil {
		return StepResult{Outcome: Error, Reason: "assign_role: " + err.Error()}
	}

	_ = e.emitAuditLog(ctx, fc, "authflow.step.assign_role", map[string]any{
		"role_id": roleID,
		"user_id": fc.User.ID,
	})

	fc.Logger.Info("assign_role: role assigned",
		"user_id", fc.User.ID,
		"role_id", roleID)

	return StepResult{Outcome: Continue}
}

// executeAddToOrg adds the running user to an organization.
//
// Config:
//
//	org_id   (string, required) â€” ID of the target organization.
//	role_id  (string, optional) â€” org role to assign; defaults to "viewer".
//	                               Note: this maps to OrganizationMember.Role,
//	                               not the global roles table.
//
// Idempotency: if the user is already a member the step returns Continue
// without error (idempotent add).
func (e *Engine) executeAddToOrg(ctx context.Context, step *storage.FlowStep, fc *Context) StepResult {
	if fc.User == nil {
		return StepResult{Outcome: Error, Reason: "add_to_org: no user context"}
	}

	orgID := strFromConfig(step.Config, "org_id", "")
	if orgID == "" {
		return StepResult{Outcome: Error, Reason: "add_to_org: missing org_id in config"}
	}

	roleID := strFromConfig(step.Config, "role_id", "viewer")

	// Idempotency check: already a member â†’ succeed silently.
	existing, err := e.store.GetOrganizationMember(ctx, orgID, fc.User.ID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return StepResult{Outcome: Error, Reason: "add_to_org: store error: " + err.Error()}
	}
	if existing != nil {
		fc.Logger.Info("add_to_org: user already member (idempotent)",
			"user_id", fc.User.ID, "org_id", orgID)
		return StepResult{Outcome: Continue}
	}

	member := &storage.OrganizationMember{
		OrganizationID: orgID,
		UserID:         fc.User.ID,
		Role:           roleID,
		JoinedAt:       time.Now().UTC().Format(time.RFC3339),
	}
	if err := e.store.CreateOrganizationMember(ctx, member); err != nil {
		return StepResult{Outcome: Error, Reason: "add_to_org: " + err.Error()}
	}

	_ = e.emitAuditLog(ctx, fc, "authflow.step.add_to_org", map[string]any{
		"org_id":  orgID,
		"role_id": roleID,
		"user_id": fc.User.ID,
	})

	fc.Logger.Info("add_to_org: member added",
		"user_id", fc.User.ID,
		"org_id", orgID,
		"role", roleID)

	return StepResult{Outcome: Continue}
}

// TODO(F2.1): wire custom_check â€” similar to webhook but with a
// configurable pass/fail contract (not just 2xx = pass).
func (e *Engine) executeCustomCheck(_ context.Context, _ *storage.FlowStep, fc *Context) StepResult {
	fc.Logger.Warn("custom_check not yet fully implemented", "step", "custom_check")
	return StepResult{Outcome: Continue}
}

// TODO(F2.1): wire delay â€” useful for rate-limit simulation and for
// staggered webhook fan-outs. MVP skips the sleep so tests stay fast.
func (e *Engine) executeDelay(_ context.Context, _ *storage.FlowStep, fc *Context) StepResult {
	fc.Logger.Warn("delay not yet fully implemented", "step", "delay")
	return StepResult{Outcome: Continue}
}

// emitAuditLog persists a single audit entry for a step side effect. Errors
// are logged but not propagated â€” a dropped audit row must never block auth.
func (e *Engine) emitAuditLog(ctx context.Context, fc *Context, action string, meta map[string]any) error {
	metaJSON, _ := json.Marshal(meta)

	buf := make([]byte, 8)
	_, _ = rand.Read(buf)
	id := "al_" + hex.EncodeToString(buf)

	actorID := ""
	if fc.User != nil {
		actorID = fc.User.ID
	}

	// TargetID anchors the audit row to a specific subject so dashboard
	// filters can drill in. For authflow steps the user is the only stable
	// scope plumbed through Context (flow_id and run_id sit on the engine
	// loop, not the StepContext) â€” fall back to "" if no user is bound.
	targetID := ""
	if fc.User != nil {
		targetID = fc.User.ID
	}

	entry := &storage.AuditLog{
		ID:         id,
		ActorID:    actorID,
		ActorType:  "user",
		Action:     action,
		TargetType: "user",
		TargetID:   targetID,
		Metadata:   string(metaJSON),
		Status:     "success",
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}
	if err := e.store.CreateAuditLog(ctx, entry); err != nil {
		e.logger.Warn("authflow: audit log write failed",
			"action", action, "err", err)
		return err
	}
	return nil
}

// containsSpecial reports whether s has at least one non-alphanumeric
// character. The rule here is intentionally simple â€” admins who need more
// can layer their own custom_check step on top.
func containsSpecial(s string) bool {
	const special = "!@#$%^&*()_+-=[]{}|;:,.<>?/~`\"'\\"
	return strings.ContainsAny(s, special)
}
