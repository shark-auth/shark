# YAML Deprecation Plan

**Goal:** Source of truth = SQLite (runtime config) + env/flags (bootstrap-only) + Dashboard/CLI (mutation surfaces). `sharkauth.yaml` removed entirely.

**Scope:** server config, proxy rules import, app/key/init CLI commands, dashboard UI strings, docs.

---

## 1. Inventory (current state, 2026-04-25)

### 1.1 Backend YAML touch points

| Site | Role | Action |
|------|------|--------|
| `internal/config/config.go` | koanf load (file+env), `Config.Save()` writes YAML | Replace file provider with DB-backed provider; delete `Save()` (or keep one-shot exporter) |
| `internal/server/server.go:102` | `config.Load(opts.ConfigPath)` at boot | Replace with `config.Bootstrap(env)` + `config.LoadRuntime(store)` |
| `internal/server/server.go:112` | `yamlHasLegacyProxyRules()` warning | Move to one-shot import command |
| `internal/api/router.go:34,140` | propagates `ConfigPath` to handlers | Drop field |
| `internal/api/admin_system_handlers.go:706-708` | PATCH `/admin/config` → `cfg.Save(ConfigPath)` | Replace with `store.SetConfig(cfg)` |
| `internal/api/proxy_admin_v15_handlers.go:172` | POST `/admin/proxy/rules/import` accepts YAML body | Convert to one-shot CLI: `shark admin import-rules <file>` |
| `internal/api/router.go:686` | mounts `/proxy/rules/import` | Remove route |
| `cmd/shark/cmd/init.go` | writes `sharkauth.yaml` via `renderYAML()` | Rewrite: prompt → seed DB row + emit env exports |
| `cmd/shark/cmd/serve.go:59` | `--config sharkauth.yaml` flag | Replace with `--db`, `--secret`, `--base-url`, `--port` flags + env fallback |
| `cmd/shark/cmd/app_create.go,delete.go,list.go,rotate.go,show.go,update.go` | each loads `sharkauth.yaml` directly | Switch to admin API (`adminDo()` like `admin_config_dump.go`) — they already need a running server for SQLite anyway |
| `cmd/shark/cmd/keys.go` | loads YAML | Same: route via admin API |
| `internal/telemetry/ping.go:19` | comment | Update wording |

### 1.2 Frontend drift (HARD violations of `.impeccable.md` NO-YAML rule)

| File:Line | String | Fix |
|-----------|--------|-----|
| `admin/src/components/authentication.tsx:334` | "Edit `auth.jwt.mode` in sharkauth.yaml and restart" | Replace with inline JWT mode editor → PATCH `/admin/config` |
| `admin/src/components/organizations.tsx:707` | "Edit via sharkauth.yaml or API" | Replace with inline editor or remove field |
| `admin/src/components/proxy_config.tsx:422` | "Enable proxy in sharkauth.yaml" | Replace with Enable button → PATCH `/admin/config` |
| `admin/src/components/proxy_config.tsx:13,141` | code comments | Update to "DB-backed; was YAML pre-W17" |
| `admin/src/components/settings.tsx:12` | code comment | Already correct |

### 1.3 Docs that still position YAML as source of truth

- `documentation/inner_docs/ARCHITECTURE.md` — "Run with production config: `./shark serve --config /etc/sharkauth.yaml`"
- `documentation/inner_docs/cmd/shark/cmd/init.go.md` — describes YAML write
- `documentation/inner_docs/cmd/shark/cmd/serve.go.md`
- `documentation/inner_docs/internal/config/config.go.md`
- `documentation/inner_docs/internal/server/server.go.md`
- `documentation/inner_docs/DISTRIBUTION_PLAN.md`
- 6× proxy docs reference YAML rules
- `README.md` (top-level) — likely shows yaml example
- `documentation/inner_docs/cmd/shark/cmd/keys.go.md`, `proxy_admin.go.md`

### 1.4 Reconciliation gaps already noted

- `.impeccable.md` already declares NO-YAML — frontend strings above contradict it.
- `internal/config/config.go:65` comment references `docs/proxy_v1_5/migration/yaml_deprecation.md` — file does NOT exist. Either create it as part of W17 or fix the comment.

---

## 2. Bootstrap vs Runtime split

Some config is required BEFORE SQLite opens. Cannot live in DB. Solution: two layers.

**Bootstrap layer** (env vars + flags only — no file):
- `SHARK_SECRET` — AES+HMAC session key (auto-generated if absent in dev mode)
- `SHARK_BASE_URL` — public URL
- `SHARK_DB_PATH` — SQLite path (default `./shark.db`)
- `SHARK_PORT` — default 8080
- `SHARK_DEV_MODE` — bool

**Runtime layer** (SQLite `system_config` table — JSON blob, single row):
- auth.* (session_lifetime, password_min_length, jwt.*)
- passkeys.* / magic_link.* / password_reset.*
- email.* / smtp.*
- mfa.* / social.* / sso.* / oauth_server.* / api_keys.*
- audit.* / proxy.* / telemetry.*

This matches the existing `Config` struct minus the bootstrap subset.

---

## 3. Phased rollout

### Phase A — DB-backed config store (server, no UI change)

1. Migration `NNN_system_config.sql`:
   ```sql
   CREATE TABLE system_config (
     id INTEGER PRIMARY KEY CHECK (id = 1),
     payload TEXT NOT NULL, -- JSON
     updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
   );
   INSERT OR IGNORE INTO system_config (id, payload) VALUES (1, '{}');
   ```
2. `internal/storage/system_config.go` — `GetConfig() (*Config, error)`, `SetConfig(*Config) error`. JSON encode/decode.
3. `internal/config/bootstrap.go` — `Bootstrap()` reads only env/flags, returns minimal `BootstrapConfig{Secret, BaseURL, DBPath, Port, DevMode}`.
4. `internal/config/runtime.go` — `LoadRuntime(store)` reads JSON blob, returns full `Config` (merging bootstrap defaults for non-overridable fields).
5. `internal/server/server.go` — replace `config.Load(opts.ConfigPath)` with `Bootstrap()` + (post-DB-open) `LoadRuntime(store)`.
6. `internal/api/admin_system_handlers.go` PATCH /admin/config — write via `store.SetConfig(cfg)` instead of `cfg.Save()`.
7. **Keep YAML loader behind `--import-yaml <path>` one-shot flag**: reads, writes to DB, exits 0. This is the migration path for existing deployments.
8. Smoke + unit tests confirm: PATCH /admin/config persists across restart with NO yaml file present.

**Backwards compat (Phase A only):** if `--config` flag passed AND `system_config` row is `{}`, auto-import on first boot, log warning.

### Phase B — CLI surface migration

1. Rewrite `shark init`:
   - Prompt for base_url
   - Generate secret
   - Print env exports (`export SHARK_SECRET=...; export SHARK_BASE_URL=...`)
   - Optionally write a tiny `.env` file (gitignored) — NOT yaml
   - Open DB, seed `system_config` row with sane defaults
   - Print "next: `shark serve`"
2. `shark app create/delete/list/rotate/show/update` — switch from YAML loader to `adminDo()` (the pattern already used by `admin_config_dump.go`). Requires `--url` + `--token` like other admin commands. Drop `--config` flag.
3. `shark keys` — same conversion to admin API.
4. New: `shark admin import-rules <file.yaml>` — one-shot for legacy proxy YAML rules. Removes the runtime POST `/admin/proxy/rules/import` route.
5. New: `shark admin export-config [--format json]` — operator audit (NOT for re-import; round-trip not supported).
6. Remove `--config` flag from `serve` command (replace with `--db`/`--secret`/`--base-url`/`--port`).

### Phase C — Dashboard alignment

1. Fix `authentication.tsx:334` — replace YAML-edit instruction with inline JWT mode editor (it's already in `identity_hub.tsx`; this page is the read-only mirror — switch the string to "Edit on Identity Hub").
2. Fix `organizations.tsx:707` — replace with inline editor or delete the field.
3. Fix `proxy_config.tsx:422` — replace "Enable proxy in sharkauth.yaml" with an Enable button (PATCH `/admin/config` with `proxy.enabled=true`).
4. Sweep all frontend strings: `grep -rn 'sharkauth\.yaml\|YAML' admin/src/` → expect 0 user-visible matches (comments OK only if accurate).
5. Verify no UI exposes "view as YAML" toggle, drag-drop YAML import, or YAML preview anywhere (already removed; confirm).

### Phase D — Doc + binary cleanup

1. Update `ARCHITECTURE.md`: remove `--config /etc/sharkauth.yaml` example. New canonical: `SHARK_SECRET=... SHARK_BASE_URL=... shark serve`.
2. Update inner_docs touched in §1.3.
3. Top-level `README.md` — replace yaml snippet with env-var example.
4. Create (or finally write) `documentation/inner_docs/MIGRATION_FROM_YAML.md` covering: legacy users → run `shark serve --import-yaml sharkauth.yaml` once → delete file.
5. Update `.impeccable.md` rule wording: "yaml deprecated → REMOVED" once Phase E lands.

### Phase E — Hard removal (after one release with `--import-yaml` available)

1. Delete `Config.Save()`.
2. Delete `--config` flag entirely from `serve`.
3. Delete `koanf/parsers/yaml` + `koanf/providers/file` imports.
4. Delete `gopkg.in/yaml.v3` from main deps (still needed by openapi tooling? check go.mod).
5. Delete `internal/api/proxy_admin_v15_handlers.go` `handleImportYAMLRules` + route.
6. Delete `yamlHasLegacyProxyRules` warning + `internal/server/server.go:599` open-file path.
7. Drop `yaml:` tags from `Config` struct (purely cosmetic — koanf still drives env via `koanf:` tags).
8. Delete `sharkauth.yaml` + `sharky.yaml` from repo root + `.gitignore` them.
9. Update OpenAPI spec to remove `adminImportYAMLRules` operation.

---

## 4. Test gates (per phase)

- **A**: `make test` green; smoke green with `SHARK_DB_PATH=:memory: ./shark serve` (no yaml file); PATCH /admin/config survives restart.
- **B**: `shark init` flow tested via expect/script; `shark app create` succeeds against running server with no yaml in cwd.
- **C**: `grep -rn 'sharkauth\.yaml\|YAML' admin/src/components/` returns zero user-visible strings; manual click-through of /auth, /organizations, /proxy.
- **D**: `grep -rn 'sharkauth\.yaml' documentation/` returns only migration-doc references.
- **E**: `grep -rn 'yaml.v3\|koanf/parsers/yaml' internal/ cmd/` returns zero; full smoke green; binary 1-shot bootable from env vars only.

---

## 5. Risk register

| Risk | Mitigation |
|------|-----------|
| Existing operators have prod yaml files | Phase A keeps loader; Phase B adds one-shot import; Phase E gives a release of warning before removal |
| Secret storage moves to env — operators may not have secret manager | Document `.env` file + systemd `EnvironmentFile=` pattern in MIGRATION doc |
| `shark app/keys` CLI required server running | Acceptable: SQLite is the source of truth, so commands need DB lock anyway. Document for users running these scripts on cold servers (use admin API directly OR add `--offline-db <path>` escape hatch) |
| OpenAPI bundling step uses yaml | Unrelated to runtime config — keep yaml tooling for openapi spec only |
| Telemetry opt-out used to be `telemetry.enabled: false` in yaml | Replace with env `SHARK_TELEMETRY_DISABLED=1` + dashboard toggle |

---

## 6. Sequence summary (commit-by-commit)

1. `feat(config): DB-backed system_config store + bootstrap split`
2. `feat(config): --import-yaml one-shot bootstrap from legacy yaml`
3. `refactor(api): PATCH /admin/config writes to DB instead of file`
4. `refactor(cmd): shark init seeds DB + prints env exports`
5. `refactor(cmd): app/keys subcommands route via admin API`
6. `feat(cmd): shark admin import-rules for legacy proxy yaml`
7. `fix(admin/auth): inline JWT mode editor, drop yaml string`
8. `fix(admin/organizations): inline editor, drop yaml string`
9. `fix(admin/proxy): Enable button, drop yaml string`
10. `docs: rewrite ARCHITECTURE + inner_docs for env-driven config`
11. `docs: MIGRATION_FROM_YAML.md`
12. `chore: remove yaml loader, --config flag, sharkauth.yaml from repo`

---

## 7. Open questions (need decisions before kickoff)

1. **Secret rotation**: env-only means rotation requires restart. Acceptable, or move to DB with at-rest encryption seeded by env-key?
2. **Multi-instance deployments**: DB-backed config means all instances pick up changes via cache invalidation. Need `system_config_version` column for cache busting? Or settle for "restart instances after config change"?
3. **`.env` file**: write one in `shark init`, or just print exports? Tradeoff: convenience vs leaking secrets via `cat .env`.
4. **Backwards compat window**: how many releases between Phase A (loader still works) and Phase E (loader deleted)? Recommend 2 minor versions.
