# listener.go

**Path:** `internal/proxy/listener.go`
**Package:** `proxy`
**LOC:** 203
**Tests:** `listener_test.go`

## Purpose
Single reverse-proxy listener bound to its own port with rules engine, circuit breaker, and backing http.Server. Enables multi-listener architecture (one Shark binary protecting N apps on native ports).

## Key types / functions
- `Listener` (struct, line 21) — Bind address, Upstream URL, IsTransparent flag, http.Server, Engine, Breaker, ReverseProxy, authWrap middleware
- `ListenerParams` (struct, line 44) — configuration bundle for NewListener (Bind, Upstream, Rules, BreakerConfig, AuthWrap, Logger)
- `NewListener()` (line 67) — validates config, compiles rules into Engine, constructs Breaker and ReverseProxy, prepares http.Server (NOT started)
- `Listener.Start()` (line 164) — pre-binds port via net.Listen (synchronous error on bind failure), starts Breaker, spawns serve goroutine
- `Listener.Shutdown()` (line 195) — graceful drain and Breaker.Stop()
- `Listener.SetAuthWrap()` (line 145) — install identity-resolution middleware after construction
- `Listener.Engine()` (line 127) — export compiled rules for dashboard
- `Listener.Breaker()` (line 130) — export circuit breaker for dashboard

## Imports of note
- `net.Listen` — pre-bind port to surface errors synchronously
- `http.Server` — backing server for each listener
- `ReverseProxy` — proxy handler

## Wired by
- `internal/api/proxy_lifecycle_handlers.go` (via Manager builder closure)
- Server.Build() that wires AuthWrap middleware

## Used by
- Manager as part of listener pool lifecycle
- Dashboard accessing Engine and Breaker state

## Notes
- NewListener validates and pre-builds but does NOT bind port (line 67).
- Start pre-binds synchronously so partial-Running state never observable (line 169).
- IsTransparent flag allows listener on tproxy-captured ports (line 24).
- AuthWrap wraps handler AFTER rules engine so identity is resolved before rule matching (line 104).
- Listener itself is inert until Start() — safe to construct and SetAuthWrap before serving (line 145).
