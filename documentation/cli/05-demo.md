# Demo Commands

The `shark demo` group provides runnable reference flows that exercise the full SharkAuth feature surface. Demos are self-contained — they provision synthetic resources, run the flow, and clean up. They are the canonical "show, don't tell" tool for screencasts and investor demos.

---

## `shark demo delegation-with-trace`

Run a 3-hop delegation chain demo with DPoP-bound tokens at each hop, plus a vault token retrieval on hop 4. Produces a self-contained HTML report.

### Synopsis
```bash
shark demo delegation-with-trace [flags]
```

### What it does

1. Registers a synthetic user and 3 agents: `user-proxy`, `email-service`, `followup-service`.
2. Configures `may_act` delegation policies between them.
3. Runs a token-exchange chain across all 3 hops, generating DPoP proofs at each step.
4. Fetches a vault token on hop 4.
5. Assembles a **self-contained HTML report** with full request/response traces, timing, and explanation.
6. Auto-opens the report in the default browser (unless `--no-open`).
7. Cleans up provisioned resources (unless `--keep`).

### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--base-url` | string | `http://localhost:8080` | Shark server base URL |
| `--admin-key` | string | _(env)_ | Admin API key (or `SHARK_ADMIN_KEY`) |
| `--output` | string | `./demo-report.html` | Path to write the HTML report |
| `--html` | string | _(none)_ | Alias for `--output` (deprecated — use `--output`) |
| `--plain` | bool | `false` | Force plain stdout output only; skip HTML generation |
| `--no-open` | bool | `false` | Do not auto-open the report in a browser |
| `--keep` | bool | `false` | Keep temp DB for inspection after the run |

Admin key resolution order: `--admin-key` flag → `SHARK_ADMIN_KEY` environment variable. The command exits with an error if neither is set.

### Examples

```bash
# Quickstart — server must be running at localhost:8080
shark demo delegation-with-trace --admin-key sk_admin_...

# Use environment variable (recommended for screencasts)
export SHARK_ADMIN_KEY=$(cat admin.key.firstboot)
shark demo delegation-with-trace

# Write report to custom path, don't open browser
shark demo delegation-with-trace \
  --admin-key $SHARK_ADMIN_KEY \
  --output ./traces/delegation-$(date +%Y%m%d).html \
  --no-open

# Remote server
shark demo delegation-with-trace \
  --base-url https://auth.example.com \
  --admin-key $SHARK_ADMIN_KEY

# Keep temp DB for post-mortem
shark demo delegation-with-trace \
  --admin-key $SHARK_ADMIN_KEY \
  --keep
```

### HTML Report

The output file is a **self-contained HTML document** — all assets are inlined. It can be:

- Opened directly in any browser (`file://`)
- Attached to emails or Slack messages
- Hosted on any static server
- Used as a visual aid during live screencasts (no server connection required after generation)

The report contains:
- A step-by-step timeline of all hops
- Full HTTP request/response traces (headers, bodies, tokens)
- Decoded JWT payloads at each hop
- DPoP proof details
- Vault retrieval trace
- Timing breakdown per step

### Gotchas

- The demo requires a **running shark server**. Run `shark serve` in another terminal before executing.
- The synthetic resources (user, agents) are created fresh each run. If `--keep` is not set, they are deleted after the run.
- The `--html` flag is a deprecated alias for `--output` and may be removed in a future version.
- Device flow commands (OAuth device authorization) are disabled in v0.1 and are not exercised by this demo.
