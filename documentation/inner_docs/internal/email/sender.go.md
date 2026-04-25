# sender.go

**Path:** `internal/email/sender.go`  
**Package:** `email`  
**LOC:** 15  
**Tests:** (none)

## Purpose
Email Sender interface definition — contract for all email provider implementations.

## Key types / functions
- `Message` (struct, line 4) — To, Subject, HTML, Text
- `Sender` (interface, line 13) — Send(msg *Message) error

## Imports of note
None (pure interface definitions)

## Implementations
- `SMTPSender` (internal/email/smtp.go)
- `ResendSender` (internal/email/resend.go)
- `SharkEmailSender` (internal/email/shark_email.go, placeholder)
- `DevInboxSender` (internal/email/devinbox.go, dev mode only)
- `MemoryEmailSender` (internal/testutil/email.go, test only)

## Notes
- Core abstraction; all providers implement Send(msg *Message) error
- Message carries plain-text + HTML alternatives; SMTP builder selects HTML

