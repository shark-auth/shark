package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// mustEngine compiles specs and fails the test on error. Used in the
// happy paths where spec validity is not the thing under test.
func mustEngine(t *testing.T, specs ...RuleSpec) *Engine {
	t.Helper()
	e, err := NewEngine(specs)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	return e
}

// newGetReq is a one-line helper that builds an in-memory GET request
// for a path. Method-specific tests build their own Requests.
func newGetReq(path string) *http.Request {
	return httptest.NewRequest(http.MethodGet, path, nil)
}

// -----------------------------------------------------------------------------
// Path matching
// -----------------------------------------------------------------------------

func TestPath_Exact(t *testing.T) {
	p, err := compilePath("/api/foo")
	if err != nil {
		t.Fatalf("compilePath: %v", err)
	}
	cases := map[string]bool{
		"/api/foo":     true,
		"/api/bar":     false,
		"/api/foo/bar": false,
		"/api":         false,
		"/":            false,
	}
	for urlPath, want := range cases {
		if got := p.match(urlPath); got != want {
			t.Errorf("match(%q) = %v, want %v", urlPath, got, want)
		}
	}
}

func TestPath_TrailingWildcard(t *testing.T) {
	p, err := compilePath("/api/*")
	if err != nil {
		t.Fatalf("compilePath: %v", err)
	}
	cases := map[string]bool{
		"/api":             true,
		"/api/foo":         true,
		"/api/foo/bar":     true,
		"/api/foo/bar/baz": true,
		"/other":           false,
		"/":                false,
	}
	for urlPath, want := range cases {
		if got := p.match(urlPath); got != want {
			t.Errorf("match(%q) = %v, want %v", urlPath, got, want)
		}
	}
}

func TestPath_SingleSegmentWildcard(t *testing.T) {
	p, err := compilePath("/api/*/deep")
	if err != nil {
		t.Fatalf("compilePath: %v", err)
	}
	cases := map[string]bool{
		"/api/users/deep":       true,
		"/api/things/deep":      true,
		"/api/users/inner/deep": false,
		"/api/deep":             false,
		"/api/users/deep/extra": false,
	}
	for urlPath, want := range cases {
		if got := p.match(urlPath); got != want {
			t.Errorf("match(%q) = %v, want %v", urlPath, got, want)
		}
	}
}

func TestPath_ParamPlaceholder(t *testing.T) {
	paramPat, err := compilePath("/api/{id}")
	if err != nil {
		t.Fatalf("compilePath: %v", err)
	}
	wildcardPat, err := compilePath("/api/*")
	if err != nil {
		t.Fatalf("compilePath: %v", err)
	}
	// {id} is a SINGLE-segment wildcard, unlike trailing /* which is
	// prefix. Confirm the semantic difference.
	if !paramPat.match("/api/123") {
		t.Error("param should match /api/123")
	}
	if paramPat.match("/api/123/extra") {
		t.Error("param should NOT match /api/123/extra (single segment only)")
	}
	if !wildcardPat.match("/api/123/extra") {
		t.Error("trailing /* should match /api/123/extra")
	}
}

func TestPath_NormalizesLeadingSlash(t *testing.T) {
	if _, err := compilePath("api/foo"); err == nil {
		t.Fatal("expected error for path without leading slash")
	}
	if _, err := compilePath(""); err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestPath_CaseSensitive(t *testing.T) {
	p, err := compilePath("/api/*")
	if err != nil {
		t.Fatalf("compilePath: %v", err)
	}
	if p.match("/API/foo") {
		t.Error("/api/* should NOT match /API/foo (HTTP paths are case-sensitive)")
	}
}

// -----------------------------------------------------------------------------
// Engine semantics
// -----------------------------------------------------------------------------

func TestEngine_FirstMatchWins(t *testing.T) {
	// Two overlapping rules: the first forbids, the second would allow.
	// First should win — deny.
	e := mustEngine(t,
		RuleSpec{Path: "/api/admin/*", Require: "role:admin"},
		RuleSpec{Path: "/api/*", Allow: "anonymous"},
	)
	anon := Identity{AuthMethod: "anonymous"}
	d := e.Evaluate(newGetReq("/api/admin/users"), anon)
	if d.Allow {
		t.Fatalf("expected deny, got allow (matched=%v)", d.MatchedRule)
	}
	if d.MatchedRule == nil || d.MatchedRule.Path != "/api/admin/*" {
		t.Errorf("expected first rule to match, got %v", d.MatchedRule)
	}
}

func TestEngine_NoMatchDefaultDeny(t *testing.T) {
	e := mustEngine(t,
		RuleSpec{Path: "/api/*", Allow: "anonymous"},
	)
	d := e.Evaluate(newGetReq("/other/path"), Identity{})
	if d.Allow {
		t.Fatal("expected default deny")
	}
	if d.MatchedRule != nil {
		t.Error("expected no matched rule for no-match case")
	}
	if d.Reason != "no rule matched" {
		t.Errorf("reason = %q, want \"no rule matched\"", d.Reason)
	}
}

func TestEngine_NoRulesEveryRequestDenied(t *testing.T) {
	e := mustEngine(t)
	if d := e.Evaluate(newGetReq("/anything"), Identity{UserID: "u1"}); d.Allow {
		t.Error("empty rule list must deny by default")
	}
}

func TestEngine_AnonymousAllows(t *testing.T) {
	e := mustEngine(t, RuleSpec{Path: "/public/*", Allow: "anonymous"})
	d := e.Evaluate(newGetReq("/public/ping"), Identity{})
	if !d.Allow {
		t.Errorf("anonymous-allowed path should allow anonymous: %q", d.Reason)
	}
}

func TestEngine_AuthenticatedRequiresAuth(t *testing.T) {
	e := mustEngine(t, RuleSpec{Path: "/api/*", Require: "authenticated"})
	// Anonymous
	d := e.Evaluate(newGetReq("/api/x"), Identity{})
	if d.Allow {
		t.Error("anonymous should be denied when auth required")
	}
	if !strings.Contains(d.Reason, "authentication") {
		t.Errorf("reason = %q, want mention of authentication", d.Reason)
	}
	// User
	d = e.Evaluate(newGetReq("/api/x"), Identity{UserID: "u1"})
	if !d.Allow {
		t.Error("authenticated user should be allowed")
	}
	// Agent also counts as authenticated
	d = e.Evaluate(newGetReq("/api/x"), Identity{AgentID: "a1"})
	if !d.Allow {
		t.Error("agent should count as authenticated")
	}
}

func TestEngine_RoleRequired(t *testing.T) {
	e := mustEngine(t, RuleSpec{Path: "/admin/*", Require: "role:admin"})

	d := e.Evaluate(newGetReq("/admin/dash"), Identity{UserID: "u1", Roles:     []string{"admin"}})
	if !d.Allow {
		t.Errorf("admin should be allowed: %q", d.Reason)
	}

	d = e.Evaluate(newGetReq("/admin/dash"), Identity{UserID: "u1", Roles:     []string{"user"}})
	if d.Allow {
		t.Error("non-admin should be denied")
	}
	if !strings.Contains(d.Reason, "admin") {
		t.Errorf("reason = %q, want mention of admin", d.Reason)
	}

	// Anonymous user has no roles at all.
	d = e.Evaluate(newGetReq("/admin/dash"), Identity{})
	if d.Allow {
		t.Error("anonymous should be denied when role required")
	}
}

func TestEngine_AgentRequired(t *testing.T) {
	e := mustEngine(t, RuleSpec{Path: "/webhooks/*", Require: "agent"})

	if d := e.Evaluate(newGetReq("/webhooks/in"), Identity{AgentID: "a1"}); !d.Allow {
		t.Errorf("agent should be allowed: %q", d.Reason)
	}
	if d := e.Evaluate(newGetReq("/webhooks/in"), Identity{UserID: "u1"}); d.Allow {
		t.Error("user (non-agent) should be denied by agent rule")
	}
}

func TestEngine_ScopeRequired(t *testing.T) {
	e := mustEngine(t, RuleSpec{Path: "/writes/*", Require: "scope:webhooks:write"})

	ok := Identity{AgentID: "a1", Scopes: []string{"webhooks:write"}}
	if d := e.Evaluate(newGetReq("/writes/x"), ok); !d.Allow {
		t.Errorf("scope present should allow: %q", d.Reason)
	}

	noScope := Identity{AgentID: "a1", Scopes: []string{"other:read"}}
	if d := e.Evaluate(newGetReq("/writes/x"), noScope); d.Allow {
		t.Error("missing scope should deny")
	}
}

func TestEngine_ExtraScopes_AndSemantics(t *testing.T) {
	e := mustEngine(t, RuleSpec{
		Path:    "/ops/*",
		Require: "agent",
		Scopes:  []string{"a", "b"},
	})

	// Has a but not b → deny, reason mentions b.
	partial := Identity{AgentID: "a1", Scopes: []string{"a"}}
	d := e.Evaluate(newGetReq("/ops/x"), partial)
	if d.Allow {
		t.Error("partial scope satisfaction must not allow (AND semantics)")
	}
	if !strings.Contains(d.Reason, "b") {
		t.Errorf("reason = %q, want mention of missing scope \"b\"", d.Reason)
	}

	// Has both → allow.
	full := Identity{AgentID: "a1", Scopes: []string{"a", "b"}}
	if d := e.Evaluate(newGetReq("/ops/x"), full); !d.Allow {
		t.Errorf("all scopes present should allow: %q", d.Reason)
	}

	// Primary requirement unmet (not an agent) → extra scopes never
	// inspected; reason reflects the primary failure.
	notAgent := Identity{UserID: "u1", Scopes: []string{"a", "b"}}
	d = e.Evaluate(newGetReq("/ops/x"), notAgent)
	if d.Allow {
		t.Error("primary (agent) failure must deny regardless of extra scopes")
	}
	if !strings.Contains(d.Reason, "agent") {
		t.Errorf("reason should describe primary failure, got %q", d.Reason)
	}
}

func TestEngine_MethodFilter(t *testing.T) {
	// First rule methods=[GET,POST] requires role:admin; fallback /api/*
	// allows anonymous. A PUT request should fall through to the second
	// rule (treated as no-match on the first), not be denied outright by
	// the first.
	e := mustEngine(t,
		RuleSpec{Path: "/api/admin/*", Methods: []string{"GET", "POST"}, Require: "role:admin"},
		RuleSpec{Path: "/api/*", Allow: "anonymous"},
	)

	get := httptest.NewRequest(http.MethodGet, "/api/admin/x", nil)
	d := e.Evaluate(get, Identity{})
	if d.Allow || d.MatchedRule == nil || d.MatchedRule.Path != "/api/admin/*" {
		t.Errorf("GET should hit first rule and deny (anon has no admin role), got allow=%v matched=%v", d.Allow, d.MatchedRule)
	}

	put := httptest.NewRequest(http.MethodPut, "/api/admin/x", nil)
	d = e.Evaluate(put, Identity{})
	if !d.Allow {
		t.Errorf("PUT should fall through to allow rule: allow=%v reason=%q", d.Allow, d.Reason)
	}
	if d.MatchedRule == nil || d.MatchedRule.Path != "/api/*" {
		t.Errorf("PUT should match fallback rule, got %v", d.MatchedRule)
	}

	// Method comparison is case-insensitive for YAML-entered lowercase
	// methods. Sanity check.
	eLower := mustEngine(t, RuleSpec{Path: "/x", Methods: []string{"get"}, Allow: "anonymous"})
	if d := eLower.Evaluate(newGetReq("/x"), Identity{}); !d.Allow {
		t.Error("methods should be case-insensitive")
	}
}

func TestEngine_PermissionTODO(t *testing.T) {
	e := mustEngine(t, RuleSpec{Path: "/rbac/*", Require: "permission:users:read"})
	d := e.Evaluate(newGetReq("/rbac/list"), Identity{UserID: "u1"})
	if d.Allow {
		t.Error("permission rules must deny in MVP")
	}
	if !strings.Contains(strings.ToLower(d.Reason), "not yet implemented") {
		t.Errorf("reason should mention TODO, got %q", d.Reason)
	}
}

// -----------------------------------------------------------------------------
// Compile / parse
// -----------------------------------------------------------------------------

func TestCompile_BadPath(t *testing.T) {
	_, err := NewEngine([]RuleSpec{{Path: "no-slash", Allow: "anonymous"}})
	if err == nil {
		t.Fatal("expected error for missing leading slash")
	}
	if !strings.Contains(err.Error(), "no-slash") {
		t.Errorf("error should mention offending path, got %v", err)
	}
}

func TestCompile_BothRequireAndAllow(t *testing.T) {
	_, err := NewEngine([]RuleSpec{{Path: "/x", Require: "authenticated", Allow: "anonymous"}})
	if err == nil {
		t.Fatal("expected error when both require and allow are set")
	}
}

func TestCompile_UnknownRequirement(t *testing.T) {
	_, err := NewEngine([]RuleSpec{{Path: "/x", Require: "weird:thing"}})
	if err == nil {
		t.Fatal("expected error for unknown requirement kind")
	}
}

func TestCompile_EmptyRequireEmptyAllow(t *testing.T) {
	_, err := NewEngine([]RuleSpec{{Path: "/x"}})
	if err == nil {
		t.Fatal("expected error when neither require nor allow is set")
	}
}

func TestCompile_EmptyRoleValue(t *testing.T) {
	_, err := NewEngine([]RuleSpec{{Path: "/x", Require: "role:"}})
	if err == nil {
		t.Fatal("expected error for empty role value")
	}
}

func TestCompile_BadAllow(t *testing.T) {
	_, err := NewEngine([]RuleSpec{{Path: "/x", Allow: "everyone"}})
	if err == nil {
		t.Fatal("expected error for allow other than \"anonymous\"")
	}
}

func TestCompile_EmptyMethod(t *testing.T) {
	_, err := NewEngine([]RuleSpec{{Path: "/x", Methods: []string{""}, Allow: "anonymous"}})
	if err == nil {
		t.Fatal("expected error for empty method entry")
	}
}

func TestEngine_RulesAccessor(t *testing.T) {
	e := mustEngine(t,
		RuleSpec{Path: "/a", Allow: "anonymous"},
		RuleSpec{Path: "/b/*", Require: "authenticated"},
	)
	if got := len(e.Rules()); got != 2 {
		t.Errorf("Rules() len = %d, want 2", got)
	}
}

// -----------------------------------------------------------------------------
// Integration with ReverseProxy
// -----------------------------------------------------------------------------

func TestReverseProxy_DenyWritesDeniedResponse(t *testing.T) {
	var hits int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
	}))
	defer upstream.Close()

	// Empty rule list: default deny for every request.
	engine := mustEngine(t)
	p, err := New(Config{
		Enabled:       true,
		Upstream:      upstream.URL,
		Timeout:       time.Second,
		StripIncoming: true,
	}, engine, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, newGetReq("/anything"))

	// Default-deny for an anonymous caller with no matching rule is treated
	// as "authentication required" (W15b): unauthenticated callers get 401,
	// not 403.
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want 401", rec.Code)
	}
	if got := rec.Header().Get(HeaderDenyReason); got == "" {
		t.Error("X-Shark-Deny-Reason header should be set on deny")
	}
	body, _ := io.ReadAll(rec.Body)
	if !strings.Contains(string(body), "unauthorized") {
		t.Errorf("body should mention unauthorized, got %q", body)
	}
	if atomic.LoadInt32(&hits) != 0 {
		t.Error("upstream must not be contacted when denied")
	}
}

func TestReverseProxy_AllowPassesThrough(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer upstream.Close()

	engine := mustEngine(t, RuleSpec{Path: "/api/*", Allow: "anonymous"})
	p, err := New(Config{
		Enabled:       true,
		Upstream:      upstream.URL,
		Timeout:       time.Second,
		StripIncoming: true,
	}, engine, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, newGetReq("/api/ping"))

	if rec.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", rec.Code)
	}
	body, _ := io.ReadAll(rec.Body)
	if string(body) != "ok" {
		t.Errorf("body: got %q", body)
	}
}

func TestReverseProxy_EngineNilPassthrough(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	// No engine: legacy P1 behavior — every request forwarded.
	p, err := New(Config{
		Enabled:       true,
		Upstream:      upstream.URL,
		Timeout:       time.Second,
		StripIncoming: true,
	}, nil, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, newGetReq("/whatever"))
	if rec.Code != http.StatusOK {
		t.Errorf("nil engine should passthrough, got status %d", rec.Code)
	}
}

func TestReverseProxy_DenyReasonInHeader(t *testing.T) {
	// Engine rule requires role:admin; caller is an authenticated user who
	// lacks the admin role. 403 is the correct status (authenticated-but-
	// unauthorized) and the deny reason should surface in the header.
	var hits int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
	}))
	defer upstream.Close()

	engine := mustEngine(t, RuleSpec{Path: "/admin/*", Require: "role:admin"})
	p, err := New(Config{
		Enabled:       true,
		Upstream:      upstream.URL,
		Timeout:       time.Second,
		StripIncoming: true,
	}, engine, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	rec := httptest.NewRecorder()
	req := newGetReq("/admin/dash")
	req = req.WithContext(WithIdentity(req.Context(), Identity{UserID: "u1", Roles:     []string{"user"}}))
	p.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status: got %d, want 403", rec.Code)
	}
	if reason := rec.Header().Get(HeaderDenyReason); !strings.Contains(reason, "admin") {
		t.Errorf("deny reason header = %q, want mention of admin", reason)
	}
	if atomic.LoadInt32(&hits) != 0 {
		t.Error("upstream must not be called on deny")
	}
}

// -----------------------------------------------------------------------------
// atomic.Pointer swap — lock-free concurrency
// -----------------------------------------------------------------------------

// TestEngine_SetRulesAtomicSwap verifies the pointer-swap semantics: after
// SetRules returns, subsequent Evaluate calls observe the new rule set;
// a failed compile leaves the previous snapshot untouched.
func TestEngine_SetRulesAtomicSwap(t *testing.T) {
	e := mustEngine(t, RuleSpec{Path: "/a", Allow: "anonymous"})

	// Initial snapshot has exactly one rule.
	if got := len(e.Rules()); got != 1 {
		t.Fatalf("initial Rules() len = %d, want 1", got)
	}

	// Successful swap — new snapshot replaces the old one.
	if err := e.SetRules([]RuleSpec{
		{Path: "/a", Allow: "anonymous"},
		{Path: "/b", Allow: "anonymous"},
	}); err != nil {
		t.Fatalf("SetRules: %v", err)
	}
	if got := len(e.Rules()); got != 2 {
		t.Errorf("after swap Rules() len = %d, want 2", got)
	}

	// Failed compile — previous snapshot must remain in place.
	if err := e.SetRules([]RuleSpec{{Path: "no-slash", Allow: "anonymous"}}); err == nil {
		t.Fatal("expected compile error for missing leading slash")
	}
	if got := len(e.Rules()); got != 2 {
		t.Errorf("after failed swap Rules() len = %d, want 2 (last-good retained)", got)
	}
}

// -----------------------------------------------------------------------------
// Tier + GlobalRole + Decision kinds (Lane A A3)
// -----------------------------------------------------------------------------

// TestEngine_TierPredicates is the table-driven home for tier:X rule
// behavior. Covers exact match, mismatch → PaywallRedirect,
// anonymous → DenyAnonymous (paywall suppressed), and ensures
// RequiredTier is populated on the deny decision.
func TestEngine_TierPredicates(t *testing.T) {
	type tc struct {
		name         string
		require      string
		id           Identity
		wantAllow    bool
		wantKind     DecisionKind
		wantReqTier  string
		wantInReason string
	}
	cases := []tc{
		{
			name:      "tier match allows",
			require:   "tier:pro",
			id:        Identity{UserID: "u1", Tier: "pro"},
			wantAllow: true,
			wantKind:  DecisionAllow,
		},
		{
			name:         "tier mismatch → PaywallRedirect",
			require:      "tier:pro",
			id:           Identity{UserID: "u1", Tier: "free"},
			wantKind:     DecisionPaywallRedirect,
			wantReqTier:  "pro",
			wantInReason: "tier",
		},
		{
			name:         "tier mismatch empty tier → PaywallRedirect",
			require:      "tier:pro",
			id:           Identity{UserID: "u1"},
			wantKind:     DecisionPaywallRedirect,
			wantReqTier:  "pro",
			wantInReason: "tier",
		},
		{
			name:         "anonymous tier → DenyAnonymous (paywall suppressed)",
			require:      "tier:pro",
			id:           Identity{},
			wantKind:     DecisionDenyAnonymous,
			wantReqTier:  "pro",
			wantInReason: "tier",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := mustEngine(t, RuleSpec{Path: "/premium/*", Require: c.require})
			d := e.Evaluate(newGetReq("/premium/x"), c.id)
			if d.Allow != c.wantAllow {
				t.Errorf("Allow = %v, want %v (reason=%q)", d.Allow, c.wantAllow, d.Reason)
			}
			if d.Kind != c.wantKind {
				t.Errorf("Kind = %d, want %d", d.Kind, c.wantKind)
			}
			if !c.wantAllow && d.RequiredTier != c.wantReqTier {
				t.Errorf("RequiredTier = %q, want %q", d.RequiredTier, c.wantReqTier)
			}
			if c.wantInReason != "" && !strings.Contains(d.Reason, c.wantInReason) {
				t.Errorf("Reason %q does not contain %q", d.Reason, c.wantInReason)
			}
		})
	}
}

// TestEngine_GlobalRolePredicate covers global_role:X — distinct kind
// from role:X but identical membership check against Identity.Roles.
// Also verifies the back-compat alias: role:X still works.
func TestEngine_GlobalRolePredicate(t *testing.T) {
	e := mustEngine(t, RuleSpec{Path: "/ops/*", Require: "global_role:admin"})

	if d := e.Evaluate(newGetReq("/ops/x"), Identity{UserID: "u1", Roles: []string{"admin"}}); !d.Allow {
		t.Errorf("global_role match must allow, got reason=%q", d.Reason)
	} else if d.Kind != DecisionAllow {
		t.Errorf("Kind = %d, want DecisionAllow", d.Kind)
	}

	d := e.Evaluate(newGetReq("/ops/x"), Identity{UserID: "u1", Roles: []string{"user"}})
	if d.Allow {
		t.Error("global_role miss must deny")
	}
	if d.Kind != DecisionDenyForbidden {
		t.Errorf("authenticated-miss Kind = %d, want DecisionDenyForbidden", d.Kind)
	}
	if !strings.Contains(d.Reason, "global_role") {
		t.Errorf("reason should mention global_role, got %q", d.Reason)
	}

	// Anonymous caller on role/global_role rule → DenyAnonymous, not
	// Forbidden. The proxy uses this to pick 401 vs 403.
	anonD := e.Evaluate(newGetReq("/ops/x"), Identity{})
	if anonD.Kind != DecisionDenyAnonymous {
		t.Errorf("anonymous Kind = %d, want DecisionDenyAnonymous", anonD.Kind)
	}

	// role: alias still resolves to the same underlying check.
	aliasE := mustEngine(t, RuleSpec{Path: "/ops/*", Require: "role:admin"})
	if d := aliasE.Evaluate(newGetReq("/ops/x"), Identity{UserID: "u1", Roles: []string{"admin"}}); !d.Allow {
		t.Errorf("role: alias must still allow on match: %q", d.Reason)
	}
}

// TestEngine_DecisionKind_NoMatch covers the default-deny path's Kind:
// anonymous → DenyAnonymous, authenticated → DenyForbidden.
func TestEngine_DecisionKind_NoMatch(t *testing.T) {
	e := mustEngine(t, RuleSpec{Path: "/matches/nothing", Allow: "anonymous"})

	anon := e.Evaluate(newGetReq("/other"), Identity{})
	if anon.Kind != DecisionDenyAnonymous {
		t.Errorf("anon no-match Kind = %d, want DenyAnonymous", anon.Kind)
	}
	auth := e.Evaluate(newGetReq("/other"), Identity{UserID: "u1"})
	if auth.Kind != DecisionDenyForbidden {
		t.Errorf("auth no-match Kind = %d, want DenyForbidden", auth.Kind)
	}
}

// TestParseRequirement_NewKinds pins the parser's handling of tier:X and
// global_role:X including empty-value errors.
func TestParseRequirement_NewKinds(t *testing.T) {
	r, err := parseRequirement("tier:pro", "")
	if err != nil {
		t.Fatalf("tier:pro: %v", err)
	}
	if r.Kind != ReqTier || r.Value != "pro" {
		t.Errorf("tier:pro parsed as %+v", r)
	}

	r, err = parseRequirement("global_role:admin", "")
	if err != nil {
		t.Fatalf("global_role:admin: %v", err)
	}
	if r.Kind != ReqGlobalRole || r.Value != "admin" {
		t.Errorf("global_role:admin parsed as %+v", r)
	}

	if _, err := parseRequirement("tier:", ""); err == nil {
		t.Error("empty tier value must fail")
	}
	if _, err := parseRequirement("global_role:", ""); err == nil {
		t.Error("empty global_role value must fail")
	}
}

// TestRequirementKind_String pins the diagnostic spelling for the new
// kinds so YAML authors see predictable reasons in deny messages.
func TestRequirementKind_String(t *testing.T) {
	cases := map[RequirementKind]string{
		ReqTier:       "tier",
		ReqGlobalRole: "global_role",
	}
	for k, want := range cases {
		if got := k.String(); got != want {
			t.Errorf("%d.String() = %q, want %q", k, got, want)
		}
	}
}

// TestEngine_ConcurrentReadWrite exercises the atomic.Pointer swap under
// concurrent readers and writers. The test passes when -race is clean
// and every Evaluate call observes a consistent (non-torn) snapshot.
func TestEngine_ConcurrentReadWrite(t *testing.T) {
	e := mustEngine(t, RuleSpec{Path: "/a", Allow: "anonymous"})

	stop := make(chan struct{})
	done := make(chan struct{}, 8)

	// Four readers: hammer Evaluate, assert Decision is coherent.
	for i := 0; i < 4; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			for {
				select {
				case <-stop:
					return
				default:
				}
				d := e.Evaluate(newGetReq("/a"), Identity{})
				// Matched rule's AppID must be empty (that's what we wrote);
				// if we ever saw a torn value this would flip.
				if d.MatchedRule != nil && d.MatchedRule.AppID != "" {
					t.Errorf("torn MatchedRule.AppID = %q", d.MatchedRule.AppID)
					return
				}
			}
		}()
	}

	// Four writers: flip between two distinct rule sets.
	for i := 0; i < 4; i++ {
		go func(n int) {
			defer func() { done <- struct{}{} }()
			specA := []RuleSpec{{Path: "/a", Allow: "anonymous"}}
			specB := []RuleSpec{
				{Path: "/a", Allow: "anonymous"},
				{Path: "/b", Allow: "anonymous"},
			}
			for iter := 0; iter < 200; iter++ {
				select {
				case <-stop:
					return
				default:
				}
				var err error
				if iter%2 == 0 {
					err = e.SetRules(specA)
				} else {
					err = e.SetRules(specB)
				}
				if err != nil {
					t.Errorf("SetRules: %v", err)
					return
				}
			}
		}(i)
	}

	// Let them race briefly, then stop.
	time.Sleep(50 * time.Millisecond)
	close(stop)
	for i := 0; i < 8; i++ {
		<-done
	}

	// Post-condition: Rules() returns a fresh copy distinct from the
	// internal pointer. Mutating the returned slice must not affect the
	// next Evaluate.
	snap := e.Rules()
	if len(snap) == 0 {
		t.Fatal("expected at least one rule after races settle")
	}
	snap[0] = nil // corrupt caller-owned copy
	d := e.Evaluate(newGetReq("/a"), Identity{})
	if d.MatchedRule == nil {
		t.Error("mutating Rules() return value must not affect Evaluate")
	}
}
