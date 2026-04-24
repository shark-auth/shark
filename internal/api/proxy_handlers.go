package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sharkauth/sharkauth/internal/identity"
	"github.com/sharkauth/sharkauth/internal/proxy"
)

// proxyStatusResponse wraps a BreakerStats so the JSON is keyed under
// "data" — matching the conventions of the other v2 admin endpoints that
// the dashboard consumes.
type proxyStatusResponse struct {
	Data proxyStatusPayload `json:"data"`
}

// proxyStatusPayload flattens the internal BreakerStats into a
// dashboard-friendly shape. We rename a few fields (camelCase vs Go's
// natural capitalisation) and convert the latency to milliseconds so the
// UI doesn't have to know about time.Duration's nanosecond units.
type proxyStatusPayload struct {
	Enabled       bool   `json:"enabled"`
	State         string `json:"state"`
	CacheSize     int    `json:"cache_size"`
	NegCacheSize  int    `json:"neg_cache_size"`
	Failures      int    `json:"failures"`
	LastCheck     string `json:"last_check,omitempty"`
	LastLatencyMs int64  `json:"last_latency_ms"`
	LastStatus    int    `json:"last_status"`
	HealthURL     string `json:"health_url,omitempty"`
	Upstream      string `json:"upstream,omitempty"`
}

// proxyRuleView is the JSON-safe projection of a compiled proxy.Rule. We
// mirror the user-facing YAML shape (path + methods + require/allow +
// scopes) rather than echoing every internal Requirement field — the
// dashboard should show config-equivalent output so operators can copy it
// back into sharkauth.yaml.
type proxyRuleView struct {
	Path    string   `json:"path"`
	Methods []string `json:"methods"`
	Require string   `json:"require"`
	Scopes  []string `json:"scopes"`
}

// proxySimulateRequest is the POST body consumed by the simulator API. The
// "identity" field is optional — omitting it models an anonymous request,
// which is exactly how a dashboard operator tests whether a given public
// path is actually reachable without credentials.
type proxySimulateRequest struct {
	Method   string                 `json:"method"`
	Path     string                 `json:"path"`
	Identity proxySimulatedIdentity `json:"identity"`
}

// proxySimulatedIdentity mirrors proxy.Identity but in JSON-input shape
// (snake_case). Kept separate from proxy.Identity so the simulator API
// can evolve independently of the internal struct.
type proxySimulatedIdentity struct {
	UserID     string   `json:"user_id"`
	UserEmail  string   `json:"user_email"`
	Roles      []string `json:"roles"`
	AgentID    string   `json:"agent_id"`
	AgentName  string   `json:"agent_name"`
	AuthMethod string   `json:"auth_method"`
	Scopes     []string `json:"scopes"`
}

// proxySimulateResponse is the simulator's output: which rule matched
// (null when nothing did), the allow/deny bit, a human-readable reason,
// the exact headers that would be injected at the upstream, and the
// evaluation cost in microseconds so operators can spot pathological
// rules at a glance.
type proxySimulateResponse struct {
	MatchedRule     *proxyRuleView    `json:"matched_rule"`
	Decision        string            `json:"decision"`
	Reason          string            `json:"reason"`
	InjectedHeaders map[string]string `json:"injected_headers"`
	EvalUs          int64             `json:"eval_us"`
}

// handleProxyStatus returns a point-in-time snapshot of the proxy circuit
// breaker's state. 404 when the proxy is disabled so the dashboard can
// branch on HTTP status rather than parsing "enabled: false" semantics.
func (s *Server) handleProxyStatus(w http.ResponseWriter, r *http.Request) {
	if s.ProxyBreaker == nil {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, proxyStatusResponse{Data: s.buildProxyStatusPayload()})
}

// buildProxyStatusPayload extracts breaker stats into the JSON-friendly
// view. Extracted from handleProxyStatus so the SSE stream handler can
// reuse exactly the same shape without drift.
func (s *Server) buildProxyStatusPayload() proxyStatusPayload {
	stats := s.ProxyBreaker.Stats()
	payload := proxyStatusPayload{
		Enabled:       true,
		State:         stats.State,
		CacheSize:     stats.CacheSize,
		NegCacheSize:  stats.NegCacheSize,
		Failures:      stats.Failures,
		LastLatencyMs: stats.LastLatency.Milliseconds(),
		LastStatus:    stats.LastStatus,
		HealthURL:     stats.HealthURL,
	}
	if !stats.LastCheck.IsZero() {
		payload.LastCheck = stats.LastCheck.UTC().Format(time.RFC3339)
	}
	if s.Config != nil {
		payload.Upstream = s.Config.Proxy.Upstream
	}
	return payload
}

// handleProxyRules lists the compiled rules in their original user-facing
// shape so the dashboard displays config-equivalent output.
func (s *Server) handleProxyRules(w http.ResponseWriter, r *http.Request) {
	if s.ProxyEngine == nil {
		http.NotFound(w, r)
		return
	}
	rules := s.ProxyEngine.Rules()
	out := make([]proxyRuleView, 0, len(rules))
	for _, rule := range rules {
		out = append(out, proxyRuleToView(rule))
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": out})
}

// proxyRuleToView converts a compiled proxy.Rule into the wire view. The
// require string is reconstructed by combining the Requirement's Kind
// and Value so operators see "role:admin" rather than the internal enum.
func proxyRuleToView(rule *proxy.Rule) proxyRuleView {
	methods := make([]string, 0, len(rule.Methods))
	for m := range rule.Methods {
		methods = append(methods, m)
	}
	// Methods is a map; deterministic order helps tests & diffing.
	sortStrings(methods)

	return proxyRuleView{
		Path:    rule.Path,
		Methods: methods,
		Require: formatRequirement(rule.Require),
		Scopes:  append([]string{}, rule.Scopes...),
	}
}

// formatRequirement reconstructs the user-facing require string
// (e.g. "authenticated", "role:admin") from a compiled Requirement. This
// is the inverse of proxy.parseRequirement and lives here rather than in
// the proxy package so the proxy package can stay free of dashboard-
// specific JSON concerns.
func formatRequirement(req proxy.Requirement) string {
	switch req.Kind {
	case proxy.ReqAnonymous:
		return "anonymous"
	case proxy.ReqAuthenticated:
		return "authenticated"
	case proxy.ReqAgent:
		return "agent"
	case proxy.ReqRole:
		return "role:" + req.Value
	case proxy.ReqPermission:
		return "permission:" + req.Value
	case proxy.ReqScope:
		return "scope:" + req.Value
	default:
		return req.Kind.String()
	}
}

// sortStrings sorts in place; kept local to avoid importing sort just for
// the dashboard view.
func sortStrings(ss []string) {
	// Simple insertion sort — rule method lists are always <= 7 items in
	// practice (the standard HTTP verbs), so the allocation-free loop
	// beats bringing in sort.Strings.
	for i := 1; i < len(ss); i++ {
		j := i
		for j > 0 && ss[j-1] > ss[j] {
			ss[j-1], ss[j] = ss[j], ss[j-1]
			j--
		}
	}
}

// handleProxySimulate lets dashboard operators check what a given
// request+identity combo would do against the current rule set. Returns
// the matched rule (nullable), the decision + reason, the exact headers
// that would be injected, and evaluation time in microseconds.
func (s *Server) handleProxySimulate(w http.ResponseWriter, r *http.Request) {
	if s.ProxyEngine == nil {
		http.NotFound(w, r)
		return
	}

	var req proxySimulateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "invalid JSON body"))
		return
	}
	if req.Method == "" {
		req.Method = http.MethodGet
	}
	if req.Path == "" {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "path is required"))
		return
	}
	if !strings.HasPrefix(req.Path, "/") {
		req.Path = "/" + req.Path
	}

	// Construct a minimal synthetic http.Request for the engine. We don't
	// wire headers or body — the engine only looks at method + URL.Path,
	// so filling those is sufficient and keeps the simulator decoupled
	// from the real ServeHTTP plumbing.
	syntheticURL := &url.URL{Path: req.Path}
	synReq := &http.Request{
		Method: strings.ToUpper(req.Method),
		URL:    syntheticURL,
	}

	id := proxy.Identity{
		UserID:     req.Identity.UserID,
		UserEmail:  req.Identity.UserEmail,
		Roles:      req.Identity.Roles,
		AgentID:    req.Identity.AgentID,
		AgentName:  req.Identity.AgentName,
		AuthMethod: identity.AuthMethod(req.Identity.AuthMethod),
		Scopes:     req.Identity.Scopes,
	}

	start := time.Now()
	decision := s.ProxyEngine.Evaluate(synReq, id)
	elapsed := time.Since(start)

	resp := proxySimulateResponse{
		Decision:        decisionLabel(decision.Allow),
		Reason:          decision.Reason,
		InjectedHeaders: computeInjectedHeaders(id),
		EvalUs:          elapsed.Microseconds(),
	}
	if decision.MatchedRule != nil {
		view := proxyRuleToView(decision.MatchedRule)
		resp.MatchedRule = &view
	}

	writeJSON(w, http.StatusOK, resp)
}

// decisionLabel translates the engine's allow bit into the wire-level
// string. Keeping it a tiny helper makes the simulator output stable if
// we ever add a third outcome (e.g. "degraded").
func decisionLabel(allow bool) string {
	if allow {
		return "allow"
	}
	return "deny"
}

// computeInjectedHeaders renders the exact header set InjectIdentity would
// write at the upstream. Implemented by calling proxy.InjectIdentity into a
// disposable http.Header so the simulator can never drift from the real
// injection logic.
func computeInjectedHeaders(id proxy.Identity) map[string]string {
	h := http.Header{}
	proxy.InjectIdentity(h, id)
	out := make(map[string]string, len(h))
	for k, v := range h {
		if len(v) == 0 {
			continue
		}
		out[k] = v[0]
	}
	return out
}

// handleProxyStatusStream is the Server-Sent Events endpoint the dashboard
// subscribes to for live circuit-breaker status. Emits one event
// immediately so the UI paints promptly, then one every 2s until the
// client disconnects.
func (s *Server) handleProxyStatusStream(w http.ResponseWriter, r *http.Request) {
	if s.ProxyBreaker == nil {
		http.NotFound(w, r)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		WriteError(w, http.StatusInternalServerError,
			NewError(CodeInternal, "streaming unsupported").WithDocsURL(CodeInternal))
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // defeat Nginx buffering if proxied

	writeProxyStatusEvent(w, flusher, s.buildProxyStatusPayload())

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			writeProxyStatusEvent(w, flusher, s.buildProxyStatusPayload())
		}
	}
}

// writeProxyStatusEvent marshals payload as a single SSE "data:" line and
// flushes. Errors on the write path are silent: the only meaningful
// recovery is to terminate the stream, which the next iteration's ctx
// check will do when the client times out anyway.
func writeProxyStatusEvent(w http.ResponseWriter, flusher http.Flusher, payload proxyStatusPayload) {
	b, err := json.Marshal(payload)
	if err != nil {
		return
	}
	// SSE framing: "data: <json>\n\n". No event name (clients subscribe
	// with EventSource.onmessage by default).
	fmt.Fprintf(w, "data: %s\n\n", b) //#nosec G104 -- SSE writes fail only on client-gone
	flusher.Flush()
}
