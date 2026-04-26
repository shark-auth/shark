# W17 yaml-deprecation — Handoff

**Date:** 2026-04-26
**Branch:** main (pushed to origin)
**Last commit:** `d162c70` fix(serve): wire RunFirstBoot into Build, default email=dev, write admin.key.firstboot

## What shipped

YAML fully removed. Source of truth = SQLite. `./shark serve` boots from defaults, generates admin key on first boot.

### Phase commits (oldest → newest)

| Phase | Commit | What |
|-------|--------|------|
| A | `9b8b10e` | DB-backed `system_config` + `secrets` tables, bootstrap split |
| F | `a16da52` | Frontend yaml-string sweep (3 violations) |
| E | `68e9601` | CLI commands → admin API + `shark cli` + `import-rules` |
| G | `c451425` | Branded ASCII glyph + slog handler + Print* helpers |
| B | `a930b04` | First-boot magic + magic-link signin + `/admin/setup` page |
| C | `e345cb1` | `shark mode` + `shark reset` + Danger Zone (graceful-restart fallback) |
| D | `b64ecc7` | Decompose `--dev` flag into runtime toggles |
| H | `3317c6d` | Hard yaml removal (loader, --config, koanf yaml/file, sharkauth.yaml, init cmd) |
| build | `dc26448` | Rebuild dist post-W17 |
| recover | `94f3ee2` | Restore firstboot.go + setup_handlers.go (lost in B merge) |
| firstboot | `d162c70` | Wire RunFirstBoot into Build, default email=dev, admin.key.firstboot |

## Current state

- **Bootstrap surface:** 3 flags only — `--no-prompt` (skip first-boot browser open), `--proxy-upstream`, plus globals `--token`/`--url`/`--verbose`. NO `--config`, NO `--dev`, NO `--db-path`.
- **First-boot UX:** `./shark serve` → if TTY → interactive prompt; if non-TTY → silent defaults. Generates secret + JWT signing key + admin API key. Prints masked key + setup URL to stdout. Writes FULL admin key to `<db_dir>/admin.key.firstboot` (perms 0600) for one-time scripted pickup.
- **Setup page:** `/admin/setup?token=<setup-token>` shows admin key once + admin email form + magic-link wait.
- **Mode:** `shark mode dev/prod` swaps DB via state-file write + restart. Full hot-drain (atomic store pointer, drain middleware) deferred — graceful-restart fallback in place.
- **Reset:** `shark reset dev|prod|key`. Prod requires typed phrase + admin token. Dashboard buttons in Settings → Danger Zone.
- **Email:** `email.provider=dev` defaulted on first boot. Operators flip to resend/smtp from Settings → Email Delivery (reversible via `previous_provider`).
- **CORS:** runtime toggle `system_config.server.cors_relaxed`.

## Known gaps / followups

1. **Hot-drain still graceful-restart** — true atomic-pointer swap deferred (would need threading `atomic.Pointer[Store]` through all 50+ handlers). Document as Phase J if/when needed.
2. **OS-keyring secrets** — current threat model accepts plaintext-on-disk-perms-0600. Add KEK derivation if external secret manager required.
3. **Multi-instance config cache busting** — single-instance only. Add `system_config_version` column + cache invalidation if HA needed.
4. **Pre-existing test failures unrelated to W17** — `TestAdminConfigShape`, `TestHandleHostedAssets`, etc. Lived before W17, not regressed.
5. **`internal/admin/dist/`** — rebuilt at build time. After H/I, may need fresh `npm run build` if frontend changes land.

## How to build

Windows:
```
make -f Makefile.windows
```

Or manually:
```
cd admin && npm run build && cd ..
go build -trimpath -ldflags="-s -w" -o shark.exe ./cmd/shark
```

## How to run smoke

```
cd tests/smoke
cp ../../shark.exe .
rm -f shark.db dev.db server.log admin.key.firstboot
python -m pytest --tb=line
```

Conftest spawns server with `--no-prompt`, parses admin key from `admin.key.firstboot`.

## Files of note

- `YAML_DEPRECATION_PLAN.md` — original plan (deleted in cleanup; see git history `git show 6d75068`)
- `internal/server/firstboot.go` — first-boot detection + secret generation
- `internal/api/setup_handlers.go` — `/admin/setup` endpoints
- `internal/config/bootstrap.go` + `runtime.go` — config layer split
- `internal/storage/system_config.go` + `secrets.go` — DB store
- `cmd/shark/cmd/mode.go` + `reset.go` — admin commands
- `cmd/shark/cmd/cli.go` — `shark cli` discoverability
- `internal/cli/branding.go` + `logger.go` — pretty terminal
- `Makefile.windows` — Windows build pipeline

## Context for next session

If continuing W17 work: the user wants pytest run from orchestrator only (it kills other shark processes). Verify subagent commits include implementation files (not just tests) — Phase B nearly shipped without firstboot.go. Use worktrees for parallel agents but one merge target at a time.
