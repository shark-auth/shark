package proxy

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

// RuleSpec is the raw, pre-compile shape of a rule. It mirrors the
// user-facing config.ProxyRule but is duplicated here so internal/proxy
// does not import internal/config (which would create a cycle at wiring
// time and make this package harder to test in isolation). Callers
// translate config.ProxyRule into RuleSpec at New-engine time.
type RuleSpec struct {
	AppID   string
	Path    string
	Methods []string
	Require string
	Allow   string
	Scopes  []string
}

// RequirementKind enumerates the predicate family for a rule. Each kind
// has a slightly different evaluation path; see Engine.evaluateRequirement.
type RequirementKind int

const (
	// ReqAnonymous matches any caller — authenticated or not. Use for
	// genuinely public paths. Does NOT require the caller to be anonymous;
	// it simply waives the auth check.
	ReqAnonymous RequirementKind = iota
	// ReqAuthenticated matches any caller with a resolved user or agent.
	ReqAuthenticated
	// ReqRole matches callers whose UserRoles contains Value.
	ReqRole
	// ReqPermission is the RBAC hook (Phase 6.5). MVP: always deny with a
	// clear reason so operators know the rule is recognized but the
	// lookup plumbing is still pending.
	ReqPermission
	// ReqAgent matches callers authenticated as an agent (AgentID set).
	ReqAgent
	// ReqScope matches callers whose Scopes contains Value.
	ReqScope
)

// String renders a RequirementKind for diagnostic messages. Uses the
// same spelling users write in YAML so a rule's "reason" text reads as
// a mirror of its configuration.
func (k RequirementKind) String() string {
	switch k {
	case ReqAnonymous:
		return "anonymous"
	case ReqAuthenticated:
		return "authenticated"
	case ReqRole:
		return "role"
	case ReqPermission:
		return "permission"
	case ReqAgent:
		return "agent"
	case ReqScope:
		return "scope"
	default:
		return "unknown"
	}
}

// Requirement is the compiled predicate for a single rule. Value's
// meaning depends on Kind:
//   - ReqRole: role name, e.g. "admin"
//   - ReqPermission: "resource:action" (not yet evaluated)
//   - ReqScope: scope string, e.g. "webhooks:write"
//   - ReqAnonymous/ReqAuthenticated/ReqAgent: unused (Value == "")
type Requirement struct {
	Kind  RequirementKind
	Value string
}

// patternSegment is one URL path segment in a compiled pattern.
// A segment is either a literal (exact string match) or a wildcard
// (matches any single segment). Trailing "/*" is represented separately
// on pathPattern — not as a segment — because it matches zero or more
// segments rather than exactly one.
type patternSegment struct {
	literal  string
	wildcard bool
}

// pathPattern is a compiled rule path. trailing==true means the pattern
// ended with "/*" and therefore matches the prefix formed by segments
// plus any number of additional segments (including zero — so "/api/*"
// matches "/api").
type pathPattern struct {
	segments []patternSegment
	trailing bool
	raw      string
}

// Rule is a compiled rule ready for matching at request time. Fields
// are unexported where they're implementation detail (pattern) and
// exported where they're useful for diagnostics or the simulator API
// landing in P4 (Path, Methods, Require, Scopes).
type Rule struct {
	AppID   string
	Path    string
	pattern pathPattern
	Methods map[string]struct{} // empty = any method
	Require Requirement
	Scopes  []string
}

// Matches reports whether r should be considered for request q. Path,
// method and AppID must all match. requirement evaluation happens separately
// in Engine.Evaluate so method/app misses fall through to subsequent rules
// rather than hard-denying.
func (r *Rule) Matches(method, urlPath, appID string) bool {
	if r.AppID != "" && r.AppID != appID {
		return false
	}
	if !r.MethodAllowed(method) {
		return false
	}
	return r.pattern.match(urlPath)
}

// MethodAllowed reports whether method satisfies the rule's method
// filter. Empty Methods means any method matches. Comparison is
// case-insensitive so "get" in YAML still matches r.Method=="GET".
func (r *Rule) MethodAllowed(method string) bool {
	if len(r.Methods) == 0 {
		return true
	}
	_, ok := r.Methods[strings.ToUpper(method)]
	return ok
}

// method must matches; requirement evaluation happens separately
func (r *Rule) MethodMatches(method string) bool {
	return r.MethodAllowed(method)
}

// Decision is what Engine.Evaluate returns: the allow/deny bit, which
// rule drove the decision (nil if nothing matched), and a human-readable
// reason surfaced to operators via the X-Shark-Deny-Reason response
// header and logs.
type Decision struct {
	Allow       bool
	MatchedRule *Rule
	Reason      string
}

// Engine is the compiled rule set. The rules slice is guarded by a RWMutex
// so SetRules can atomically swap the entire compiled list under a write
// lock while Evaluate continues serving readers. Reads take a brief read
// lock and snapshot the slice header (not its contents) so the per-request
// hot path avoids contention with mutators.
type Engine struct {
	mu          sync.RWMutex
	rules       []*Rule
	defaultDeny bool
}

// NewEngine compiles raw rule specs into an Engine. Every spec is
// validated individually; the first error aborts compilation so
// operators see the earliest configuration problem rather than a
// cascade. defaultDeny is always true in the MVP — kept as a field so
// future work can expose it if a truly permissive mode is ever needed.
func NewEngine(raw []RuleSpec) (*Engine, error) {
	compiled, err := compileSpecs(raw)
	if err != nil {
		return nil, err
	}
	return &Engine{rules: compiled, defaultDeny: true}, nil
}

// SetRules atomically replaces the engine's compiled rule set. Called by the
// admin proxy-rules CRUD endpoints (Wave D) after every mutation so DB rows
// take effect without restarting the server. The compile step runs before
// the write lock is taken so a partially-compiled set is never visible to
// concurrent Evaluate calls.
//
// On compile failure the previous rule set remains in place — the caller
// gets the error and is expected to surface it; the proxy keeps serving
// the last-known-good configuration rather than returning blanket denies.
func (e *Engine) SetRules(raw []RuleSpec) error {
	compiled, err := compileSpecs(raw)
	if err != nil {
		return err
	}
	e.mu.Lock()
	e.rules = compiled
	e.mu.Unlock()
	return nil
}

// compileSpecs is the shared compile loop used by NewEngine + SetRules.
// Extracted so both paths produce identical error wrapping ("proxy: rule N
// (path): <inner>").
func compileSpecs(raw []RuleSpec) ([]*Rule, error) {
	compiled := make([]*Rule, 0, len(raw))
	for i, spec := range raw {
		rule, err := compileRule(spec)
		if err != nil {
			return nil, fmt.Errorf("proxy: rule %d (%q): %w", i, spec.Path, err)
		}
		compiled = append(compiled, rule)
	}
	return compiled, nil
}

// Rules returns a snapshot of the compiled rules. Intended for diagnostics
// and the simulator API. The returned slice header is a fresh copy so
// callers can iterate safely while SetRules races; the *Rule values inside
// are immutable post-compilation so sharing them is safe.
func (e *Engine) Rules() []*Rule {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]*Rule, len(e.rules))
	copy(out, e.rules)
	return out
}

// Evaluate finds the first rule whose path + method matches the inbound
// request and returns its decision. If no rule matches, the default
// behavior is deny with a clear reason so the operator's logs explain
// the 403.
func (e *Engine) Evaluate(r *http.Request, id Identity) Decision {
	e.mu.RLock()
	rules := e.rules
	defaultDeny := e.defaultDeny
	e.mu.RUnlock()

	// Extract AppID header if present
	appID := r.Header.Get("X-Shark-App-ID")

	for _, rule := range rules {
		if !rule.Matches(r.Method, r.URL.Path, appID) {
			continue
		}
		allow, reason := e.evaluateRequirement(rule.Require, rule.Scopes, id)
		return Decision{Allow: allow, MatchedRule: rule, Reason: reason}
	}

	return Decision{Allow: !defaultDeny, Reason: "no rule matched"}
}

// evaluateRequirement runs the predicate for req plus the AND-combined
// extraScopes list against id. Extracted from Evaluate so rule-testing
// tools can call it directly with a synthetic Identity (simulator API).
func (e *Engine) evaluateRequirement(req Requirement, extraScopes []string, id Identity) (bool, string) {
	// Primary requirement first. If it fails, skip extra-scope evaluation
	// — the reason the operator cares about is the primary one.
	if ok, reason := evaluatePrimary(req, id); !ok {
		return false, reason
	}
	// Extra scopes AND with the primary. Every listed scope must be
	// granted; the first missing one is surfaced.
	for _, s := range extraScopes {
		if !containsString(id.Scopes, s) {
			return false, fmt.Sprintf("scope %q required", s)
		}
	}
	return true, ""
}

// evaluatePrimary dispatches on the requirement kind. Split out from
// evaluateRequirement so the control flow reads top-to-bottom and each
// predicate's reason string lives next to its check.
func evaluatePrimary(req Requirement, id Identity) (bool, string) {
	switch req.Kind {
	case ReqAnonymous:
		return true, ""
	case ReqAuthenticated:
		if id.UserID != "" || id.AgentID != "" {
			return true, ""
		}
		return false, "authentication required"
	case ReqRole:
		if containsString(id.UserRoles, req.Value) {
			return true, ""
		}
		return false, fmt.Sprintf("role %q required", req.Value)
	case ReqPermission:
		// Phase 6.5 will wire in the RBAC permission store. Until then
		// we fail closed — better a visible deny in dev than a silent
		// allow in prod.
		return false, fmt.Sprintf("permission %q required (permission-based rules not yet implemented)", req.Value)
	case ReqAgent:
		if id.AgentID != "" {
			return true, ""
		}
		return false, "agent authentication required"
	case ReqScope:
		if containsString(id.Scopes, req.Value) {
			return true, ""
		}
		return false, fmt.Sprintf("scope %q required", req.Value)
	default:
		return false, "unknown requirement"
	}
}

// containsString is a tiny helper for slice-membership checks on the
// few string slices we care about (roles, scopes). Linear scan is fine
// — these slices are tiny and run in request-critical paths where an
// allocation-free implementation beats a map build-up.
func containsString(haystack []string, needle string) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}
	return false
}

// compileRule builds a Rule from a RuleSpec, validating all fields. The
// errors returned are wrapped by NewEngine with the rule index and path
// so operators get a pinpoint diagnostic.
func compileRule(spec RuleSpec) (*Rule, error) {
	pattern, err := compilePath(spec.Path)
	if err != nil {
		return nil, err
	}

	req, err := parseRequirement(spec.Require, spec.Allow)
	if err != nil {
		return nil, err
	}

	methods := make(map[string]struct{}, len(spec.Methods))
	for _, m := range spec.Methods {
		trimmed := strings.TrimSpace(m)
		if trimmed == "" {
			return nil, errors.New("empty method in methods list")
		}
		methods[strings.ToUpper(trimmed)] = struct{}{}
	}

	return &Rule{
		AppID:   spec.AppID,
		Path:    spec.Path,
		pattern: pattern,
		Methods: methods,
		Require: req,
		Scopes:  append([]string(nil), spec.Scopes...),
	}, nil
}

// parseRequirement resolves the require/allow string pair into a
// Requirement. Shaped as parseRequirement(require, allow) so callers
// don't have to juggle ordering; the error messages mention whichever
// field the operator actually wrote.
func parseRequirement(require, allow string) (Requirement, error) {
	require = strings.TrimSpace(require)
	allow = strings.TrimSpace(allow)

	if require != "" && allow != "" {
		return Requirement{}, errors.New("rule has both require and allow; choose one")
	}
	if require == "" && allow == "" {
		return Requirement{}, errors.New("rule must set require or allow")
	}

	if allow != "" {
		if allow != "anonymous" {
			return Requirement{}, fmt.Errorf("allow %q: only \"anonymous\" is supported", allow)
		}
		return Requirement{Kind: ReqAnonymous}, nil
	}

	switch {
	case require == "anonymous":
		return Requirement{Kind: ReqAnonymous}, nil
	case require == "authenticated":
		return Requirement{Kind: ReqAuthenticated}, nil
	case require == "agent":
		return Requirement{Kind: ReqAgent}, nil
	case strings.HasPrefix(require, "role:"):
		value := strings.TrimPrefix(require, "role:")
		if value == "" {
			return Requirement{}, errors.New("role: requires a value, e.g. role:admin")
		}
		return Requirement{Kind: ReqRole, Value: value}, nil
	case strings.HasPrefix(require, "permission:"):
		value := strings.TrimPrefix(require, "permission:")
		if value == "" {
			return Requirement{}, errors.New("permission: requires a value, e.g. permission:users:read")
		}
		return Requirement{Kind: ReqPermission, Value: value}, nil
	case strings.HasPrefix(require, "scope:"):
		value := strings.TrimPrefix(require, "scope:")
		if value == "" {
			return Requirement{}, errors.New("scope: requires a value, e.g. scope:webhooks:write")
		}
		return Requirement{Kind: ReqScope, Value: value}, nil
	default:
		return Requirement{}, fmt.Errorf("unknown require %q (expected anonymous, authenticated, agent, role:X, permission:X:Y, or scope:X)", require)
	}
}

// compilePath turns a chi-style path pattern into a pathPattern ready
// for segment-by-segment matching. Supports:
//
//   - exact: /api/foo
//   - prefix: /api/foo/*  (matches /api/foo and everything under it)
//   - single-segment wildcard: /api/*/deep
//   - {param} placeholder: /api/{id} (treated same as /api/*)
//
// The leading "/" is required — patterns without it are rejected so
// typos like "api/foo" don't silently not match anything.
func compilePath(p string) (pathPattern, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return pathPattern{}, errors.New("path is required")
	}
	if !strings.HasPrefix(p, "/") {
		return pathPattern{}, fmt.Errorf("path %q must start with '/'", p)
	}

	// Strip the leading slash for segmentation; restore meaning via the
	// fact that we ALWAYS compare against paths that also have one.
	trimmed := strings.TrimPrefix(p, "/")

	// Trailing /* is prefix-match sugar. Strip it and set the trailing
	// flag. The edge case "/*" alone becomes an empty-segments pattern
	// with trailing=true, which matches everything — intentional.
	trailing := false
	if trimmed == "*" {
		return pathPattern{segments: nil, trailing: true, raw: p}, nil
	}
	if strings.HasSuffix(trimmed, "/*") {
		trailing = true
		trimmed = strings.TrimSuffix(trimmed, "/*")
	}

	// After trimming a possible trailing /*, split remaining literal or
	// wildcard segments.
	var segments []patternSegment
	if trimmed != "" {
		for _, seg := range strings.Split(trimmed, "/") {
			if seg == "" {
				return pathPattern{}, fmt.Errorf("path %q contains an empty segment", p)
			}
			switch {
			case seg == "*":
				segments = append(segments, patternSegment{wildcard: true})
			case strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}"):
				// {id}-style placeholder: treat as single-segment wildcard.
				// MVP does not capture the value — rules engine only needs
				// match/no-match, not extraction.
				segments = append(segments, patternSegment{wildcard: true})
			default:
				segments = append(segments, patternSegment{literal: seg})
			}
		}
	}

	return pathPattern{segments: segments, trailing: trailing, raw: p}, nil
}

// match reports whether urlPath satisfies p. Matching is case-sensitive
// (HTTP convention — /API/foo is a different resource from /api/foo).
func (p pathPattern) match(urlPath string) bool {
	if !strings.HasPrefix(urlPath, "/") {
		// Every normalized http.Request.URL.Path starts with "/"; be
		// defensive for handcrafted Requests used in tests.
		urlPath = "/" + urlPath
	}
	trimmed := strings.TrimPrefix(urlPath, "/")

	// "/*" — trailing wildcard, zero segments — matches every path.
	if len(p.segments) == 0 && p.trailing {
		return true
	}

	// Root or empty-trimmed edge. An empty segments+non-trailing
	// pattern means the rule path was "/" alone — match exact root.
	if trimmed == "" {
		return len(p.segments) == 0 && !p.trailing
	}

	parts := strings.Split(trimmed, "/")

	if p.trailing {
		// Prefix match: every pattern segment must match the
		// corresponding request segment, and the request may have
		// extra segments beyond them (zero or more).
		if len(parts) < len(p.segments) {
			return false
		}
		for i, seg := range p.segments {
			if !seg.matches(parts[i]) {
				return false
			}
		}
		return true
	}

	// Exact match: segment counts must align.
	if len(parts) != len(p.segments) {
		return false
	}
	for i, seg := range p.segments {
		if !seg.matches(parts[i]) {
			return false
		}
	}
	return true
}

// matches checks a single path segment against a compiled patternSegment.
// Wildcard segments accept any non-empty value (empty would only arise
// from a malformed input we already reject in compilePath).
func (s patternSegment) matches(part string) bool {
	if s.wildcard {
		return part != ""
	}
	return s.literal == part
}
