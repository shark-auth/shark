package storage

import "time"

// Auth flow trigger names. The same set is enforced at the API layer.
const (
	AuthFlowTriggerSignup        = "signup"
	AuthFlowTriggerLogin         = "login"
	AuthFlowTriggerPasswordReset = "password_reset"
	AuthFlowTriggerMagicLink     = "magic_link"
	AuthFlowTriggerOAuthCallback = "oauth_callback"
)

// Auth flow run outcome labels.
const (
	AuthFlowOutcomeContinue = "continue"
	AuthFlowOutcomeBlock    = "block"
	AuthFlowOutcomeRedirect = "redirect"
	AuthFlowOutcomeError    = "error"
)

// AuthFlow is an ordered pipeline of FlowSteps that run at a specific trigger
// point (signup, login, etc). Admins compose flows in the dashboard; the engine
// (F2) walks the Steps array in order. Higher Priority flows win when multiple
// flows match the same trigger, tie-broken by CreatedAt.
//
// Steps and Conditions are JSON-encoded in the DB; callers always work with the
// typed Go representations below.
type AuthFlow struct {
	ID         string         `json:"id"`          // flow_<hex>
	Name       string         `json:"name"`
	Trigger    string         `json:"trigger"`     // see AuthFlowTrigger* constants
	Steps      []FlowStep     `json:"steps"`
	Enabled    bool           `json:"enabled"`
	Priority   int            `json:"priority"`    // higher = evaluated first
	Conditions map[string]any `json:"conditions"`  // when this flow applies (e.g. {"user.org_id": "org_123"})
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

// FlowStep is one node in a flow pipeline. A step can either be a leaf action
// (Type != "conditional") or a branch (Type == "conditional") with a Condition
// expression plus ThenSteps/ElseSteps that recursively nest FlowSteps.
//
// The engine (F2) interprets Type + Config; storage just round-trips the JSON.
type FlowStep struct {
	Type   string         `json:"type"`             // require_email_verification | require_mfa_enrollment | redirect | webhook | conditional | ...
	Config map[string]any `json:"config,omitempty"`

	// Branch-only fields (when Type == "conditional").
	Condition string     `json:"condition,omitempty"` // expression, e.g. "user.email_domain == 'acme.com'"
	ThenSteps []FlowStep `json:"then,omitempty"`
	ElseSteps []FlowStep `json:"else,omitempty"`
}

// AuthFlowRun records a single evaluation of a flow against a trigger. Rows
// are append-only; the dashboard "History" tab (F4) reads recent rows per
// flow to show outcomes and where blocks happened.
//
// UserID is empty for pre-signup triggers; BlockedAtStep is nil unless the
// run halted on a require_* step.
type AuthFlowRun struct {
	ID            string         `json:"id"`         // fr_<hex>
	FlowID        string         `json:"flow_id"`
	UserID        string         `json:"user_id,omitempty"`   // may be empty (pre-signup)
	Trigger       string         `json:"trigger"`
	Outcome       string         `json:"outcome"`    // see AuthFlowOutcome* constants
	BlockedAtStep *int           `json:"blocked_at_step,omitempty"`
	Reason        string         `json:"reason,omitempty"`
	Metadata      map[string]any `json:"metadata"`
	StartedAt     time.Time      `json:"started_at"`
	FinishedAt    time.Time      `json:"finished_at"`
}
