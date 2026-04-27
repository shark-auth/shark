# Admin Commands

Commands for runtime configuration, branding, signing key management, and instance introspection.

All commands require a running server and admin credentials:
```bash
export SHARK_ADMIN_TOKEN=$(cat admin.key.firstboot)
export SHARK_URL=http://localhost:8080
```

---

## `shark admin config dump`

Fetch and pretty-print the live admin configuration from the running server.

### Synopsis
```bash
shark admin config dump [flags]
```

### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--json` | bool | `false` | Included for consistency; output is always JSON |

The command always emits indented JSON — config is inherently structured and has no useful human-readable representation.

### Examples

```bash
shark admin config dump

# Pipe to jq for specific fields
shark admin config dump | jq .server

shark admin config dump | jq '.email.host'

# Save snapshot before a change
shark admin config dump > config-snapshot-$(date +%Y%m%d).json
```

### Gotchas

- Config is **DB-backed** (yaml was removed in W17). Changes made via the dashboard or API are reflected immediately.
- Sensitive fields (secrets, SMTP passwords) are redacted in the API response.

---

## `shark auth config show`

Show the current authentication configuration (alias of `admin config dump` scoped to auth settings).

### Synopsis
```bash
shark auth config show [flags]
```

### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--json` | bool | `false` | Emit JSON |

### Examples

```bash
shark auth config show

shark auth config show --json | jq .
```

---

## `shark branding get`

Read current branding design tokens for an app.

### Synopsis
```bash
shark branding get <slug> [flags]
```

### Arguments

| Argument | Description |
|---|---|
| `<slug>` | App slug (currently global; slug is reserved for future per-app scoping) |

### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--json` | bool | `false` | Emit JSON (output is always JSON) |

### Examples

```bash
# Read current tokens
shark branding get my-app

# Pipe to jq
shark branding get my-app | jq .design_tokens
```

---

## `shark branding set`

Set design tokens for an app. Accepts key=value pairs or a JSON file.

### Synopsis
```bash
shark branding set <slug> [flags]
```

### Arguments

| Argument | Description |
|---|---|
| `<slug>` | App slug |

### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--token` | string[] | _(none)_ | Design token as `key=value` (repeatable) |
| `--from-file` | string | _(none)_ | Path to a JSON file containing design tokens |
| `--json` | bool | `false` | Emit updated branding as JSON |

Exactly one of `--token` or `--from-file` must be provided.

### Examples

```bash
# Set individual tokens
shark branding set my-app \
  --token colors.primary=#6366f1 \
  --token typography.font_family=Inter

# Multiple tokens
shark branding set my-app \
  --token colors.primary=#6366f1 \
  --token colors.secondary=#8b5cf6 \
  --token radius.button=8px

# From a JSON file
shark branding set my-app --from-file tokens.json

# Example tokens.json
cat tokens.json
# {
#   "colors.primary": "#6366f1",
#   "typography.font_family": "Inter",
#   "radius.button": "8px"
# }

# JSON response
shark branding set my-app --token colors.primary=#6366f1 --json
```

### Gotchas

- Token keys are dot-notation paths (e.g. `colors.primary`, `typography.font_family`). The format is defined by the dashboard's design token system.
- Changes take effect immediately — no server restart required.

---

## `shark keys generate-jwt`

Rotate the active RS256 JWT signing keypair via the admin API.

### Synopsis
```bash
shark keys generate-jwt [flags]
```

### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--rotate` | bool | `false` | Accepted for compatibility; the endpoint always generates a new key |
| `--json` | bool | `false` | Emit `{"kid":"...","algorithm":"...","status":"..."}` |

### Examples

```bash
# Rotate signing key
shark keys generate-jwt

# Machine-readable
shark keys generate-jwt --json | jq .kid

# With compatibility flag
shark keys generate-jwt --rotate
```

### Gotchas

- The old key is **retired** (not deleted). Both old and new keys remain in the JWKS endpoint until `rotated_at + 2 × access_token_ttl` has elapsed, ensuring in-flight tokens remain valid.
- `shark doctor` warns if any active key is older than 90 days.
- Rotating mid-traffic is safe — the JWKS endpoint serves all non-expired keys.

---

## `shark whoami`

Introspect the current token and print identity information.

### Synopsis
```bash
shark whoami [flags]
```

### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--json` | bool | `false` | Emit identity JSON |
| `--token` | string | _(env)_ | Token to introspect (defaults to `SHARK_ADMIN_TOKEN`) |
| `--url` | string | _(env)_ | Server URL |

### Behavior

1. Calls `GET /api/v1/auth/me` with the configured token as a Bearer.
2. If that returns 401 (e.g. the token is an admin API key, not a user JWT), falls back to `GET /api/v1/admin/stats` to verify the key is valid.
3. Reports `token_type: admin_api_key` in that case.

### Examples

```bash
# Check current token
shark whoami

# Verify admin key
export SHARK_ADMIN_TOKEN=$(cat admin.key.firstboot)
shark whoami

# Machine-readable
shark whoami --json | jq '{id, email}'

# Check a specific token
shark whoami --token eyJhbGci... --json
```

---

## `shark health`

Ping the `/healthz` endpoint. Documented in full in [01-server.md](./01-server.md).

---

## `shark cli`

Print a grouped summary of all shark CLI commands to stdout.

### Synopsis
```bash
shark cli
```

No flags. Useful for a quick reference when offline.

### Examples

```bash
shark cli
```

---

## Deprecated / Removed Commands

| Command | Status | Notes |
|---|---|---|
| `shark init` | **Removed** | YAML-based init was removed in W17. All config is now DB-backed. First boot is automatic via `shark serve`. |
| Device flow commands | **Disabled** | OAuth device authorization flow is disabled in v0.1. |
