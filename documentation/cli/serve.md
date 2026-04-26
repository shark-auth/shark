# `shark serve`

Starts the SharkAuth HTTP server with the admin API, dashboard, and OAuth
endpoints.

```
shark serve
```

## Boot output (W1.8 Surface 1)

After the branded ASCII glyph (printed by `cli.PrintHeader`), the server
prints a short post-boot banner:

```
  Dashboard   http://localhost:8080/admin
  Setup key   data/admin.key.firstboot  (one-time)
```

The "Setup key" line only appears on **first boot**, when shark just generated
a fresh admin API key. On subsequent boots the file is no longer the canonical
source — the key is now stored hashed in the database. After first boot, the
banner shows only the Dashboard URL.

The banner respects TTY detection: ANSI bold/dim codes are stripped on
non-TTY stdout (CI, log files, redirected output).

## Underlying boot phases

`internal/server/Build` runs these phases in order:

1. Load config from `system_config` table
2. Open SQLite at the resolved storage path
3. Apply migrations
4. First-boot bootstrap (generate `server.secret`, JWT signing key, admin API
   key if the secrets table is empty)
5. Validate secret length
6. Initialize JWT signer + JWKS
7. Initialize vault (if enabled)
8. Initialize email provider (dev-inbox by default)
9. Build HTTP router
10. Start telemetry ping loop (no-op if disabled)
11. Start proxy listeners (if any are configured)
12. Listen on the main port

`internal/server/bootlog.go` ships the `Phase` helper (`StartPhase` / `Done` /
`Fail`) so subsequent waves can wrap individual phases with ✓/✗ markers
without a single big-bang refactor. Heartbeat (single live status line) and
framework-log suppression are deferred to W+1.

## Flags

- `--no-prompt` — skip the first-boot "open dashboard?" prompt (CI / headless).
- `--proxy-upstream <url>` — mount the reverse proxy to this upstream URL.

## Environment overrides

- `SHARK_PORT` — override `system_config.server.port` (used by smoke tests for
  per-test port isolation).
- `SHARK_DB_PATH` — override `system_config.storage.path`.

## Smoke coverage

`tests/smoke/test_w18_boot_polish.py` asserts the Dashboard banner appears in
`server.log` after the conftest fixture spawns the server, and that boot does
not produce a Go panic.
