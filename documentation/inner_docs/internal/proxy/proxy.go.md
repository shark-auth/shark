# proxy.go

**Path:** `internal/proxy/proxy.go`
**Package:** `proxy`
**LOC:** 502
**Tests:** `proxy_test.go`

## Purpose
Reverse proxy handler: request forwarding, identity header injection, panic recovery, DPoP proof validation, upstream timeout enforcement.

## Key types / functions
- `ReverseProxy` (struct, line 54) — Config, upstream URL, Engine, AppResolver, httputil.ReverseProxy backend, DPoP cache, logger
- `ResolvedApp` (struct, line 38) — result of dynamic host lookup (ID, Slug, IntegrationMode, UpstreamURL, Engine)
- `AppResolver` (interface, line 47) — ResolveApp(ctx, host) → (*ResolvedApp, error)
- `New()` (line 70) — constructs ReverseProxy from Config + Engine; validates, parses upstream, wires httputil.ReverseProxy
- `ReverseProxy.ServeHTTP()` (line 156) — panic recovery, strip/inject identity headers, DPoP validation, rule evaluation, forward to upstream
- `ReverseProxy.director()` — rewrites request Host/RequestURI/URL for upstream; injects X-Shark-* identity headers
- `ReverseProxy.SetResolver()` (line 130) — install dynamic app resolver (takes precedence over static upstream)
- `ReverseProxy.SetDPoPCache()` (line 143) — wire DPoP JTI replay cache for bearer proof validation
- `ReverseProxy.validateDPoPProof()` — validates DPoP proof against bearer + JTI cache

## Imports of note
- `net/http/httputil` — ReverseProxy backend
- `internal/identity` — Identity context passing
- `internal/oauth` — DPoP cache

## Wired by
- `internal/proxy/listener.go` (NewListener constructs ReverseProxy with Config + Engine)
- Server boot path that calls SetResolver, SetIssuer, SetDPoPCache

## Used by
- Listener's http.Server.Handler chain for request forwarding
- Dashboard to configure upstream/resolver at runtime

## Notes
- Panic recovery always replies 503 so buggy handlers/transports never crash server (line 157).
- StripIncoming flag controls whether to strip client-supplied identity headers (line 171).
- DPoP validation only runs when identity.AuthMethod == AuthMethodDPoP AND dpopCache is wired (line 187).
- Rules engine (Engine.Evaluate) called in ServeHTTP; if no rule matches, default is deny (line 309).
- Static upstream Config fallback when no AppResolver set (line 84).
- Transport clone avoids mutating process-wide http.DefaultTransport (line 106).
