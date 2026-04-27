# Identity Management Commands

Commands for managing users, agent identities, API keys, and OAuth consents.

All commands require a running server and admin credentials:
```bash
export SHARK_ADMIN_TOKEN=$(cat admin.key.firstboot)
export SHARK_URL=http://localhost:8080
```

---

## User Commands (`shark user`)

### `shark user list`

List all users.

#### Synopsis
```bash
shark user list [flags]
```

#### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--limit` | int | `50` | Maximum number of users to return |
| `--search` | string | _(none)_ | Filter by email or name (substring match) |
| `--json` | bool | `false` | Emit JSON array |

#### Examples

```bash
# List first 50 users
shark user list

# Search by email prefix
shark user list --search alice

# All users as JSON, pipe to jq
shark user list --json | jq '.[].email'

# Increase limit
shark user list --limit 200
```

---

### `shark user create`

Create a new user.

#### Synopsis
```bash
shark user create --email <email> [flags]
```

#### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--email` | string | _(required)_ | User email address |
| `--name` | string | _(none)_ | Display name |
| `--password` | string | _(none)_ | Initial password (optional; user can set via magic link otherwise) |
| `--json` | bool | `false` | Emit created user as JSON |

#### Examples

```bash
# Minimal creation
shark user create --email alice@example.com

# With display name and password
shark user create --email bob@example.com --name "Bob Smith" --password s3cr3t

# Capture new user ID
ID=$(shark user create --email dev@example.com --json | jq -r '.data.id')
```

---

### `shark user show`

Show a single user by ID.

#### Synopsis
```bash
shark user show <id> [flags]
```

#### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--json` | bool | `false` | Emit user object as JSON |

#### Examples

```bash
shark user show usr_abc123

shark user show usr_abc123 --json | jq .data
```

---

### `shark user update`

Update a user's fields (partial update — only changed flags are sent).

#### Synopsis
```bash
shark user update <id> [flags]
```

#### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--name` | string | _(none)_ | New display name |
| `--email` | string | _(none)_ | New email address |
| `--email-verified` | bool | `false` | Set email-verified state |
| `--json` | bool | `false` | Emit updated user as JSON |

#### Examples

```bash
# Rename user
shark user update usr_abc123 --name "Alice A."

# Mark email as verified
shark user update usr_abc123 --email-verified

# Change email
shark user update usr_abc123 --email newalice@example.com --json
```

#### Gotchas

- At least one flag must be provided; the command errors if nothing changes.

---

### `shark user delete`

Delete a user (destructive — irreversible).

#### Synopsis
```bash
shark user delete <id> [flags]
```

#### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--yes` | bool | `false` | Skip interactive confirmation prompt |
| `--json` | bool | `false` | Emit `{"deleted":"<id>"}` |

#### Examples

```bash
# Interactive prompt
shark user delete usr_abc123

# Skip prompt (scripts / CI)
shark user delete usr_abc123 --yes

shark user delete usr_abc123 --yes --json
```

---

## Agent Commands (`shark agent`)

Agents are non-human identities (services, bots, workers) that authenticate via OAuth client credentials or DPoP-bound tokens.

### `shark agent register`

Register a new agent identity. The client secret is shown only once.

#### Synopsis
```bash
shark agent register --name <name> [flags]
```

#### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--name` | string | _(required)_ | Human-readable agent name |
| `--description` | string | _(none)_ | Optional description |
| `--app` | string | _(none)_ | App slug to associate this agent with (stored in metadata) |
| `--json` | bool | `false` | Emit created agent as JSON |

#### Examples

```bash
# Register an agent
shark agent register --name "email-service"

# With description and app association
shark agent register --name "followup-service" \
  --description "Sends follow-up emails" \
  --app my-saas

# Capture credentials
CREDS=$(shark agent register --name "worker" --json)
CLIENT_ID=$(echo $CREDS | jq -r '.data.client_id')
SECRET=$(echo $CREDS | jq -r '.data.client_secret')
```

#### Gotchas

- The `client_secret` is shown exactly once. Save it immediately — there is no way to retrieve it again.
- To get a new secret, use `shark agent rotate-secret <id>`.

---

### `shark agent show`

Show a single agent by ID.

#### Synopsis
```bash
shark agent show <id> [flags]
```

#### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--json` | bool | `false` | Emit agent object as JSON |

#### Examples

```bash
shark agent show agt_abc123

shark agent show agt_abc123 --json | jq .data.client_id
```

---

### `shark agent update`

Update an agent's name or description.

#### Synopsis
```bash
shark agent update <id> [flags]
```

#### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--name` | string | _(none)_ | New agent name |
| `--description` | string | _(none)_ | New description |
| `--json` | bool | `false` | Emit updated agent as JSON |

#### Examples

```bash
shark agent update agt_abc123 --name "email-service-v2"

shark agent update agt_abc123 --description "Handles transactional email" --json
```

---

### `shark agent delete`

Deactivate (soft-delete) an agent.

#### Synopsis
```bash
shark agent delete <id> [flags]
```

#### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--yes` | bool | `false` | Skip confirmation prompt |
| `--json` | bool | `false` | Emit `{"deleted":"<id>"}` |

#### Examples

```bash
shark agent delete agt_abc123

shark agent delete agt_abc123 --yes
```

---

### `shark agent rotate-secret`

Rotate an agent's client secret. The new secret is shown only once.

#### Synopsis
```bash
shark agent rotate-secret <id> [flags]
```

#### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--json` | bool | `false` | Emit `{"client_secret":"..."}` as JSON |

#### Examples

```bash
shark agent rotate-secret agt_abc123

# Capture new secret
NEW_SECRET=$(shark agent rotate-secret agt_abc123 --json | jq -r '.data.client_secret')
```

---

### `shark agent revoke-tokens`

Revoke all active OAuth tokens for an agent (force re-authentication).

#### Synopsis
```bash
shark agent revoke-tokens <id> [flags]
```

#### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--json` | bool | `false` | Emit `{"revoked":<count>}` |

#### Examples

```bash
shark agent revoke-tokens agt_abc123

shark agent revoke-tokens agt_abc123 --json
```

---

## API Key Commands (`shark api-key`)

Admin API keys authenticate CLI and programmatic access to the admin API.

### `shark api-key create`

Create a new admin API key. The key value is shown only once.

#### Synopsis
```bash
shark api-key create --name <name> [flags]
```

#### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--name` | string | _(required)_ | Name for the API key |
| `--json` | bool | `false` | Emit created key as JSON |

#### Examples

```bash
shark api-key create --name "ci-deploy"

# Capture key value
KEY=$(shark api-key create --name "staging" --json | jq -r '.data.key')
```

#### Gotchas

- The `key` value is shown exactly once at creation. Set it as `SHARK_ADMIN_TOKEN` before it scrolls off.

---

### `shark api-key list`

List all API keys (key values are never shown — prefix only).

#### Synopsis
```bash
shark api-key list [flags]
```

#### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--json` | bool | `false` | Emit JSON array |

#### Examples

```bash
shark api-key list

shark api-key list --json | jq '.[].name'
```

---

### `shark api-key revoke`

Permanently revoke an API key by ID.

#### Synopsis
```bash
shark api-key revoke <id> [flags]
```

#### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--json` | bool | `false` | Emit `{"revoked":"<id>"}` |

#### Examples

```bash
shark api-key revoke key_abc123

shark api-key revoke key_abc123 --json
```

---

### `shark api-key rotate`

Rotate an API key — generates a new value, invalidates the old one. New key shown once.

#### Synopsis
```bash
shark api-key rotate <id> [flags]
```

#### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--json` | bool | `false` | Emit `{"key":"..."}` with new value |

#### Examples

```bash
shark api-key rotate key_abc123

NEW_KEY=$(shark api-key rotate key_abc123 --json | jq -r '.data.key')
```

---

## Consent Commands (`shark consents`)

### `shark consents list`

List OAuth consents granted by users.

#### Synopsis
```bash
shark consents list [flags]
```

#### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--user` / `-u` | string | _(none)_ | Filter by user ID |
| `--json` | bool | `false` | Emit JSON array |

#### Examples

```bash
# List all consents
shark consents list

# Filter to one user
shark consents list --user usr_abc123

shark consents list --json | jq '.[].scopes'
```
