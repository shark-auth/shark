# Audit Commands

Commands for exporting and working with SharkAuth audit logs.

Audit logs record all significant events in the system: user logins, agent token exchanges, admin operations, consent grants/revocations, and more.

All commands require a running server and admin credentials:
```bash
export SHARK_ADMIN_TOKEN=$(cat admin.key.firstboot)
export SHARK_URL=http://localhost:8080
```

---

## `shark audit export`

Export audit logs as CSV. Optionally filter by date range and write to a file.

### Synopsis
```bash
shark audit export [flags]
```

### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--since` | string | _(none)_ | Start date, RFC3339 or `YYYY-MM-DD` (maps to `from` in the API) |
| `--until` | string | _(none)_ | End date, RFC3339 or `YYYY-MM-DD` (maps to `to` in the API) |
| `--output` / `-o` | string | _(stdout)_ | Write CSV to this file path instead of stdout |

The server returns CSV by default. The command streams the response body directly to stdout or the output file — no buffering.

### Examples

```bash
# Export all audit logs to stdout
shark audit export

# Filter to a specific month
shark audit export --since 2026-01-01 --until 2026-01-31

# Write to file
shark audit export --since 2026-01-01 --output audit-jan.csv

# Full date range with RFC3339 timestamps
shark audit export \
  --since 2026-01-01T00:00:00Z \
  --until 2026-01-31T23:59:59Z \
  --output /tmp/audit.csv

# Pipe to csvkit for quick inspection
shark audit export --since 2026-04-01 | csvstat
```

### Output format

The server returns CSV with a header row. Typical columns:

```
id,actor_id,actor_type,action,target_id,target_type,ip_address,user_agent,created_at
```

### Gotchas

- Without `--since` / `--until`, the export includes all records. For large deployments this can be slow — always use date filters in production.
- The progress message (`audit logs exported to <file>`) is written to **stderr**, not stdout, so it does not corrupt CSV output when writing to a file.
- The endpoint is `POST /api/v1/audit-logs/export` (not GET) — the date filters are sent in the request body.

---

## Notable Behaviors

### Retention

SharkAuth does not automatically prune audit logs. Use `--since`/`--until` to export before archiving or rotating the database.

### Streaming

The response is streamed directly — the command does not buffer the full CSV in memory, making it safe for large exports.

### Integration example (cron)

```bash
#!/usr/bin/env bash
# Daily audit export script
set -euo pipefail

DATE=$(date -u +%Y-%m-%d)
YESTERDAY=$(date -u -d "yesterday" +%Y-%m-%d 2>/dev/null || date -u -v-1d +%Y-%m-%d)

shark audit export \
  --since "$YESTERDAY" \
  --until "$DATE" \
  --output "/var/log/sharkauth/audit-${YESTERDAY}.csv"

echo "Exported audit log for $YESTERDAY"
```
