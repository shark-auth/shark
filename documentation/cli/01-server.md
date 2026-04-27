# Server Lifecycle Commands

Commands for starting the server, checking its health, running diagnostics, and managing the active mode.

---

## `shark serve`

Start the SharkAuth HTTP server (admin API, dashboard, OAuth endpoints).

### Synopsis

```bash
shark serve [flags]
```

### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--proxy-upstream` | string | _(none)_ | Mount a reverse proxy to this upstream URL (e.g. `http://localhost:3000`) |
| `--no-prompt` | bool | `false` | Skip the first-boot browser-open prompt (for CI / headless environments) |
| `--verbose` / `-v` | bool | `false` | Enable debug-level logging to stderr |

### First-boot behavior (F4)

On the very first run against an empty database, `shark serve`:

1. Runs all pending database migrations automatically.
2. Generates RS256 JWT signing keys and stores them in the database.
3. Creates an initial admin API key and writes it to `admin.key.firstboot` beside the database.
4. Prints a branded header with version, binary size, and URLs.
5. Opens the dashboard in the default browser (unless `--no-prompt` is set).

On subsequent runs the re-run banner shows the admin key location and detected mode.

### Examples

```bash
# First boot — interactive, opens browser
shark serve

# With a reverse proxy (your app is at :3000)
shark serve --proxy-upstream http://localhost:3000

# Headless / CI — no browser prompt
shark serve --no-prompt

# Debug logging
shark serve --verbose
```

### Gotchas

- The server binds to the port configured in `server.port` (default 8080). Run `shark doctor` before `serve` to confirm the port is free.
- `admin.key.firstboot` is written once. If deleted, you can still find the key in the database's `api_keys` table, or run `shark reset key` to rotate.
- `SIGINT` / `SIGTERM` triggers a graceful shutdown.

---

## `shark doctor`

Run 9 self-diagnostic checks against a configured SharkAuth deployment. Introduced in F5.

### Synopsis

```bash
shark doctor [flags]
```

### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--json` | bool | `false` | Emit one JSON object per check to stdout (machine-readable) |

### Checks performed

| Check | What it verifies |
|---|---|
| `config` | Config loads without error; prints base_url, port, db path |
| `db_writability` | SQLite database opens and accepts a write transaction |
| `migrations` | All goose migrations are applied (no pending) |
| `jwt_keys` | At least one active RS256 signing key exists; warns if >90 days old |
| `port_bind` | Configured port is free (server is not already running) |
| `base_url` | `GET <base_url>/api/v1/health` returns HTTP 200 |
| `admin_key` | `admin.key.firstboot` file exists OR at least one `api_keys` row in DB |
| `smtp` | If SMTP is configured, TCP connectivity (and TLS for port 465) is verified |
| `vault` | `server.secret` is set and AES-256-GCM round-trip works |

### Exit codes

| Code | Meaning |
|---|---|
| `0` | All checks passed |
| `1` | One or more checks failed |
| `2` | Config could not be loaded (fatal — no checks run) |

### Examples

```bash
# Human-readable output
shark doctor

# Machine-readable (one JSON object per check, then a summary object)
shark doctor --json

# Use in CI — exit 1 if any check fails
shark doctor || exit 1

# Capture summary line
shark doctor 2>&1 | tail -1
```

### JSON output format

```json
{"name":"config","ok":true,"detail":"base_url=http://localhost:8080 port=8080 db=data/shark.db"}
{"name":"db_writability","ok":true,"detail":"path=data/shark.db size=2.4MB"}
...
{"summary":"9/9 checks passed","passed":9,"total":9}
```

---

## `shark health`

Ping the `/healthz` endpoint of a running shark instance.

### Synopsis

```bash
shark health [flags]
```

### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--url` | string | `http://localhost:8080` | Base URL of the running instance |
| `--json` | bool | `false` | Emit JSON response |

### Examples

```bash
# Default (localhost)
shark health

# Remote instance
shark health --url https://auth.example.com

# Machine-readable
shark health --json
```

---

## `shark version`

Print the shark binary version and the branded header.

### Synopsis

```bash
shark version
```

No flags. Prints the build-injected version string (e.g. `v0.1.0`) or `dev` if running from source.

### Examples

```bash
shark version
```

---

## `shark mode`

Get or set the active database mode (`dev` or `prod`).

### Synopsis

```bash
shark mode [dev|prod] [flags]
```

### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--json` | bool | `false` | Output as JSON `{"mode":"dev"}` |

### Behavior

- With no argument: reads and prints the current mode from `~/.shark/state`.
- With `dev` or `prod`: writes the new mode to `~/.shark/state`. Takes effect on next `shark serve`.

### Examples

```bash
# Get current mode
shark mode

# Switch to dev (takes effect on restart)
shark mode dev

# Switch to prod with JSON output
shark mode prod --json
```

---

## `shark reset`

Wipe a database or rotate the admin API key via the running server's admin API.

### Synopsis

```bash
shark reset <dev|prod|key> [flags]
```

### Arguments

| Argument | Effect |
|---|---|
| `dev` | Wipe `dev.db`, regenerate all secrets (no confirmation) |
| `prod` | Wipe `shark.db`, regenerate secrets — **requires typing** `RESET PROD` |
| `key` | Rotate the admin API key only (no data loss) |

### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--url` | string | `http://localhost:8080` | SharkAuth server base URL |
| `--key` | string | _(env)_ | Admin API key (or `SHARK_ADMIN_KEY`) |
| `--json` | bool | `false` | Print JSON response |

### Examples

```bash
# Rotate admin key only
shark reset key --key sk_admin_...

# Wipe dev DB (no prompt)
shark reset dev

# Wipe production DB (requires typing "RESET PROD" at prompt)
shark reset prod --key sk_admin_...

# JSON output
shark reset key --json
```

### Gotchas

- `reset prod` requires the confirmation phrase `RESET PROD` typed interactively. Use `--json` to parse the new key from the response.
- The new admin key is printed once in the response. Store it immediately.

---

## `shark debug decode-jwt`

Decode and pretty-print a JWT (header + payload) without verifying the signature. Local operation — no server call.

### Synopsis

```bash
shark debug decode-jwt <token> [flags]
```

### Arguments

| Argument | Description |
|---|---|
| `<token>` | The JWT string to decode (3-part base64url) |

### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--json` | bool | `false` | Emit `{"header":{...},"payload":{...}}` as JSON |

### Examples

```bash
# Human-readable
shark debug decode-jwt eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9...

# Machine-readable — pipe to jq
shark debug decode-jwt $TOKEN --json | jq .payload.sub

# Inspect expiry
shark debug decode-jwt $TOKEN --json | jq .payload.exp
```

### Gotchas

- The signature is **not verified**. This is a developer convenience tool only — do not use for security decisions.
- Requires exactly 3 dot-separated segments; exits with an error otherwise.
