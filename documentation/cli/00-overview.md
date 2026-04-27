# SharkAuth CLI — Overview

The `shark` binary is the operator's single interface for managing the SharkAuth identity server: starting the server, inspecting state, managing users, agents, applications, API keys, audit logs, and runtime configuration.

## Authentication

Most commands that call the admin API require two things:

| What | Flag | Environment variable | Default |
|---|---|---|---|
| Server URL | `--url` | `SHARK_URL` | `http://localhost:8080` |
| Admin token | `--token` | `SHARK_ADMIN_TOKEN` | _(none — required)_ |

The first-boot admin API key is written to `admin.key.firstboot` beside the database file. Set it as `SHARK_ADMIN_TOKEN` before running any management command.

```bash
export SHARK_ADMIN_TOKEN=$(cat admin.key.firstboot)
export SHARK_URL=http://localhost:8080
```

## Global Flags

These flags are available on every command:

| Flag | Default | Description |
|---|---|---|
| `--url` | `http://localhost:8080` | Base URL of the running shark instance |
| `--token` | _(env)_ | Admin API token |
| `--verbose` / `-v` | `false` | Enable debug-level logging to stderr |

## Output Modes

- **Human-readable** (default): tabular or inline key/value output suitable for terminals.
- **`--json`**: emits structured JSON to stdout. Supported on most subcommands. Errors in JSON mode are written to stderr as `{"error":"code","message":"..."}`.

## Command Pattern

```
shark <noun> <verb> [flags] [args]
```

Examples:
```bash
shark user create --email alice@example.com
shark agent register --name "email-service"
shark app list --json
shark audit export --since 2026-01-01 --output audit.csv
```

## Command Groups

| Group | Commands |
|---|---|
| **Server** | `serve`, `health`, `version`, `doctor`, `mode`, `reset` |
| **Debug** | `debug decode-jwt` |
| **Identity** | `user`, `agent`, `api-key`, `consents` |
| **Applications** | `app` |
| **Audit** | `audit` |
| **Demo** | `demo` |
| **Admin** | `admin`, `branding`, `auth`, `keys`, `whoami` |
| **Utility** | `cli`, `completion` |

## Quick Reference

```bash
# Server
shark serve                                  # start the server
shark serve --proxy-upstream http://app:3000 # start with reverse proxy
shark doctor                                 # run 9 diagnostic checks
shark health                                 # ping /healthz
shark version                                # print version
shark mode [dev|prod]                        # get/set active mode
shark reset <dev|prod|key>                   # wipe DB or rotate admin key

# Identity
shark user list
shark user create --email alice@example.com --name "Alice"
shark agent register --name "email-service"
shark api-key create --name "ci-deploy"
shark consents list --user usr_abc123

# Applications
shark app list
shark app create --name "My App" --callback https://app.example.com/callback

# Audit
shark audit export --since 2026-01-01 --output audit.csv

# Debug
shark debug decode-jwt <token>
shark whoami

# Admin
shark admin config dump
shark branding get my-app
shark keys generate-jwt
```
