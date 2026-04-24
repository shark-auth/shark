// Package identity is the canonical definition of SharkAuth's request
// Identity. Auth middleware stashes an Identity on the request context via
// WithIdentity; downstream consumers (the reverse proxy, rule engine,
// admin handlers) read it back via FromContext.
//
// The struct is intentionally wider than any single call site needs — it
// is the union of every field that is safe to propagate after the auth
// decision has been made. Consumers should treat zero-valued fields as
// "absent" rather than "false" — e.g. an empty Tier means "no tier
// claim", not "free tier".
package identity

import (
	"context"
	"time"
)

// AuthMethod enumerates how the caller proved identity on the inbound
// request. The set is closed: anything not in this set is treated as
// anonymous by downstream policy.
type AuthMethod string

const (
	// AuthMethodCookie — browser session cookie (shark_session).
	AuthMethodCookie AuthMethod = "cookie"
	// AuthMethodJWT — Bearer access or session JWT (RS256/ES256).
	AuthMethodJWT AuthMethod = "jwt"
	// AuthMethodAPIKey — sk_live_* API key. Implies ActorTypeAgent.
	AuthMethodAPIKey AuthMethod = "apikey"
	// AuthMethodDPoP — DPoP-bound Bearer token (RFC 9449). The DPoP
	// proof header is validated at the proxy edge before the rule
	// engine runs.
	AuthMethodDPoP AuthMethod = "dpop"
	// AuthMethodAnonymous is the sentinel for unauthenticated requests.
	// Not listed in the spec enum but carried for back-compat with the
	// existing proxy's anonymous-path logic.
	AuthMethodAnonymous AuthMethod = "anonymous"
)

// ActorType distinguishes human callers (session/JWT/cookie) from
// machine/agent callers (API key, DPoP-bound M2M tokens). Used by the
// audit log and by rules that want to restrict a path to machines only.
type ActorType string

const (
	// ActorTypeHuman — request was authenticated as a person.
	ActorTypeHuman ActorType = "human"
	// ActorTypeAgent — request was authenticated as an agent/service
	// (API key, M2M token, DPoP-bound machine credential).
	ActorTypeAgent ActorType = "agent"
)

// Identity is the resolved principal for an inbound request. Auth
// middleware populates it; the rule engine + reverse proxy consume it.
// Zero values for individual fields are valid and mean "not set".
//
// Fields mirror PROXYV1_5 §3. Extra fields (AgentName, CacheAge) are
// retained so the existing proxy's header-injection + circuit-breaker
// cache plumbing keep working during the migration — they are NOT part
// of the rule-engine authorization surface.
type Identity struct {
	// Core principal — exactly one of UserID/AgentID is typically set.
	UserID    string
	UserEmail string
	AgentID   string
	// AgentName is the human-readable name for the agent (e.g. "claude").
	// Injected as X-Agent-Name on the upstream request. Not used by the
	// rule engine.
	AgentName string

	// APIKeyID is the api_keys row ID when AuthMethod == AuthMethodAPIKey.
	APIKeyID string

	// Tier is the caller's plan/tier ("free" | "pro" | ...), sourced from
	// users.metadata JSON. Empty means unset (treated as "no tier" by
	// ReqTier predicates).
	Tier string

	// Roles is the caller's global-role name list. Populated from the
	// RBAC store at token issuance time and baked into the access JWT.
	// ReqRole/ReqGlobalRole rules test membership against this slice.
	Roles []string

	// Scopes is the OAuth/agent scope list granted to this request.
	// Used by ReqScope rules and the AND-combined extra-scopes
	// constraint. Not emitted as an upstream header — scopes drive
	// authorization decisions in front of the upstream.
	Scopes []string

	AuthMethod AuthMethod
	ActorType  ActorType

	// SessionID is the session row ID when the request rides a session
	// (cookie or session JWT). Empty for stateless bearer tokens.
	SessionID string
	// MFAPassed mirrors the matching JWT/session claim.
	MFAPassed bool

	// CacheAge is the staleness of a cached session-resolution result.
	// 0 when the identity was resolved live; >0 when served from the
	// circuit-breaker cache. Emitted as X-Shark-Cache-Age on the
	// upstream request.
	CacheAge time.Duration
}

// identityCtxKey is the unexported context key used to stash Identity.
// Using a struct type (not a string) prevents collisions with other
// packages per the context.WithValue convention.
type identityCtxKey struct{}

// WithIdentity returns a copy of ctx carrying id. Auth middleware calls
// this once it has resolved the request's principal; downstream handlers
// retrieve the value via FromContext.
func WithIdentity(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, identityCtxKey{}, id)
}

// FromContext returns the Identity stashed by WithIdentity and whether
// one was present. When ok is false callers should treat the request as
// anonymous.
func FromContext(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(identityCtxKey{}).(Identity)
	return id, ok
}

// IsAnonymous reports whether id carries no authenticated principal.
// Used by the proxy deny path to distinguish 401 (no credentials) from
// 403 (authenticated but unauthorized).
func (id Identity) IsAnonymous() bool {
	return id.UserID == "" && id.AgentID == ""
}

// HasRole reports whether id.Roles contains name. Linear scan — role
// lists are tiny and this runs on the request hot path.
func (id Identity) HasRole(name string) bool {
	for _, r := range id.Roles {
		if r == name {
			return true
		}
	}
	return false
}

// HasScope reports whether id.Scopes contains scope.
func (id Identity) HasScope(scope string) bool {
	for _, s := range id.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}
