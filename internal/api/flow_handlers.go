package api

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/sharkauth/sharkauth/internal/authflow"
	"github.com/sharkauth/sharkauth/internal/storage"
)

// validAuthFlowTriggers enumerates the trigger names the engine knows how to
// fire. Keeping this in a set rather than a switch gives us O(1) validation
// in handleCreateFlow and handleUpdateFlow without duplicating the list.
var validAuthFlowTriggers = map[string]bool{
	storage.AuthFlowTriggerSignup:        true,
	storage.AuthFlowTriggerLogin:         true,
	storage.AuthFlowTriggerPasswordReset: true,
	storage.AuthFlowTriggerMagicLink:     true,
	storage.AuthFlowTriggerOAuthCallback: true,
}

// supportedFlowStepTypes enumerates the step types the engine dispatcher
// recognises today. Unknown types are rejected at the API layer so that
// dashboards can't silently persist flows that will always fail at runtime.
var supportedFlowStepTypes = map[string]bool{
	"require_email_verification": true,
	"require_mfa_enrollment":     true,
	"require_mfa_challenge":      true,
	"require_password_strength":  true,
	"redirect":                   true,
	"webhook":                    true,
	"set_metadata":               true,
	"assign_role":                true,
	"add_to_org":                 true,
	"custom_check":               true,
	"delay":                      true,
	"conditional":                true,
}

// flowResponse shapes AuthFlow for wire output. No secret fields yet, but
// going through an explicit struct means the day a field like "debug" or
// "internal_config" shows up it lands in storage only until we opt it in.
type flowResponse struct {
	ID         string              `json:"id"`
	Name       string              `json:"name"`
	Trigger    string              `json:"trigger"`
	Steps      []storage.FlowStep  `json:"steps"`
	Enabled    bool                `json:"enabled"`
	Priority   int                 `json:"priority"`
	Conditions map[string]any      `json:"conditions"`
	CreatedAt  time.Time           `json:"created_at"`
	UpdatedAt  time.Time           `json:"updated_at"`
}

func flowToResponse(f *storage.AuthFlow) flowResponse {
	conds := f.Conditions
	if conds == nil {
		conds = map[string]any{}
	}
	steps := f.Steps
	if steps == nil {
		steps = []storage.FlowStep{}
	}
	return flowResponse{
		ID:         f.ID,
		Name:       f.Name,
		Trigger:    f.Trigger,
		Steps:      steps,
		Enabled:    f.Enabled,
		Priority:   f.Priority,
		Conditions: conds,
		CreatedAt:  f.CreatedAt,
		UpdatedAt:  f.UpdatedAt,
	}
}

// flowRunResponse is the on-wire shape of a persisted run. Metadata is
// caller-controlled — the engine only ever writes what step MetadataPatch
// sets, but we document the surface explicitly so future sanitisation has
// a single place to land.
type flowRunResponse struct {
	ID            string         `json:"id"`
	FlowID        string         `json:"flow_id"`
	UserID        string         `json:"user_id,omitempty"`
	Trigger       string         `json:"trigger"`
	Outcome       string         `json:"outcome"`
	BlockedAtStep *int           `json:"blocked_at_step,omitempty"`
	Reason        string         `json:"reason,omitempty"`
	Metadata      map[string]any `json:"metadata"`
	StartedAt     time.Time      `json:"started_at"`
	FinishedAt    time.Time      `json:"finished_at"`
}

func flowRunToResponse(r *storage.AuthFlowRun) flowRunResponse {
	md := r.Metadata
	if md == nil {
		md = map[string]any{}
	}
	return flowRunResponse{
		ID:            r.ID,
		FlowID:        r.FlowID,
		UserID:        r.UserID,
		Trigger:       r.Trigger,
		Outcome:       r.Outcome,
		BlockedAtStep: r.BlockedAtStep,
		Reason:        r.Reason,
		Metadata:      md,
		StartedAt:     r.StartedAt,
		FinishedAt:    r.FinishedAt,
	}
}

// --- requests ---

type createFlowRequest struct {
	Name       string             `json:"name"`
	Trigger    string             `json:"trigger"`
	Steps      []storage.FlowStep `json:"steps"`
	Enabled    *bool              `json:"enabled,omitempty"`
	Priority   int                `json:"priority"`
	Conditions map[string]any     `json:"conditions,omitempty"`
}

type updateFlowRequest struct {
	Name       *string             `json:"name,omitempty"`
	Trigger    *string             `json:"trigger,omitempty"`
	Steps      *[]storage.FlowStep `json:"steps,omitempty"`
	Enabled    *bool               `json:"enabled,omitempty"`
	Priority   *int                `json:"priority,omitempty"`
	Conditions *map[string]any     `json:"conditions,omitempty"`
}

type testFlowRequest struct {
	User     *mockFlowUser  `json:"user"`
	Password string         `json:"password"`
	Metadata map[string]any `json:"metadata"`
}

// mockFlowUser mirrors the fields dry-run steps read off storage.User.
// Admins seed this via the Flow Builder "Test this flow" button; we don't
// force them to construct a complete User row.
type mockFlowUser struct {
	ID            string  `json:"id"`
	Email         string  `json:"email"`
	EmailVerified bool    `json:"email_verified"`
	Name          *string `json:"name,omitempty"`
	MFAEnabled    bool    `json:"mfa_enabled"`
	MFAVerified   bool    `json:"mfa_verified"`
	MFASecret     *string `json:"mfa_secret,omitempty"`
}

func (m *mockFlowUser) toUser() *storage.User {
	if m == nil {
		return nil
	}
	u := &storage.User{
		ID:            m.ID,
		Email:         m.Email,
		EmailVerified: m.EmailVerified,
		Name:          m.Name,
		MFAEnabled:    m.MFAEnabled,
		MFAVerified:   m.MFAVerified,
		MFASecret:     m.MFASecret,
	}
	if u.ID == "" {
		u.ID = "usr_dryrun"
	}
	return u
}

// --- handlers ---

// handleCreateFlow persists a new AuthFlow. Validates trigger + step types up
// front so dashboards surface configuration errors immediately.
func (s *Server) handleCreateFlow(w http.ResponseWriter, r *http.Request) {
	var req createFlowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Invalid JSON body"))
		return
	}

	if err := validateFlowPayload(req.Name, req.Trigger, req.Steps); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_flow", err.Error()))
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	conditions := req.Conditions
	if conditions == nil {
		conditions = map[string]any{}
	}

	now := time.Now().UTC().Truncate(time.Second)
	flow := &storage.AuthFlow{
		ID:         newAuthFlowID(),
		Name:       req.Name,
		Trigger:    req.Trigger,
		Steps:      req.Steps,
		Enabled:    enabled,
		Priority:   req.Priority,
		Conditions: conditions,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.Store.CreateAuthFlow(r.Context(), flow); err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, flowToResponse(flow))
}

// handleListFlows returns all flows, optionally filtered by ?trigger=.
// Response shape matches the other admin list endpoints: {data:[],total:N}.
func (s *Server) handleListFlows(w http.ResponseWriter, r *http.Request) {
	trigger := r.URL.Query().Get("trigger")

	var flows []*storage.AuthFlow
	var err error
	if trigger != "" {
		flows, err = s.Store.ListAuthFlowsByTrigger(r.Context(), trigger)
	} else {
		flows, err = s.Store.ListAuthFlows(r.Context())
	}
	if err != nil {
		internal(w, err)
		return
	}

	out := make([]flowResponse, 0, len(flows))
	for _, f := range flows {
		out = append(out, flowToResponse(f))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data":  out,
		"total": len(out),
	})
}

// handleGetFlow returns a single flow by ID.
func (s *Server) handleGetFlow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	flow, err := s.Store.GetAuthFlowByID(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Flow not found"))
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, flowToResponse(flow))
}

// handleUpdateFlow performs a partial update. Only fields present in the
// body are touched; all others retain their stored values.
func (s *Server) handleUpdateFlow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateFlowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Invalid JSON body"))
		return
	}

	flow, err := s.Store.GetAuthFlowByID(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Flow not found"))
		return
	}
	if err != nil {
		internal(w, err)
		return
	}

	// Apply patch. Each field validated against the same rules as create.
	if req.Name != nil {
		if *req.Name == "" {
			writeJSON(w, http.StatusBadRequest, errPayload("invalid_flow", "name cannot be empty"))
			return
		}
		flow.Name = *req.Name
	}
	if req.Trigger != nil {
		if !validAuthFlowTriggers[*req.Trigger] {
			writeJSON(w, http.StatusBadRequest, errPayload("invalid_flow", "unsupported trigger: "+*req.Trigger))
			return
		}
		flow.Trigger = *req.Trigger
	}
	if req.Steps != nil {
		if err := validateSteps(*req.Steps); err != nil {
			writeJSON(w, http.StatusBadRequest, errPayload("invalid_flow", err.Error()))
			return
		}
		flow.Steps = *req.Steps
	}
	if req.Enabled != nil {
		flow.Enabled = *req.Enabled
	}
	if req.Priority != nil {
		flow.Priority = *req.Priority
	}
	if req.Conditions != nil {
		flow.Conditions = *req.Conditions
	}

	if err := s.Store.UpdateAuthFlow(r.Context(), flow); err != nil {
		internal(w, err)
		return
	}

	// Re-read so response reflects the UPDATE's refreshed updated_at.
	fresh, err := s.Store.GetAuthFlowByID(r.Context(), id)
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, flowToResponse(fresh))
}

// handleDeleteFlow removes a flow. FK ON DELETE CASCADE on auth_flow_runs
// takes care of history rows.
func (s *Server) handleDeleteFlow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Explicit existence check so we can respond 404 vs 204 — DeleteAuthFlow
	// is a no-op on unknown id without surfacing the absence.
	if _, err := s.Store.GetAuthFlowByID(r.Context(), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, errPayload("not_found", "Flow not found"))
			return
		}
		internal(w, err)
		return
	}

	if err := s.Store.DeleteAuthFlow(r.Context(), id); err != nil {
		internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleTestFlow runs a flow against caller-supplied mock data via
// Engine.ExecuteDryRun. No persistence — the test endpoint must never
// pollute the run history the dashboard reads.
func (s *Server) handleTestFlow(w http.ResponseWriter, r *http.Request) {
	if s.FlowEngine == nil {
		writeJSON(w, http.StatusServiceUnavailable, errPayload("not_configured", "Flow engine not initialised"))
		return
	}

	id := chi.URLParam(r, "id")
	flow, err := s.Store.GetAuthFlowByID(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Flow not found"))
		return
	}
	if err != nil {
		internal(w, err)
		return
	}

	var req testFlowRequest
	// Tolerate empty body — admins might just want to smoke the wiring.
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Invalid JSON body"))
			return
		}
	}
	metadata := req.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}

	fc := &authflow.Context{
		Trigger:  flow.Trigger,
		User:     req.User.toUser(),
		Password: req.Password,
		Metadata: metadata,
		Request:  r,
	}

	result, err := s.FlowEngine.ExecuteDryRun(r.Context(), flow, fc)
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// handleListFlowRuns returns history for a flow. ?limit= defaults to 20 and
// is capped at 100 at the handler layer (the store clamps independently at
// 500 as a belt-and-braces defence).
func (s *Server) handleListFlowRuns(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Make sure the flow exists so we can 404 cleanly; otherwise the empty
	// result for a nonexistent id would look identical to "no runs yet".
	if _, err := s.Store.GetAuthFlowByID(r.Context(), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, errPayload("not_found", "Flow not found"))
			return
		}
		internal(w, err)
		return
	}

	limit := 20
	if q := r.URL.Query().Get("limit"); q != "" {
		if v, err := strconv.Atoi(q); err == nil && v > 0 {
			if v > 100 {
				v = 100
			}
			limit = v
		}
	}

	runs, err := s.Store.ListAuthFlowRunsByFlowID(r.Context(), id, limit)
	if err != nil {
		internal(w, err)
		return
	}
	out := make([]flowRunResponse, 0, len(runs))
	for _, rr := range runs {
		out = append(out, flowRunToResponse(rr))
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": out})
}

// runAuthFlow fires a flow for the given trigger. Used by auth handlers.
// Non-fatal: on error or block/redirect outcome, writes the appropriate
// response AND returns handled=true. Callers must stop processing and
// return when handled=true.
func (s *Server) runAuthFlow(w http.ResponseWriter, r *http.Request, trigger string, user *storage.User, password string) (handled bool) {
	if s == nil || s.FlowEngine == nil {
		return false
	}
	fc := &authflow.Context{
		Trigger:  trigger,
		User:     user,
		Password: password,
		Request:  r,
		Metadata: map[string]any{},
	}
	result, err := s.FlowEngine.Execute(r.Context(), fc)
	if err != nil {
		slog.Default().Warn("auth flow execution failed", "trigger", trigger, "err", err)
		return false // non-fatal — let auth proceed
	}
	switch result.Outcome {
	case authflow.Continue:
		return false
	case authflow.Block:
		writeJSON(w, http.StatusForbidden, map[string]string{
			"error":   "flow_blocked",
			"message": result.Reason,
		})
		return true
	case authflow.Redirect:
		// Prefer JSON body with redirect_url for SDK clients; browser callers
		// can follow the Location header.
		w.Header().Set("Location", result.RedirectURL)
		writeJSON(w, http.StatusFound, map[string]string{"redirect_url": result.RedirectURL})
		return true
	case authflow.AwaitMFA:
		// Flow paused at require_mfa_challenge step. Return 401 with challenge_id
		// so the SDK can prompt the user for their TOTP code and then call
		// POST /api/v1/auth/flow/mfa/verify to resume.
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"error":        "mfa_required",
			"mfa_required": true,
			"challenge_id": result.ChallengeID,
			"message":      "MFA verification required to continue",
		})
		return true
	case authflow.Error:
		// Degraded: log + continue so flow errors don't brick auth.
		slog.Default().Error("auth flow errored", "trigger", trigger, "reason", result.Reason)
		return false
	}
	return false
}

// --- helpers ---

// newAuthFlowID mints the "flow_<20-hex>" identifier the schema and
// existing tests both use.
func newAuthFlowID() string {
	buf := make([]byte, 10)
	if _, err := rand.Read(buf); err != nil {
		return "flow_" + hex.EncodeToString([]byte(time.Now().UTC().Format("20060102150405")))
	}
	return "flow_" + hex.EncodeToString(buf)
}

// validateFlowPayload runs the full create-time ruleset: name, trigger, and
// steps. Update reuses validateSteps + inline trigger + name checks.
func validateFlowPayload(name, trigger string, steps []storage.FlowStep) error {
	if name == "" {
		return errors.New("name is required")
	}
	if !validAuthFlowTriggers[trigger] {
		return errors.New("unsupported trigger: " + trigger)
	}
	return validateSteps(steps)
}

// validateSteps checks the slice is non-empty and every step has a known
// Type. Branch steps (type=conditional) recursively validate their nested
// then/else blocks — a malformed conditional is just as fatal as a bad
// top-level step.
func validateSteps(steps []storage.FlowStep) error {
	if len(steps) == 0 {
		return errors.New("steps must contain at least one element")
	}
	return validateStepsRecursive(steps)
}

func validateStepsRecursive(steps []storage.FlowStep) error {
	for i, step := range steps {
		if step.Type == "" {
			return errors.New("step " + strconv.Itoa(i) + " missing type")
		}
		if !supportedFlowStepTypes[step.Type] {
			return errors.New("step " + strconv.Itoa(i) + " unsupported type: " + step.Type)
		}
		if step.Type == "conditional" {
			if len(step.ThenSteps) == 0 && len(step.ElseSteps) == 0 {
				return errors.New("conditional step " + strconv.Itoa(i) + " needs at least one branch")
			}
			if err := validateStepsRecursive(step.ThenSteps); err != nil {
				return err
			}
			if err := validateStepsRecursive(step.ElseSteps); err != nil {
				return err
			}
		}
	}
	return nil
}
