# identity.go

**Path:** `internal/identity/identity.go`  
**Package:** `identity`  
**LOC:** 152  
**Tests:** `identity_test.go`

## Purpose
Canonical request Identity definition. Auth middleware stashes Identity on request context; downstream consumers (proxy, rule engine, admin handlers) read it via FromContext. Union of all authenticated principal fields safe to propagate post-auth-decision.

## Key types / functions
- `AuthMethod` (type, line 21) — cookie, jwt, apikey, dpop, anonymous
- `ActorType` (type, line 43) — human, agent
- `Identity` (struct, line 61) — UserID, UserEmail, AgentID, AgentName, APIKeyID, Tier, Roles, Scopes, AuthMethod, ActorType, SessionID, MFAPassed, CacheAge
- `WithIdentity(ctx, id)` (func, line 114) — stash Identity on context
- `FromContext(ctx)` (func, line 121) — retrieve Identity + bool (ok)
- `IsAnonymous()` (func, line 129) — check UserID=="" && AgentID==""
- `HasRole(name)` (func, line 135) — test role membership
- `HasScope(scope)` (func, line 145) — test scope membership

## Imports of note
- `context` — context.WithValue, context.Value
- `time` — CacheAge duration

## Wired by
- Auth middleware (internal/auth/*.go) calls WithIdentity after successful auth
- Proxy rule engine queries identity via FromContext
- Admin handlers check identity fields for RBAC
- Reverse proxy injects identity fields as X-* upstream headers

## Notes
- Zero values valid and mean "not set"; e.g., empty Tier = "no tier claim", not "free"
- Either UserID OR AgentID set, not both (typically)
- Scopes: OAuth/agent scope list granted to request (not emitted upstream as header)
- SessionID: non-empty when request rides a session (cookie or session JWT)
- CacheAge: staleness of cached session-resolution (0 = fresh, >0 = from circuit-breaker cache)
- Context key: unexported struct{} type (prevents collisions per context.WithValue convention)

