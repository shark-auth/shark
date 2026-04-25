# resend.go

**Path:** `internal/email/resend.go`  
**Package:** `email`  
**LOC:** 80  
**Tests:** `resend_test.go`

## Purpose
Transactional email via Resend HTTP API. Wraps HTTP POST with JSON marshaling and error handling.

## Key types / functions
- `ResendSender` (struct, line 17) — apiKey, from, fromName, http.Client
- `NewResendSender(cfg)` (func, line 26) — constructor; reuses SMTPConfig (Password = API key)
- `Send(msg *Message)` (func, line 43) — POSTs to api.resend.com/emails

## Imports of note
- `bytes`, `encoding/json` — request marshaling
- `net/http` — HTTP client + request
- `io` — response body read
- `internal/config` — SMTPConfig (reused for API key)

## Wired by
- `server.Build()` selects ResendSender when cfg.Email.Provider="resend"
- API handlers call Send() for all transactional mail

## Notes
- Endpoint: https://api.resend.com/emails (hardcoded)
- Auth: Bearer token (apiKey header)
- From header: email only, or "Name <email>" if FromName set
- To is wrapped in []string (single recipient per Resend call)
- Timeout: 10s per request
- HTTP >=400 status unpacked into error message for debugging

