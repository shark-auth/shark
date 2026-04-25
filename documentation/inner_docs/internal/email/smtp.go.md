# smtp.go

**Path:** `internal/email/smtp.go`  
**Package:** `email`  
**LOC:** 142  
**Tests:** `smtp_test.go`

## Purpose
Production SMTP email sender with STARTTLS (port 587) and implicit TLS (port 465) support, plaintext auth, and multipart MIME body formatting.

## Key types / functions
- `SMTPSender` (struct, line 32) — host, port, username, password, from, fromName
- `NewSMTPSender(cfg)` (func, line 42) — constructor from config.SMTPConfig
- `Send(msg *Message)` (func, line 54) — connects, auth, sends email
- `plainAuthNoTLSCheck` (struct, line 18) — custom auth for implicit TLS (port 465)

## Imports of note
- `crypto/tls` — TLS dial and config
- `net/smtp` — SMTP client
- `net` — JoinHostPort
- `internal/config` — SMTPConfig struct

## Wired by
- `server.Build()` checks cfg.SMTP and constructs SMTPSender when smtp.host set
- `internal/api` handlers call Sender.Send() for password reset, magic link, verify email

## Notes
- Port 465: implicit TLS via tls.Dial(); plainAuthNoTLSCheck bypasses Go's PlainAuth TLS check
- Port 587: StartTLS after MAIL command
- From header: email only, or "Name <email>" if FromName set
- Body: multipart; sends HTML variant
- Errors: connection, auth, MAIL/RCPT/DATA commands all returned to caller
- No retry logic; caller (webhook + API) may retry

