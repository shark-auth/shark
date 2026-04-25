# shark_email.go

**Path:** `internal/email/shark_email.go`  
**Package:** `email`  
**LOC:** 34  
**Tests:** `shark_email_test.go`

## Purpose
Placeholder SharkEmailSender for Phase 2 wiring. Fails loudly instead of silently dropping when email.provider="shark" before the relay service goes live.

## Key types / functions
- `SharkEmailSender` (struct, line 13) — apiKey, from, fromName (fields stored, not used yet)
- `NewSharkEmailSender(apiKey, from, fromName)` (func, line 21) — constructor
- `Send(msg *Message)` (func, line 29) — returns explanatory error + slog.Warn

## Imports of note
- `fmt`, `slog` — error/warning output

## Wired by
- `server.Build()` selects SharkEmailSender when cfg.Email.Provider="shark"
- Config path works end-to-end for operators who pre-configure the provider

## Notes
- Intentionally fails rather than silently dropping to signal misconfiguration
- Logs warning with documentation link: https://sharkauth.com/docs/email#shark-email
- Relay service not live (Phase 2 ships wiring only)
- Operators must switch to "resend" or "smtp" until relay launches

