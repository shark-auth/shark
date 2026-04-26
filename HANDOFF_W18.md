# W18 Recovery — Post-Rogue-Agent Handoff

**Date:** 2026-04-26
**Branch:** main (committed locally; not yet pushed)
**Last commit:** `2226285` build: rebuild dist post first-boot UX (W18 task 5)

## Why this exists

A previous agent shipped W17 yaml-deprecation but caused major regressions:
- Nuked Identity Hub UI surface (`identity_hub.tsx` deleted, route unwired).
- Settings Danger Zone overlapped scrolling sections (positioned outside scroll container).
- Dev Email tab auto-redirected operators to /overview when `email.provider != 'dev'`.
- ASCII glyph + admin key never printed on `shark serve` boot (PrintHeader called only by `shark version`; key surfaced only as masked prefix+suffix).
- Smoke test suite: 9 files broke on removed `--dev`/`--config` flags + missing yaml.

W18 = recovery wave. All items below shipped today.

## What shipped (W18)

| Commit | What |
|--------|------|
| `dc8aadb`…`6aec456` | 8 smoke files: stripped yaml/--dev/--config, route via admin API |
| `20cf848` | Delete obsolete `test_proxy_yaml_deprecation.py` (warning source removed in W17 Phase H) |
| `b0ba136`, `74adb79`, `88ba909` | Smoke wave 2: cli_ops, cli_user_sso, dev_inbox tests rewired |
| (in `feat(cli)`) | Wire `SHARK_PORT` / `SHARK_DB_PATH` env vars to `cfg.Server.Port` / `cfg.Storage.Path` (was dead code; tests need it for port isolation) |
| `e1f40a6`, `372ec80`, `da33ef7`, `f5cda5a` | **Identity Hub rebuilt** via /impeccable craft — 987-line component, 6 sections, monochrome strict, wired in App.tsx + layout.tsx + inner_doc |
| `19ac3f6` | **CLI glyph + key banner** — bigger ASCII shark, `cli.PrintHeader` called on every boot, `cli.PrintAdminKeyBanner` shows full `sk_live_*` in 80-col yellow banner with setup URL + key file path |
| `289f355` | **dev-email tab unblocked** + **danger zone scrolls inline** + dropped `--dev` references from dev_email.tsx |
| `6aff3eb`, `6f31961`, `2226285` | **First-boot dashboard UX** — full-width key banner above TopBar after magic-link signin (Copy + Dismiss) → lands on /get-started; small dismissable tutorial tab on /overview while walkthrough not seen |

## Investigation report (cross-cutting)

`INVESTIGATE_REPORT.md` at repo root. Sonnet read-only audit:

### Proxy P0 bugs (NOT yet fixed)

1. **Hop-by-hop headers never stripped** in `internal/proxy/proxy.go director()` — `Connection`, `Upgrade`, `Proxy-Authorization`, `TE`, `Trailers`, `Transfer-Encoding`, `Keep-Alive` pass through to upstream. Header smuggling vector. No test coverage.
2. **DPoP JTI cache never wired into NewListener** — `internal/proxy/listener.go` `NewListener()` lacks `SetDPoPCache`. All multi-listener deployments have DPoP enforcement silently disabled.

### Proxy P1 bugs (NOT yet fixed)

- HalfOpen circuit allows concurrent probes (race → Open→Closed in one tick)
- `validateDPoPProof` swallows specific error detail
- `fullURL` trusts client `X-Forwarded-Proto` (open-redirect via `javascript:` scheme)
- `http.Server` in Listener missing `ReadTimeout` / `WriteTimeout` (Slowloris exposure)
- `probe()` body not drained before close (connection pool poisoning)

### CLI parity gaps

| Resource | Gap |
|---|---|
| webhook | No CLI subcommands (`shark webhook list/create/delete/test`) — API present |
| rbac | No CLI for global roles + permissions — API present |
| organization | CLI has only `org show`; missing create/list/update/delete |
| consent | Parent command exists but empty (no subcommands wired) |
| oauth DCR | No CLI for OAuth client management |
| vault | Only `vault provider show`; missing list/create/update/delete + connection mgmt |
| audit | Only `audit export`; missing `audit list` with filters |
| session | Missing `session purge-expired`, admin-revoke-all |
| api-key | Missing `api-key show <id>` |

## Smoke status (current)

- **Total tests:** 150 (after `test_proxy_yaml_deprecation.py` deletion)
- **Estimated green:** ~95% — exact count blocked by ctx_execute 10min cap; needs dedicated terminal session.
- **Known residual fails (~6):**
  - `test_admin_deep.py::test_admin_org_mgmt` — "Org seed failed" (likely API removal regression)
  - `test_admin_mgmt.py::test_api_key_crud` — 500 on POST
  - `test_admin_mgmt.py::test_sso_connections_crud` — 400 on POST
  - `test_admin_mgmt.py::test_dev_inbox_access` — 404 (endpoint exists per subagent investigation; likely needs `email.provider=dev` PATCH first)
  - `test_w15_advanced.py::test_w15_multi_listener_isolation` — multi-listener config has no admin API equivalent (W17 hard-removed yaml multi-listener); test partially descoped to single-listener invariants
  - `test_w15_gateway.py::test_transparent_gateway_porter` — 404 vs 302 on transparent gateway path

## Files of note (W18)

- `AGENT_BRIEF.md` — full task brief used by all subagents (root)
- `INVESTIGATE_REPORT.md` — proxy + CLI parity findings (root)
- `internal/cli/branding.go` — `PrintAdminKeyBanner` + bigger glyph
- `internal/server/server.go` — env var wiring + banner call orchestration
- `internal/server/firstboot.go` — masked-print removed (full key in banner)
- `cmd/shark/cmd/serve.go` — `cli.PrintHeader` on every boot
- `admin/src/components/identity_hub.tsx` — rebuilt
- `admin/src/components/App.tsx` — first-boot banner + redirect removal
- `admin/src/components/dev_email.tsx` — empty-state copy fix
- `admin/src/components/settings.tsx` — Danger Zone moved into scroll
- `admin/src/components/setup.tsx` — stores key in localStorage post-magic-link
- `admin/src/components/overview.tsx` — tutorial tab

## How to build (Windows)

```
make -f Makefile.windows
```

Or manually:

```
cd admin && npm run build && cd ..
go build -trimpath -ldflags="-s -w" -o shark.exe ./cmd/shark
```

## How to run smoke (orchestrator only — kills shark instances on machine)

```
taskkill /F /IM shark.exe 2>/dev/null
rm -rf tests/smoke/data tests/smoke/shark.db tests/smoke/dev.db tests/smoke/server.log
cp shark.exe tests/smoke/shark.exe
cd tests/smoke
python -m pytest --tb=line --maxfail=200
```

## Open work (next session)

1. **Proxy P0 fixes** — strip hop-by-hop headers + wire DPoP JTI cache into NewListener. See `INVESTIGATE_REPORT.md`.
2. **Proxy P1 fixes** — circuit probe race, X-Forwarded-Proto allowlist, ReadTimeout/WriteTimeout on listener, probe body drain.
3. **CLI parity fills** — webhook, rbac, organization, consent, oauth DCR, vault, audit subcommands. Highest-leverage: webhook + rbac (whole resources missing).
4. **Smoke residual ~6** — see list above. Each is individual-bug investigation, not regression.
5. **Glyph cleanup duplicate** — `==== SharkAuth — Open Source Auth for Agents and Humans ====` mid-boot in firstboot.go duplicates `cli.PrintHeader` at top. Remove the mid-boot line.
6. **HANDOFF_W17.md** — leave as historical record. This file (W18) supersedes it for current state.

## Memory updates worth recording

- `tests/smoke/shark.exe` is gitignored but conftest needs it copied from repo root after every build. Doc this if not already.
- `SHARK_PORT` / `SHARK_DB_PATH` env vars now actually work (W18 wired them; previously dead bootstrap code path).
- `b.AdminKey` in `internal/server/server.go` now sourced from `fbResult.AdminKey` first (W17 path) before `bootstrapAdminKey` (legacy).
