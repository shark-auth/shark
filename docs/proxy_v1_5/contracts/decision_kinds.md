# Contract — Decision kinds

`proxy.DecisionKind` classifies the outcome of `Engine.Evaluate`. The proxy handler dispatches on `Kind` to decide between allowing the request, returning 401 (anonymous), returning 403 (authenticated but forbidden), or redirecting to the paywall.

The four kinds are mutually exclusive — every `Evaluate` return carries exactly one.

## The enum

```go
type DecisionKind int

const (
    DecisionAllow DecisionKind = iota
    DecisionDenyAnonymous
    DecisionDenyForbidden
    DecisionPaywallRedirect
)
```

## When each fires

### DecisionAllow

A rule matched and all predicates passed (path, method, app, Require, Scopes, M2M). The handler forwards to the upstream.

**Decision fields populated:**
- `Allow = true`
- `Kind = DecisionAllow`
- `MatchedRule` points at the compiled rule
- `Reason`, `PaywallApp`, `RequiredTier` are empty

**HTTP outcome:** 2xx/3xx/4xx/5xx — whatever the upstream returns. The proxy passes through transparently after injecting `X-User-*` / `X-Agent-*` headers.

### DecisionDenyAnonymous

A rule matched (or no rule matched and default-deny applies) AND the caller carried no authenticated principal (`Identity.IsAnonymous()` — no UserID and no AgentID). The intent is "login first, then we'll re-check" rather than "you specifically cannot have this".

**HTTP outcome:**
- Browser callers (Accept header contains `text/html`): `302 Found` to the hosted login page with `return=<original-url>` query param.
- API callers: `401 Unauthorized` with a `WWW-Authenticate: Bearer` header and a JSON error body.

### DecisionDenyForbidden

The caller is authenticated (UserID or AgentID set) but lacks the required attribute — missing role, missing scope, not an agent when the rule required `agent`, M2M rule hit by a human, etc.

**HTTP outcome:** `403 Forbidden` with a JSON error body. Browser callers are NOT redirected — this is a "the logged-in user cannot have this" deny and login won't fix it.

**Reason strings you'll see:**
- `"role \"admin\" required"`
- `"scope \"webhooks:write\" required"`
- `"agent authentication required"`
- `"rule requires agent (m2m)"` (from the M2M gate — see `m2m_rule_flag.md`)

### DecisionPaywallRedirect

The caller is authenticated but on the wrong tier for this rule (`ReqTier` mismatch). Semantically "wrong plan, not wrong person" — the handler redirects to the paywall so the caller can upgrade rather than hitting a generic 403.

**Decision fields populated:**
- `Allow = false`
- `Kind = DecisionPaywallRedirect`
- `RequiredTier = "pro"` (or whatever the rule wanted)
- `PaywallApp` — app slug for the paywall page (empty when not configured; the handler falls back to 403)

**HTTP outcome:**
- When `PaywallApp` is set: `302 Found` to `/paywall/{PaywallApp}?tier={RequiredTier}&return=<original-url>`.
- When `PaywallApp` is empty: `403 Forbidden` with a reason string identifying the tier mismatch.

### Special case: anonymous hitting a `ReqTier` rule

When the caller is anonymous AND the rule requires a tier, Kind is overridden to `DecisionDenyAnonymous` — they need to sign in first before we can know their tier. `RequiredTier` is still populated for diagnostics.

## Dispatch table

| Decision.Kind | Allow | HTTP status | Body |
|---|---|---|---|
| DecisionAllow | true | upstream | upstream |
| DecisionDenyAnonymous | false | 302 (browser) / 401 (API) | login redirect / error JSON |
| DecisionDenyForbidden | false | 403 | error JSON with reason |
| DecisionPaywallRedirect | false | 302 (with PaywallApp) / 403 (without) | paywall redirect / error JSON |

## Why the four-way split matters

A naive proxy returns 401 or 403 and lets the caller figure it out. That collapses three very different failure modes onto two status codes. Splitting them lets the dashboard render three distinct UI surfaces — "login", "you don't have access", "upgrade your plan" — and lets the proxy route paywall denies to a branded page without surfacing a generic 403 to browsers.

For SDKs, every method that can trigger an authorization check (e.g. `listProxyRules`, which surfaces the proxy simulator) should model the return as `Allow | DenyAnonymous | DenyForbidden | PaywallRedirect` rather than a boolean — lossy boolean returns force clients to re-parse the Reason string.
