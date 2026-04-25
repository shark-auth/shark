# config.go

**Path:** `internal/proxy/config.go`
**Package:** `proxy`
**LOC:** 73
**Tests:** see `proxy_test.go` and siblings

## Purpose
Defines the `Config` struct + `Validate()` for the SharkAuth reverse proxy that fronts a single upstream and injects identity headers. Pure data + validation; no behaviour.

## Key types
- `Config` struct fields:
  - `Enabled bool` — master switch. `Validate()` returns nil when false.
  - `Upstream string` — backend base URL (scheme+host required, path/query ignored).
  - `Timeout time.Duration` — per-request upstream timeout (default 30s).
  - `BufferSize int` — response buffer; 0 = `httputil` default.
  - `TrustedHeaders []string` — allowlist preserved through `StripIdentityHeaders`.
  - `StripIncoming bool` — strip inbound `X-User-*` / `X-Agent-*` / `X-Shark-*` from clients (default true; only secure setting).
  - `Rules []RuleSpec` — first-match-wins authorization rules; informational, real engine compiled separately.
- `DefaultTimeout = 30 * time.Second`.
- `Validate() error` — fails only when enabled and `Upstream == ""`.

## Imports
- `errors`, `time`.

## Wired by
- `cmd/sharkauth/server.go` (proxy init) and `internal/proxy/proxy.go` (`New()`).

## Used by
- `ReverseProxy` constructor + admin/CLI surfaces that round-trip proxy settings via YAML.

## Notes
- Struct stays YAML-serializable on purpose — compiled rule engine is passed separately to `New()`.
- `Rules` field documents intent; lifecycle/CRUD lives in `internal/proxy/rules*.go` and `internal/admin/proxyrules*.go`.
