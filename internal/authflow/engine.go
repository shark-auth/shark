// Package authflow is the Phase 6 Auth Flow execution engine.
//
// F1 (storage layer) shipped the AuthFlow / FlowStep / AuthFlowRun types plus
// SQLite persistence. This package is F2: the runtime that selects a flow for
// a trigger, walks its steps in order, and reports a Result back to the
// caller. Admin-configured flows live in the DB; step Type + Config strings
// are interpreted here without code changes — wire once, configure many.
//
// Design notes
//
//   - Steps read from Context and return StepResult. They MUST NOT mutate the
//     DB directly; side effects are queued through Result.Metadata and (in
//     later work) explicit effect slices so the caller decides when/whether
//     to persist.
//   - The engine never panics on nil inputs: nil user, nil metadata map, nil
//     http.Client and nil logger all default to safe values.
//   - Timeline entries are populated for every executed step — including
//     steps that Block or Error — so the dashboard History tab can show
//     exactly where a run halted.
//   - Run records are persisted on Execute; ExecuteDryRun skips persistence
//     so the admin "Test this flow" button doesn't pollute history.
package authflow

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/sharkauth/sharkauth/internal/storage"
)

// Outcome is what the engine decides after running a flow.
type Outcome string

// Outcome values.
const (
	Continue Outcome = "continue"    // flow completed, allow normal flow to proceed
	Block    Outcome = "block"       // flow blocked execution (e.g., email unverified)
	Redirect Outcome = "redirect"    // flow requests a redirect to RedirectURL
	Error    Outcome = "error"       // runtime error (webhook timeout, etc.)
	AwaitMFA Outcome = "awaiting_mfa" // flow paused — MFA challenge issued, waiting for TOTP code
)

// Context carries per-execution state into each step.
//
// Steps read fields from Context and return StepResult telling the
// dispatcher what to do next. Steps must NOT mutate the underlying user in
// the DB directly; they queue effects via StepResult.MetadataPatch so the
// caller decides when/whether to persist.
type Context struct {
	Trigger   string          // signup | login | password_reset | magic_link | oauth_callback
	User      *storage.User   // may be nil for pre-signup validation
	Password  string          // only populated for signup / password_reset flows
	Request   *http.Request   // for IP, UA, headers (may be nil)
	Metadata  map[string]any  // stage-accumulated state (readable by later steps)
	StartedAt time.Time       // when the flow kicked off; engine fills if zero
	Logger    *slog.Logger    // per-request logger (may be nil → engine default)
	UserRoles []string        // role names assigned to the user (for user_has_role)
}

// StepResult is what each executor returns to the dispatcher.
type StepResult struct {
	Outcome       Outcome
	Reason        string         // human readable; populates Result.Reason on Block/Error
	RedirectURL   string         // populated when Outcome == Redirect
	MetadataPatch map[string]any // merged into Context.Metadata + surfaced in Result
	ChallengeID   string         // populated when Outcome == AwaitMFA
}

// Result is the aggregate of a whole flow run.
type Result struct {
	Outcome       Outcome             `json:"outcome"`
	Reason        string              `json:"reason,omitempty"`
	RedirectURL   string              `json:"redirect_url,omitempty"`
	ChallengeID   string              `json:"challenge_id,omitempty"` // set when Outcome == AwaitMFA
	BlockedAtStep *int                `json:"blocked_at_step,omitempty"`
	Metadata      map[string]any      `json:"metadata"`
	Timeline      []StepTimelineEntry `json:"timeline"`
	StartedAt     time.Time           `json:"started_at"`
	FinishedAt    time.Time           `json:"finished_at"`
	FlowID        string              `json:"flow_id,omitempty"` // id of flow that ran; empty if none matched
}

// StepTimelineEntry is a single step's outcome on the Result.Timeline slice,
// used by the Flow Builder History tab.
type StepTimelineEntry struct {
	Index     int           `json:"index"`
	Type      string        `json:"type"`
	Outcome   Outcome       `json:"outcome"`
	Reason    string        `json:"reason,omitempty"`
	StartedAt time.Time     `json:"started_at"`
	Duration  time.Duration `json:"duration_ns"`
}

// Engine executes flows against a storage.Store.
//
// An Engine is safe for concurrent use: all mutable state lives on the
// Context passed to Execute.
type Engine struct {
	store  storage.Store
	logger *slog.Logger
	http   *http.Client     // webhook + custom_check client; configurable timeout
	now    func() time.Time // injectable for deterministic timeline tests
}

// NewEngine constructs an Engine with the given store and logger.
//
// A nil logger falls back to slog.Default(). The HTTP client used for
// webhook / custom_check steps is a fresh http.Client with a 30s hard cap;
// per-step timeouts are enforced via request contexts.
func NewEngine(store storage.Store, logger *slog.Logger) *Engine {
	if logger == nil {
		logger = slog.Default()
	}
	return &Engine{
		store:  store,
		logger: logger,
		http:   &http.Client{Timeout: 30 * time.Second},
		now:    time.Now,
	}
}

// WithHTTPClient returns a shallow copy of the Engine using c for outbound
// HTTP. Tests point this at httptest.NewServer's client.
func (e *Engine) WithHTTPClient(c *http.Client) *Engine {
	cp := *e
	if c != nil {
		cp.http = c
	}
	return &cp
}

// WithNow returns a shallow copy of the Engine using fn as the clock.
// Tests use this so timeline entries are deterministic.
func (e *Engine) WithNow(fn func() time.Time) *Engine {
	cp := *e
	if fn != nil {
		cp.now = fn
	}
	return &cp
}

// Execute finds flows for the trigger, selects the highest-priority
// matching one (conditions evaluated top-down), runs its steps in sequence,
// persists a run record, and returns the aggregate Result.
//
// If no enabled flow exists for the trigger (or no conditions match),
// Execute returns {Outcome: Continue} with no persistence — the caller
// proceeds as if no flow were configured.
//
// On step Block / Error / Redirect the flow short-circuits: subsequent
// steps are not run, and Result.BlockedAtStep marks the index.
func (e *Engine) Execute(ctx context.Context, fc *Context) (*Result, error) {
	e.prepareContext(fc)

	result := &Result{
		Outcome:   Continue,
		StartedAt: e.now(),
		Metadata:  map[string]any{},
		Timeline:  []StepTimelineEntry{},
	}

	flows, err := e.store.ListAuthFlowsByTrigger(ctx, fc.Trigger)
	if err != nil {
		return nil, err
	}

	var flow *storage.AuthFlow
	for _, candidate := range flows {
		if candidate == nil || !candidate.Enabled {
			continue
		}
		ok, err := Evaluate(candidate.Conditions, fc)
		if err != nil {
			e.logger.Warn("skipping flow with invalid conditions",
				"flow_id", candidate.ID, "err", err)
			continue
		}
		if ok {
			flow = candidate
			break
		}
	}

	if flow == nil {
		result.FinishedAt = e.now()
		return result, nil // no matching flow → Continue
	}

	result.FlowID = flow.ID

	// Seed accumulated metadata with anything the caller already staged.
	for k, v := range fc.Metadata {
		result.Metadata[k] = v
	}

	e.runSteps(ctx, flow.Steps, fc, result)
	result.FinishedAt = e.now()

	if err := e.persistRun(ctx, flow, fc, result); err != nil {
		e.logger.Warn("persist auth flow run failed",
			"flow_id", flow.ID, "err", err)
	}

	return result, nil
}

// ExecuteDryRun runs a flow but does NOT persist an auth_flow_runs record.
//
// Used by the /admin/flows/{id}/test endpoint so admins can preview a flow
// against seeded mock data without polluting history. The flow is taken
// directly (no lookup) and its Conditions are ignored — the caller has
// already decided it's the one they want to test.
func (e *Engine) ExecuteDryRun(ctx context.Context, flow *storage.AuthFlow, fc *Context) (*Result, error) {
	e.prepareContext(fc)

	result := &Result{
		Outcome:   Continue,
		StartedAt: e.now(),
		Metadata:  map[string]any{},
		Timeline:  []StepTimelineEntry{},
	}
	if flow == nil {
		result.FinishedAt = e.now()
		return result, nil
	}
	result.FlowID = flow.ID

	for k, v := range fc.Metadata {
		result.Metadata[k] = v
	}

	e.runSteps(ctx, flow.Steps, fc, result)
	result.FinishedAt = e.now()
	return result, nil
}

// runSteps is the shared loop used by both Execute and ExecuteDryRun.
func (e *Engine) runSteps(ctx context.Context, steps []storage.FlowStep, fc *Context, result *Result) {
	for i, step := range steps {
		step := step // local copy; address taken below
		sub := e.executeStepWithTiming(ctx, &step, fc, i, result)
		mergeMetadata(fc, sub.MetadataPatch)
		for k, v := range sub.MetadataPatch {
			result.Metadata[k] = v
		}
		if sub.Outcome != Continue {
			result.Outcome = sub.Outcome
			result.Reason = sub.Reason
			result.RedirectURL = sub.RedirectURL
			result.ChallengeID = sub.ChallengeID
			idx := i
			result.BlockedAtStep = &idx
			return
		}
	}
}

// executeStepWithTiming wraps executeStep so each dispatch appends a
// Timeline entry — regardless of Continue / Block / Error outcome.
func (e *Engine) executeStepWithTiming(ctx context.Context, step *storage.FlowStep, fc *Context, index int, result *Result) StepResult {
	started := e.now()
	sub := e.executeStep(ctx, step, fc)
	result.Timeline = append(result.Timeline, StepTimelineEntry{
		Index:     index,
		Type:      step.Type,
		Outcome:   sub.Outcome,
		Reason:    sub.Reason,
		StartedAt: started,
		Duration:  e.now().Sub(started),
	})
	return sub
}

// prepareContext fills in zero-valued defaults so steps never see nils.
func (e *Engine) prepareContext(fc *Context) {
	if fc == nil {
		return
	}
	if fc.Metadata == nil {
		fc.Metadata = map[string]any{}
	}
	if fc.Logger == nil {
		fc.Logger = e.logger
	}
	if fc.StartedAt.IsZero() {
		fc.StartedAt = e.now()
	}
}

// mergeMetadata copies patch into fc.Metadata. Safe on nil patch.
func mergeMetadata(fc *Context, patch map[string]any) {
	if len(patch) == 0 {
		return
	}
	if fc.Metadata == nil {
		fc.Metadata = map[string]any{}
	}
	for k, v := range patch {
		fc.Metadata[k] = v
	}
}

// persistRun writes an auth_flow_runs row for the completed flow. Never
// returns an error to the caller — storage failures are logged and the HTTP
// response still flows through (a dropped history row must not block auth).
func (e *Engine) persistRun(ctx context.Context, flow *storage.AuthFlow, fc *Context, result *Result) error {
	run := &storage.AuthFlowRun{
		ID:            newRunID(),
		FlowID:        flow.ID,
		Trigger:       fc.Trigger,
		Outcome:       string(result.Outcome),
		BlockedAtStep: result.BlockedAtStep,
		Reason:        result.Reason,
		Metadata:      result.Metadata,
		StartedAt:     result.StartedAt,
		FinishedAt:    result.FinishedAt,
	}
	if fc.User != nil {
		run.UserID = fc.User.ID
	}
	return e.store.CreateAuthFlowRun(ctx, run)
}

// newRunID mints an "fr_<20-hex>" id matching the schema the storage layer
// already accepts.
func newRunID() string {
	buf := make([]byte, 10)
	if _, err := rand.Read(buf); err != nil {
		// rand.Read on modern Go effectively cannot fail; fall back to a
		// timestamp-shaped suffix so we never return an empty id.
		return "fr_" + hex.EncodeToString([]byte(time.Now().UTC().Format("20060102150405")))
	}
	return "fr_" + hex.EncodeToString(buf)
}

// --- config accessor helpers (used by step executors) ---

// strFromConfig reads a string key with a default.
func strFromConfig(cfg map[string]any, key, def string) string {
	if cfg == nil {
		return def
	}
	if v, ok := cfg[key].(string); ok && v != "" {
		return v
	}
	return def
}

// intFromConfig reads an int key with a default. JSON numbers deserialize as
// float64, so we also accept those.
func intFromConfig(cfg map[string]any, key string, def int) int {
	if cfg == nil {
		return def
	}
	switch v := cfg[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	}
	return def
}

// boolFromConfig reads a bool key with a default.
func boolFromConfig(cfg map[string]any, key string, def bool) bool {
	if cfg == nil {
		return def
	}
	if v, ok := cfg[key].(bool); ok {
		return v
	}
	return def
}

// secsFromConfig returns a positive timeout capped at max.
func secsFromConfig(cfg map[string]any, key string, def, max int) int {
	v := intFromConfig(cfg, key, def)
	if v <= 0 {
		return def
	}
	if v > max {
		return max
	}
	return v
}

// sanitizeUser makes a webhook-safe copy of a User — PasswordHash and
// MFASecret are cleared so a misbehaving webhook endpoint can never leak a
// credential.
func sanitizeUser(u *storage.User) map[string]any {
	if u == nil {
		return nil
	}
	out := map[string]any{
		"id":             u.ID,
		"email":          u.Email,
		"email_verified": u.EmailVerified,
		"mfa_enabled":    u.MFAEnabled,
		"mfa_verified":   u.MFAVerified,
		"created_at":     u.CreatedAt,
		"updated_at":     u.UpdatedAt,
	}
	if u.Name != nil {
		out["name"] = *u.Name
	}
	if u.AvatarURL != nil {
		out["avatar_url"] = *u.AvatarURL
	}
	if u.LastLoginAt != nil {
		out["last_login_at"] = *u.LastLoginAt
	}
	return out
}

// readAndDiscard drains an http.Response body so the connection can be
// reused, returning any error so callers can log it.
func readAndDiscard(body io.Reader) error {
	_, err := io.Copy(io.Discard, body)
	return err
}

// conditionMap parses a step.Condition string into a predicate map. Empty
// strings are treated as "always match" for forward-compat with flows that
// don't bother filling in a branch condition.
func conditionMap(condition string) (map[string]any, error) {
	if condition == "" {
		return map[string]any{}, nil
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(condition), &m); err != nil {
		// Don't surface raw JSON errors to dashboards; wrap with context.
		return nil, err
	}
	return m, nil
}
