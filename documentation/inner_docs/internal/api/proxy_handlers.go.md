# proxy_handlers.go

**Path:** `internal/api/proxy_handlers.go`
**Package:** `api`
**LOC:** 335
**Tests:** likely integration-tested

## Purpose
Read + simulate surface for the live reverse proxy. Exposes the circuit breaker's runtime state (snapshot + SSE stream), lists compiled rules in their YAML-equivalent shape, and provides a simulator endpoint so dashboard operators can dry-run "what would this request do?".

## Handlers exposed
- `handleProxyStatus` (line 90) — GET `/admin/proxy/status`. 404 when proxy disabled. Returns `{data: proxyStatusPayload}` (state, cache_size, neg_cache_size, failures, last_check, last_latency_ms, last_status, health_url, upstream).
- `handleProxyStatusStream` (line 290) — GET `/admin/proxy/status/stream` (SSE).
- `handleProxyRules` (line 124) — GET `/admin/proxy/rules`. Lists compiled engine rules in user-facing YAML-equivalent shape.
- `handleProxySimulate` (line 199) — POST `/admin/proxy/simulate`. Body `{method, path, identity}`. Returns `{matched_rule, decision, reason, injected_headers, eval_us}`.

## Key types
- `proxyStatusResponse`/`proxyStatusPayload` (lines 18, 26)
- `proxyRuleView` (line 44) — JSON projection of compiled `proxy.Rule` (path, methods, require, scopes).
- `proxySimulateRequest` (line 55), `proxySimulatedIdentity` (line 64), `proxySimulateResponse` (line 79)

## Helpers
- `buildProxyStatusPayload` (line 101) — extracted so SSE + snapshot share one shape.
- `proxyRuleToView` (line 140), `formatRequirement` (line 161) — inverse of `proxy.parseRequirement`, so operators see "role:admin" not internal enums.
- `sortStrings` (line 182) — local insertion sort for method-list determinism.
- `decisionLabel` (line 262), `computeInjectedHeaders` (line 273), `writeProxyStatusEvent` (line 326).

## Imports of note
- `internal/proxy` — `Rule`, `Identity`, `Requirement`, `Breaker.Stats()`
- `internal/identity` — identity helpers

## Wired by
- `internal/api/router.go:675-678`

## Notes
- Status is 404 (not 200 with `enabled:false`) when the proxy isn't wired — lets the dashboard branch on HTTP status.
- Simulator returns evaluation time in microseconds so operators spot pathological rules at a glance.
