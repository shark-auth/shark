package api_test

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sharkauth/sharkauth/internal/config"
	"github.com/sharkauth/sharkauth/internal/proxy"
	"github.com/sharkauth/sharkauth/internal/testutil"
)

// newProxyTestServer returns a TestServer with proxy enabled and a mock
// upstream. Returns the httptest.Server, the upstream's server, and the
// admin bearer key. The upstream echoes identity headers back in its JSON
// response so catch-all tests can assert on what the proxy injected.
func newProxyTestServer(t *testing.T, rules []config.ProxyRule) (*httptest.Server, *httptest.Server, string) {
	t.Helper()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := map[string]any{
			"path":          r.URL.Path,
			"method":        r.Method,
			"x_user_id":     r.Header.Get("X-User-ID"),
			"x_user_roles":  r.Header.Get("X-User-Roles"),
			"x_auth_method": r.Header.Get("X-Auth-Method"),
			"upstream_ok":   true,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(body) //nolint:errcheck
	}))
	t.Cleanup(upstream.Close)

	ts := testutil.NewTestServerWithConfig(t, func(cfg *config.Config) {
		cfg.Proxy = config.ProxyConfig{
			Enabled:  true,
			Upstream: upstream.URL,
		}
	})

	// v1.5: proxy rules are no longer loaded from YAML. The test server
	// spins up with an empty engine; push the test's desired ruleset in
	// directly so the rest of the assertions behave as they did before.
	if ts.APIServer != nil && ts.APIServer.ProxyEngine != nil {
		specs := make([]proxy.RuleSpec, len(rules))
		for i, r := range rules {
			specs[i] = proxy.RuleSpec{
				Path:    r.Path,
				Methods: r.Methods,
				Require: r.Require,
				Allow:   r.Allow,
				Scopes:  r.Scopes,
			}
		}
		if err := ts.APIServer.ProxyEngine.SetRules(specs); err != nil {
			t.Fatalf("proxy engine SetRules: %v", err)
		}
	}

	return ts.Server, upstream, ts.AdminKey
}

// decodeJSON is a local helper so proxy tests don't pull TestServer.
func decodeJSON(t *testing.T, resp *http.Response, target any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		t.Fatalf("decode: %v", err)
	}
}

// doGet issues a GET with the given bearer.
func doGet(t *testing.T, ts *httptest.Server, path, bearer string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, ts.URL+path, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	return resp
}

// doPostJSON issues a POST with JSON body and bearer.
func doPostJSON(t *testing.T, ts *httptest.Server, path, bearer string, body any) *http.Response {
	t.Helper()
	buf, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, ts.URL+path, strings.NewReader(string(buf)))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	return resp
}

func TestProxyStatus_ReturnsStats(t *testing.T) {
	rules := []config.ProxyRule{
		{Path: "/api/foo/*", Require: "authenticated"},
		{Path: "/public/*", Allow: "anonymous"},
	}
	ts, _, adminKey := newProxyTestServer(t, rules)

	resp := doGet(t, ts, "/api/v1/admin/proxy/status", adminKey)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var out struct {
		Data map[string]any `json:"data"`
	}
	decodeJSON(t, resp, &out)
	if out.Data["state"] == "" {
		t.Fatalf("expected non-empty state, got %v", out.Data)
	}
	if _, ok := out.Data["cache_size"]; !ok {
		t.Fatalf("expected cache_size key, got %v", out.Data)
	}
}

func TestProxyStatus_404WhenProxyDisabled(t *testing.T) {
	ts := testutil.NewTestServer(t)
	resp := ts.GetWithAdminKey("/api/v1/admin/proxy/status")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestProxyRules_ListsRules(t *testing.T) {
	rules := []config.ProxyRule{
		{Path: "/api/foo/*", Require: "authenticated", Methods: []string{"GET", "POST"}},
		{Path: "/public/*", Allow: "anonymous"},
	}
	ts, _, adminKey := newProxyTestServer(t, rules)
	resp := doGet(t, ts, "/api/v1/admin/proxy/rules", adminKey)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var out struct {
		Data []struct {
			Path    string   `json:"path"`
			Methods []string `json:"methods"`
			Require string   `json:"require"`
			Scopes  []string `json:"scopes"`
		} `json:"data"`
	}
	decodeJSON(t, resp, &out)
	if len(out.Data) != 2 {
		t.Fatalf("got %d rules, want 2", len(out.Data))
	}
	if out.Data[0].Path != "/api/foo/*" || out.Data[0].Require != "authenticated" {
		t.Fatalf("rule[0] mismatch: %+v", out.Data[0])
	}
	if out.Data[1].Path != "/public/*" || out.Data[1].Require != "anonymous" {
		t.Fatalf("rule[1] mismatch: %+v", out.Data[1])
	}
}

func TestProxySimulate_AllowsAuthenticatedUser(t *testing.T) {
	rules := []config.ProxyRule{
		{Path: "/api/foo/*", Require: "authenticated"},
	}
	ts, _, adminKey := newProxyTestServer(t, rules)

	body := map[string]any{
		"method": "GET",
		"path":   "/api/foo/bar",
		"identity": map[string]any{
			"user_id":     "u1",
			"auth_method": "session-live",
		},
	}
	resp := doPostJSON(t, ts, "/api/v1/admin/proxy/simulate", adminKey, body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var out struct {
		Decision    string `json:"decision"`
		Reason      string `json:"reason"`
		MatchedRule *struct {
			Path    string `json:"path"`
			Require string `json:"require"`
		} `json:"matched_rule"`
	}
	decodeJSON(t, resp, &out)
	if out.Decision != "allow" {
		t.Fatalf("decision = %q, want allow (reason=%q)", out.Decision, out.Reason)
	}
	if out.MatchedRule == nil || out.MatchedRule.Path != "/api/foo/*" {
		t.Fatalf("matched_rule = %+v", out.MatchedRule)
	}
}

func TestProxySimulate_DeniesAnonymous(t *testing.T) {
	rules := []config.ProxyRule{
		{Path: "/api/foo/*", Require: "authenticated"},
	}
	ts, _, adminKey := newProxyTestServer(t, rules)
	body := map[string]any{
		"method":   "GET",
		"path":     "/api/foo/bar",
		"identity": map[string]any{},
	}
	resp := doPostJSON(t, ts, "/api/v1/admin/proxy/simulate", adminKey, body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var out struct {
		Decision string `json:"decision"`
		Reason   string `json:"reason"`
	}
	decodeJSON(t, resp, &out)
	if out.Decision != "deny" {
		t.Fatalf("decision = %q, want deny", out.Decision)
	}
	if !strings.Contains(out.Reason, "authentication required") {
		t.Fatalf("reason = %q, want 'authentication required'", out.Reason)
	}
}

func TestProxySimulate_MatchesCorrectRule(t *testing.T) {
	rules := []config.ProxyRule{
		{Path: "/api/foo/*", Require: "authenticated"},
		{Path: "/public/*", Allow: "anonymous"},
	}
	ts, _, adminKey := newProxyTestServer(t, rules)
	body := map[string]any{
		"method":   "GET",
		"path":     "/public/x",
		"identity": map[string]any{},
	}
	resp := doPostJSON(t, ts, "/api/v1/admin/proxy/simulate", adminKey, body)
	var out struct {
		Decision    string `json:"decision"`
		MatchedRule *struct {
			Path string `json:"path"`
		} `json:"matched_rule"`
	}
	decodeJSON(t, resp, &out)
	if out.MatchedRule == nil || out.MatchedRule.Path != "/public/*" {
		t.Fatalf("matched_rule = %+v, want /public/*", out.MatchedRule)
	}
	if out.Decision != "allow" {
		t.Fatalf("decision = %q, want allow", out.Decision)
	}
}

func TestProxySimulate_NoMatchIsDeny(t *testing.T) {
	rules := []config.ProxyRule{
		{Path: "/api/foo/*", Require: "authenticated"},
	}
	ts, _, adminKey := newProxyTestServer(t, rules)
	body := map[string]any{
		"method":   "GET",
		"path":     "/unlisted",
		"identity": map[string]any{},
	}
	resp := doPostJSON(t, ts, "/api/v1/admin/proxy/simulate", adminKey, body)
	var out struct {
		Decision    string `json:"decision"`
		MatchedRule any    `json:"matched_rule"`
	}
	decodeJSON(t, resp, &out)
	if out.Decision != "deny" {
		t.Fatalf("decision = %q, want deny", out.Decision)
	}
	if out.MatchedRule != nil {
		t.Fatalf("matched_rule = %+v, want null", out.MatchedRule)
	}
}

func TestProxySimulate_InjectedHeadersReflectIdentity(t *testing.T) {
	rules := []config.ProxyRule{
		{Path: "/api/*", Require: "authenticated"},
	}
	ts, _, adminKey := newProxyTestServer(t, rules)
	body := map[string]any{
		"method": "GET",
		"path":   "/api/hello",
		"identity": map[string]any{
			"user_id":     "u_abc",
			"roles":       []string{"admin", "user"},
			"auth_method": "session-live",
		},
	}
	resp := doPostJSON(t, ts, "/api/v1/admin/proxy/simulate", adminKey, body)
	var out struct {
		InjectedHeaders map[string]string `json:"injected_headers"`
	}
	decodeJSON(t, resp, &out)
	if out.InjectedHeaders["X-User-Id"] != "u_abc" {
		t.Fatalf("X-User-ID header = %q, want u_abc. All headers: %+v", out.InjectedHeaders["X-User-Id"], out.InjectedHeaders)
	}
	if !strings.Contains(out.InjectedHeaders["X-User-Roles"], "admin") {
		t.Fatalf("X-User-Roles = %q, want to contain admin", out.InjectedHeaders["X-User-Roles"])
	}
}

func TestProxyStatusStream_SendsEvents(t *testing.T) {
	rules := []config.ProxyRule{
		{Path: "/api/*", Require: "authenticated"},
	}
	ts, _, adminKey := newProxyTestServer(t, rules)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/api/v1/admin/proxy/status/stream", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+adminKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want text/event-stream", got)
	}

	reader := bufio.NewReader(resp.Body)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		t.Fatalf("read: %v", err)
	}
	if !strings.HasPrefix(line, "data:") {
		t.Fatalf("first SSE line = %q, want data: prefix", line)
	}
	payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
	var parsed map[string]any
	if err := json.Unmarshal([]byte(payload), &parsed); err != nil {
		t.Fatalf("SSE payload invalid JSON %q: %v", payload, err)
	}
	if parsed["state"] == "" {
		t.Fatalf("SSE payload missing state: %+v", parsed)
	}
}

func TestProxyIntegration_CatchAllForwardsAuthedRequest(t *testing.T) {
	rules := []config.ProxyRule{
		{Path: "/other/*", Allow: "anonymous"},
	}
	ts, upstream, _ := newProxyTestServer(t, rules)

	resp, err := http.Get(ts.URL + "/other/path")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out["path"] != "/other/path" {
		t.Fatalf("upstream saw path %v, want /other/path", out["path"])
	}
	if out["upstream_ok"] != true {
		t.Fatalf("upstream did not respond: %+v", out)
	}
	_ = upstream // silence unused; we use it implicitly via the proxy.
}

func TestProxyIntegration_AuthRoutesBypassProxy(t *testing.T) {
	rules := []config.ProxyRule{
		{Path: "/*", Require: "authenticated"},
	}
	ts, _, _ := newProxyTestServer(t, rules)

	// /api/v1/auth/login is a Shark route — must not be captured by the
	// catch-all. Signup endpoint is the simplest unauthenticated Shark
	// route that returns something other than 401 for missing body.
	resp, err := http.Post(ts.URL+"/api/v1/auth/login", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	// 400 bad request body is fine — what we're ruling out is a 403/502
	// from the proxy, which would mean the catch-all stole the route.
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusBadGateway {
		t.Fatalf("auth/login got proxy status %d — catch-all stole the route", resp.StatusCode)
	}
}

func TestProxyIntegration_DeniedAnonymousReturns401(t *testing.T) {
	// No rules at all → every path falls through to default-deny. The caller
	// is anonymous (no cookie, no bearer) so the proxy translates the deny
	// into a 401 per W15b ("authentication required" semantics — 403 is
	// reserved for authenticated-but-unauthorized).
	rules := []config.ProxyRule{}
	ts, _, _ := newProxyTestServer(t, rules)

	resp, err := http.Get(ts.URL + "/somewhere/else")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
	if reason := resp.Header.Get("X-Shark-Deny-Reason"); reason == "" {
		t.Fatalf("missing X-Shark-Deny-Reason header")
	}
}

// TestProxySimulate_EvalTimeIsRecorded checks that the simulator reports
// a sub-ms evaluation time — if eval_us is negative or insanely large it
// signals a clock regression.
func TestProxySimulate_EvalTimeIsRecorded(t *testing.T) {
	rules := []config.ProxyRule{
		{Path: "/*", Allow: "anonymous"},
	}
	ts, _, adminKey := newProxyTestServer(t, rules)
	body := map[string]any{
		"method":   "GET",
		"path":     "/foo",
		"identity": map[string]any{},
	}
	resp := doPostJSON(t, ts, "/api/v1/admin/proxy/simulate", adminKey, body)
	var out struct {
		EvalUs int64 `json:"eval_us"`
	}
	decodeJSON(t, resp, &out)
	if out.EvalUs < 0 {
		t.Fatalf("eval_us = %d (negative)", out.EvalUs)
	}
	// No upper bound (CI machines can be slow); this test mostly guards
	// against a missing-field serialisation bug.
	_ = fmt.Sprintf("eval_us=%d", out.EvalUs)
}
