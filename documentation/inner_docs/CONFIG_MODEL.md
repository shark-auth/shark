# SharkAuth Config Model

This document explains the three config layers in SharkAuth post-W17 (Phase H). YAML files are gone. See `YAML_DEPRECATION_PLAN.md` for migration history.

---

## Three layers

### 1. Bootstrap config (env vars, startup only)

A small set of values required before the DB is open. Bound from `SHARK_*` environment variables via koanf onto the `internal/config.Config` struct.

| Env var | Purpose | Required |
|---|---|---|
| `SHARK_SERVER_SECRET` | 32+ char secret — encrypts sessions + field encryption | Yes (generated on first boot if absent) |
| `SHARK_SERVER_BASE_URL` | Drives cookie Secure flag, OAuth callbacks, magic-link URLs | Yes (prompted on first boot) |
| `SHARK_SERVER_PORT` | HTTP listen port (default: 8080) | No |
| `SHARK_STORAGE_PATH` | SQLite file path (default: `./data/shark.db`) | No |
| `SHARK_EMAIL_PROVIDER` | `resend` \| `smtp` \| `shark` | No (prompted on first boot) |

These are **read once at startup** and never re-read. No file. No reload.

### 2. Runtime config (SQLite `system_config` table)

Everything else: session lifetime, password rules, CORS origins, MFA settings, branding, email provider credentials, proxy rules. Stored in `system_config` as JSON rows, mutated via:

- **Dashboard** → Settings page (`admin/src/components/settings.tsx`)
- **Admin API** → `PATCH /api/v1/admin/config` + related endpoints
- **CLI** → `shark admin config dump` to inspect; mutations via API subcommands

Changes take effect without restart (hot-reload via DB polling).

### 3. Secrets (DB + hash, never plaintext)

| Secret | Storage | Notes |
|---|---|---|
| Admin API key | SHA-256 hash in `admin_keys` table | Shown once at first boot; rotate via `shark reset key` |
| SMTP/Resend password | AES-256-GCM in `system_config` | Encrypted with `SHARK_SERVER_SECRET` |
| JWT signing keys | Encrypted in `signing_keys` table | Rotate via `shark keys generate-jwt --rotate` |
| SSO client secrets | Encrypted per-connection in `sso_connections` | Managed via Settings → SSO |

---

## First boot flow

```
./shark serve
  │
  ├─ config.Load()          → reads SHARK_* env vars
  ├─ storage.Open()         → opens/creates SQLite, runs goose migrations
  ├─ server.Build()         → checks admin_keys table
  │   └─ empty?
  │       ├─ generate secret (if SHARK_SERVER_SECRET absent)
  │       ├─ generate admin key → SHA-256 → store; print raw key to stdout
  │       └─ first-boot prompt (unless --no-prompt):
  │           ├─ open browser to admin dashboard
  │           └─ print magic-link sign-in URL
  └─ http.ListenAndServe()
```

After first boot, all config changes go through the DB. Re-running `shark serve` picks up env vars for bootstrap fields and reads everything else from SQLite.

---

## What YAML used to do (removed Phase H)

- `sharkauth.yaml` was the single config file (koanf YAML provider).
- `Config.Save(path)` wrote mutations back to disk.
- `handleImportYAMLRules` imported proxy rules from a YAML payload.
- `yamlHasLegacyProxyRules(path)` warned if old-style rules were found.

All of these are deleted. The `.gitignore` entries `sharkauth.yaml` and `sharky.yaml` remain as safety nets in case old files are present.

---

## See also

- `YAML_DEPRECATION_PLAN.md` — full 8-phase removal history
- `internal/config/config.go` — typed Config struct + env binding
- `internal/storage/system_config.go` — runtime config persistence
- `internal/api/admin_system_handlers.go` — PATCH /admin/config handler
- `documentation/api_reference/sections/admin.yaml` — API spec
