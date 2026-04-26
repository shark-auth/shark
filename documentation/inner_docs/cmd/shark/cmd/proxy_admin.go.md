# proxy_admin.go

**Path:** `cmd/shark/cmd/proxy_admin.go`
**Package:** `cmd`
**LOC:** 407
**Tests:** proxy_admin_test.go

## Purpose
Builds the real `shark proxy ...` command tree backed by the admin HTTP API: lifecycle (start/stop/reload/status) + DB-backed rules CRUD (list/add/show/delete/import). Also home of shared helpers `extractData`, `extractDataArray`, and `openBrowser`.

## Key types / functions
- `init` (line 35) — replaces `proxyCmd` stub from proxy.go; attaches lifecycle subcommands + rules sub-tree; declares all flags for `proxy rules add`.
- `proxyStartCmd`, `proxyStopCmd`, `proxyReloadCmd`, `proxyStatusCmd` (lines 92-114) — wrappers built via `proxyLifecycleAction`.
- `proxyLifecycleAction` (func, line 117) — closure factory for lifecycle commands; renders `state listeners rules_loaded last_error`.
- `proxyRulesCmd` (var, line 149) — parent for rules subcommands.
- `proxyRulesListCmd` (line 154) — tabwriter table output.
- `proxyRulesAddCmd` (line 195) — full payload assembly with idempotent `--id` (409 → exit 2).
- `proxyRulesShowCmd` (line 276), `proxyRulesDeleteCmd` (line 297).
- `extractData` (func, line 367), `extractDataArray` (func, line 375).
- `openBrowser` (func, line 394) — `xdg-open`/`open`/`cmd /c start` per OS.

## Imports of note
- `os/exec`, `runtime`, `text/tabwriter`.
- Uses sibling `adminDo`, `apiError`, `maybeJSONErr`.

## Wired by / used by
- Backs `internal/api/proxy_admin_handlers.go` endpoints (`/api/v1/admin/proxy/...`).
- `openBrowser` is reused by `paywall.go`.

## Notes
- Lane E, milestone E1.
- `--require` / `--allow` are mutually exclusive; `name` defaults to `path` when omitted.
- Conflict on `--id` exits 2 to match the idempotency spec.
