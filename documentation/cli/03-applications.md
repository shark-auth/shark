# Application Commands

Commands for managing OAuth applications registered in SharkAuth. Each application has a `client_id` and `client_secret` used in standard OAuth 2.0 flows.

All commands require a running server and admin credentials:
```bash
export SHARK_ADMIN_TOKEN=$(cat admin.key.firstboot)
export SHARK_URL=http://localhost:8080
```

---

## `shark app list`

List all OAuth applications.

### Synopsis
```bash
shark app list [flags]
```

### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--json` | bool | `false` | Emit JSON response |

### Examples

```bash
shark app list

# Machine-readable
shark app list --json | jq '.apps[].name'

# Count registered apps
shark app list --json | jq '.apps | length'
```

---

## `shark app create`

Create a new OAuth application. The `client_secret` is shown only once.

### Synopsis
```bash
shark app create --name <name> [flags]
```

### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--name` | string | _(required)_ | Application name |
| `--callback` | string[] | _(none)_ | Allowed callback URL (repeatable) |
| `--logout` | string[] | _(none)_ | Allowed logout URL (repeatable) |
| `--origin` | string[] | _(none)_ | Allowed CORS origin (repeatable) |
| `--json` | bool | `false` | Emit created app as JSON |

URL validation rejects dangerous schemes (`javascript:`, `file:`, `data:`, `vbscript:`). Both `https://` and custom mobile app schemes (e.g. `myapp://`) are accepted.

### Examples

```bash
# Minimal web app
shark app create --name "My SaaS"

# With OAuth redirect URLs
shark app create \
  --name "My SaaS" \
  --callback https://app.example.com/callback \
  --logout https://app.example.com/logout \
  --origin https://app.example.com

# Multiple callbacks (e.g. staging + production)
shark app create \
  --name "My SaaS" \
  --callback https://app.example.com/callback \
  --callback https://staging.example.com/callback

# Mobile app with custom scheme
shark app create \
  --name "Mobile App" \
  --callback myapp://auth/callback

# Capture client credentials
CREDS=$(shark app create --name "CI App" --json)
CLIENT_ID=$(echo $CREDS | jq -r '.client_id')
SECRET=$(echo $CREDS | jq -r '.client_secret')
```

### Gotchas

- `client_secret` is shown exactly once. Store it immediately.
- The default application (created on first boot) cannot be deleted.

---

## `shark app show`

Show details of a single application. Accepts either the internal `id` or the `client_id`.

### Synopsis
```bash
shark app show <id-or-client-id> [flags]
```

### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--json` | bool | `false` | Emit full application object as JSON |

### Examples

```bash
shark app show app_abc123

# By client_id
shark app show "client_id_here"

# Inspect allowed callbacks
shark app show app_abc123 --json | jq '.allowed_callback_urls'
```

---

## `shark app update`

Update an application's name and/or allowed URLs. Use `--add-*` and `--remove-*` to mutate URL lists rather than replacing them wholesale.

### Synopsis
```bash
shark app update <id-or-client-id> [flags]
```

### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--name` | string | _(none)_ | New application name |
| `--add-callback` | string[] | _(none)_ | Add a callback URL |
| `--remove-callback` | string[] | _(none)_ | Remove a callback URL |
| `--add-logout` | string[] | _(none)_ | Add a logout URL |
| `--remove-logout` | string[] | _(none)_ | Remove a logout URL |
| `--add-origin` | string[] | _(none)_ | Add an origin |
| `--remove-origin` | string[] | _(none)_ | Remove an origin |

### Examples

```bash
# Rename
shark app update app_abc123 --name "My SaaS v2"

# Add a new staging callback
shark app update app_abc123 --add-callback https://staging.example.com/callback

# Remove old URL and add new one
shark app update app_abc123 \
  --remove-callback https://old.example.com/callback \
  --add-callback https://new.example.com/callback

# Add a CORS origin
shark app update app_abc123 --add-origin https://app.example.com
```

### Gotchas

- The command fetches the current application state first, then applies mutations client-side, then sends a full PATCH. This means concurrent updates could race — serialize updates to the same application.
- URL validation (same rules as `app create`) runs on all `--add-*` values.

---

## `shark app delete`

Delete an application (destructive — irreversible).

### Synopsis
```bash
shark app delete <id-or-client-id> [flags]
```

### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--yes` | bool | `false` | Skip confirmation prompt |

### Examples

```bash
# Interactive prompt (shows app name)
shark app delete app_abc123

# Skip prompt
shark app delete app_abc123 --yes
```

### Gotchas

- The default application (`is_default: true`) cannot be deleted. The command exits with an error if attempted.

---

## `shark app rotate-secret`

Generate a new `client_secret` for an application. The old secret is immediately invalid.

### Synopsis
```bash
shark app rotate-secret <id-or-client-id> [flags]
```

### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--json` | bool | `false` | Emit `{"client_id":"...","secret":"...","rotated_at":"..."}` |

### Examples

```bash
shark app rotate-secret app_abc123

# Capture new secret
NEW_SECRET=$(shark app rotate-secret app_abc123 --json | jq -r '.secret')

# Rotate by client_id
shark app rotate-secret "existing_client_id_here" --json
```

### Gotchas

- The new `client_secret` is shown exactly once. Update all consumers of the old secret before they next authenticate.
- The `rotated_at` timestamp is included in the JSON response for audit purposes.
