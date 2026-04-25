# devinbox.go

**Path:** `internal/email/devinbox.go`  
**Package:** `email`  
**LOC:** 62  
**Tests:** `devinbox_test.go`

## Purpose
Dev-mode email capture: persists outbound emails to dev_emails DB table for dashboard rendering. Extracts and logs links (magic links, password resets) to stdout for CLI-only developers.

## Key types / functions
- `DevInboxSender` (struct, line 19) — holds store reference
- `NewDevInboxSender(store)` (func, line 24) — constructor
- `Send(msg *Message)` (func, line 29) — persists to dev_emails, logs to stderr
- `extractLink(body)` (func, line 54) — scans body for first http(s) URL

## Imports of note
- `gonanoid` — generate dev email IDs
- `time` — RFC3339 timestamps
- `internal/storage` — DevEmail persist

## Wired by
- `server.Build()` selects DevInboxSender when opts.DevMode=true
- Can be injected via opts.EmailSenderOverride for tests

## Notes
- Email ID: `de_` + nanoid
- Links extracted as convenience for stdout logging (magic link URLs, password reset tokens)
- slog.Info level; visible in dev logs
- Non-blocking; persist errors returned but don't halt auth flow
- Never used in production (DevMode is CLI-only flag, not YAML)

