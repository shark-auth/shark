package api

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/sharkauth/sharkauth/internal/proxy"
	"github.com/sharkauth/sharkauth/internal/storage"
)

// proxyRuleResponse is the JSON projection of storage.ProxyRule. Mirrors the
// DB row shape one-to-one — the wire layer doesn't hide any field, since the
// dashboard's edit form needs every column to round-trip cleanly.
type proxyRuleResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Pattern   string    `json:"pattern"`
	Methods   []string  `json:"methods"`
	Require   string    `json:"require"`
	Allow     string    `json:"allow"`
	Scopes    []string  `json:"scopes"`
	Enabled   bool      `json:"enabled"`
	Priority  int       `json:"priority"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func proxyRuleToResponse(r *storage.ProxyRule) proxyRuleResponse {
	methods := r.Methods
	if methods == nil {
		methods = []string{}
	}
	scopes := r.Scopes
	if scopes == nil {
		scopes = []string{}
	}
	return proxyRuleResponse{
		ID:        r.ID,
		Name:      r.Name,
		Pattern:   r.Pattern,
		Methods:   methods,
		Require:   r.Require,
		Allow:     r.Allow,
		Scopes:    scopes,
		Enabled:   r.Enabled,
		Priority:  r.Priority,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

// --- requests ---

type createProxyRuleRequest struct {
	Name     string   `json:"name"`
	Pattern  string   `json:"pattern"`
	Methods  []string `json:"methods,omitempty"`
	Require  string   `json:"require,omitempty"`
	Allow    string   `json:"allow,omitempty"`
	Scopes   []string `json:"scopes,omitempty"`
	Enabled  *bool    `json:"enabled,omitempty"`
	Priority int      `json:"priority,omitempty"`
}

type updateProxyRuleRequest struct {
	Name     *string   `json:"name,omitempty"`
	Pattern  *string   `json:"pattern,omitempty"`
	Methods  *[]string `json:"methods,omitempty"`
	Require  *string   `json:"require,omitempty"`
	Allow    *string   `json:"allow,omitempty"`
	Scopes   *[]string `json:"scopes,omitempty"`
	Enabled  *bool     `json:"enabled,omitempty"`
	Priority *int      `json:"priority,omitempty"`
}

// --- handlers ---

// handleListProxyRules returns every DB-backed override rule. Always
// available (regardless of proxy enable state) so admins can stage rules
// before flipping the proxy on. Response shape matches other admin lists:
// {data:[], total:N}.
func (s *Server) handleListProxyRules(w http.ResponseWriter, r *http.Request) {
	rules, err := s.Store.ListProxyRules(r.Context())
	if err != nil {
		internal(w, err)
		return
	}
	out := make([]proxyRuleResponse, 0, len(rules))
	for _, rule := range rules {
		out = append(out, proxyRuleToResponse(rule))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data":  out,
		"total": len(out),
	})
}

// handleCreateProxyRule persists a new override rule. Validates the
// require/allow pair via the same compile path the engine uses so a row
// that survives Create is guaranteed to load cleanly on engine refresh.
func (s *Server) handleCreateProxyRule(w http.ResponseWriter, r *http.Request) {
	var req createProxyRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Invalid JSON body"))
		return
	}

	if err := validateProxyRulePayload(req.Name, req.Pattern, req.Methods, req.Require, req.Allow); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_proxy_rule", err.Error()))
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	now := time.Now().UTC().Truncate(time.Second)
	rule := &storage.ProxyRule{
		ID:        newProxyRuleID(),
		Name:      req.Name,
		Pattern:   req.Pattern,
		Methods:   normalizeMethods(req.Methods),
		Require:   req.Require,
		Allow:     req.Allow,
		Scopes:    req.Scopes,
		Enabled:   enabled,
		Priority:  req.Priority,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.Store.CreateProxyRule(r.Context(), rule); err != nil {
		internal(w, err)
		return
	}

	// Push the new rule set into the live engine. A refresh failure is
	// non-fatal for the create (the row is already persisted) — surface it
	// in the response so the operator can investigate, but keep the 201.
	refreshErr := s.refreshProxyEngineFromDB(r.Context())

	resp := map[string]any{"data": proxyRuleToResponse(rule)}
	if refreshErr != nil {
		resp["engine_refresh_error"] = refreshErr.Error()
	}
	writeJSON(w, http.StatusCreated, resp)
}

// handleGetProxyRule returns a single rule by id, 404 when missing.
func (s *Server) handleGetProxyRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rule, err := s.Store.GetProxyRuleByID(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Proxy rule not found"))
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, proxyRuleToResponse(rule))
}

// handleUpdateProxyRule applies a partial update. Each provided field is
// validated against the same rules as Create.
func (s *Server) handleUpdateProxyRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateProxyRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Invalid JSON body"))
		return
	}

	rule, err := s.Store.GetProxyRuleByID(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Proxy rule not found"))
		return
	}
	if err != nil {
		internal(w, err)
		return
	}

	if req.Name != nil {
		if strings.TrimSpace(*req.Name) == "" {
			writeJSON(w, http.StatusBadRequest, errPayload("invalid_proxy_rule", "name cannot be empty"))
			return
		}
		rule.Name = *req.Name
	}
	if req.Pattern != nil {
		rule.Pattern = *req.Pattern
	}
	if req.Methods != nil {
		rule.Methods = normalizeMethods(*req.Methods)
	}
	if req.Require != nil {
		rule.Require = *req.Require
	}
	if req.Allow != nil {
		rule.Allow = *req.Allow
	}
	if req.Scopes != nil {
		rule.Scopes = *req.Scopes
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}
	if req.Priority != nil {
		rule.Priority = *req.Priority
	}

	// Re-validate the full row before persisting — partial PATCH could land
	// us in a state Create would have rejected (e.g. Require + Allow both
	// set). Fail closed: the row stays unmodified.
	if err := validateProxyRulePayload(rule.Name, rule.Pattern, rule.Methods, rule.Require, rule.Allow); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_proxy_rule", err.Error()))
		return
	}

	if err := s.Store.UpdateProxyRule(r.Context(), rule); err != nil {
		internal(w, err)
		return
	}

	// Refresh the engine so the patched row goes live immediately. As with
	// Create, a refresh failure is reported but doesn't roll back the DB.
	refreshErr := s.refreshProxyEngineFromDB(r.Context())

	fresh, err := s.Store.GetProxyRuleByID(r.Context(), id)
	if err != nil {
		internal(w, err)
		return
	}
	resp := map[string]any{"data": proxyRuleToResponse(fresh)}
	if refreshErr != nil {
		resp["engine_refresh_error"] = refreshErr.Error()
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleDeleteProxyRule removes a rule and refreshes the engine. 404 when
// the id doesn't exist (so dashboards can distinguish a real success from a
// stale id).
func (s *Server) handleDeleteProxyRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if _, err := s.Store.GetProxyRuleByID(r.Context(), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, errPayload("not_found", "Proxy rule not found"))
			return
		}
		internal(w, err)
		return
	}

	if err := s.Store.DeleteProxyRule(r.Context(), id); err != nil {
		internal(w, err)
		return
	}

	// Refresh the engine. Best-effort — the row is already gone; failing to
	// refresh just means the live engine still has the stale rule until the
	// next mutation or restart.
	_ = s.refreshProxyEngineFromDB(r.Context())

	w.WriteHeader(http.StatusNoContent)
}

// --- engine wiring ---

// refreshProxyEngineFromDB recompiles the proxy engine's rule set from
// (YAML rules ++ DB rules), with DB rules layered first since they have
// higher priority by virtue of admin intent. No-op when the proxy engine
// isn't initialised (proxy disabled in YAML).
//
// Both YAML and DB rules are sorted into a single slice ordered by the
// rule's Priority (DESC) — DB rows default to priority 0 unless the admin
// bumps them, and YAML rows are also priority 0, so for unbumped rules the
// effective ordering is DB-rows-first then YAML, which matches the override
// intent: a DB row with the same path as a YAML row wins via first-match.
func (s *Server) refreshProxyEngineFromDB(ctx context.Context) error {
	if s.ProxyEngine == nil {
		return nil
	}

	rows, err := s.Store.ListProxyRules(ctx)
	if err != nil {
		return err
	}

	// Compose specs: DB rules (enabled only) first, then YAML rules. Within
	// each group we preserve list order; ListProxyRules already sorts by
	// priority DESC + created_at ASC.
	specs := make([]proxy.RuleSpec, 0, len(rows)+len(s.Config.Proxy.Rules))
	for _, r := range rows {
		if !r.Enabled {
			continue
		}
		specs = append(specs, proxy.RuleSpec{
			Path:    r.Pattern,
			Methods: r.Methods,
			Require: r.Require,
			Allow:   r.Allow,
			Scopes:  r.Scopes,
		})
	}
	for _, pr := range s.Config.Proxy.Rules {
		specs = append(specs, proxy.RuleSpec{
			Path:    pr.Path,
			Methods: pr.Methods,
			Require: pr.Require,
			Allow:   pr.Allow,
			Scopes:  pr.Scopes,
		})
	}

	return s.ProxyEngine.SetRules(specs)
}

// --- validation + helpers ---

// validateProxyRulePayload runs the create-time ruleset against the supplied
// fields. Mirrors proxy.compileRule's invariants so a row that passes here
// will compile cleanly during Engine.SetRules — keeps the dashboard's
// validation feedback synchronous and accurate.
func validateProxyRulePayload(name, pattern string, methods []string, require, allow string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("name is required")
	}
	if strings.TrimSpace(pattern) == "" {
		return errors.New("pattern is required")
	}
	if !strings.HasPrefix(pattern, "/") {
		return errors.New("pattern must start with '/'")
	}
	require = strings.TrimSpace(require)
	allow = strings.TrimSpace(allow)
	if require == "" && allow == "" {
		return errors.New("either require or allow must be set")
	}
	if require != "" && allow != "" {
		return errors.New("set only one of require or allow")
	}
	if allow != "" && allow != "anonymous" {
		return errors.New(`allow only supports "anonymous"`)
	}
	if require != "" && !validRequireString(require) {
		return errors.New("require must be one of: anonymous, authenticated, agent, role:<name>, permission:<resource>:<action>, scope:<name>")
	}
	for _, m := range methods {
		if strings.TrimSpace(m) == "" {
			return errors.New("methods cannot contain empty entries")
		}
	}
	return nil
}

// validRequireString accepts the same require strings that
// proxy.parseRequirement accepts. Keeping a slim duplicate here lets us
// surface a 400 with a helpful message before the row is even persisted.
func validRequireString(require string) bool {
	switch {
	case require == "anonymous", require == "authenticated", require == "agent":
		return true
	case strings.HasPrefix(require, "role:") && len(require) > len("role:"):
		return true
	case strings.HasPrefix(require, "permission:") && len(require) > len("permission:"):
		return true
	case strings.HasPrefix(require, "scope:") && len(require) > len("scope:"):
		return true
	}
	return false
}

// normalizeMethods uppercases + trims each entry and filters empties so the
// engine sees a consistent set on every request. Returns nil when the input
// is empty so storage.ProxyRule.Methods round-trips as []string{} via
// marshalStringSlice's nil/empty handling.
func normalizeMethods(methods []string) []string {
	if len(methods) == 0 {
		return nil
	}
	out := make([]string, 0, len(methods))
	for _, m := range methods {
		m = strings.ToUpper(strings.TrimSpace(m))
		if m == "" {
			continue
		}
		out = append(out, m)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// newProxyRuleID mints the "pxr_<20-hex>" identifier; mirrors the convention
// used by newAuthFlowID + agent IDs.
func newProxyRuleID() string {
	buf := make([]byte, 10)
	if _, err := rand.Read(buf); err != nil {
		return "pxr_" + hex.EncodeToString([]byte(time.Now().UTC().Format("20060102150405")))
	}
	return "pxr_" + hex.EncodeToString(buf)
}
