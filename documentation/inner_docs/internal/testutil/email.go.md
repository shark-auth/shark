# email.go

**Path:** `internal/testutil/email.go`
**Package:** `testutil`
**LOC:** 64
**Tests:** none (test helper).

## Purpose
In-memory implementation of `email.Sender` that captures sent messages so tests can assert on subject, body, and recipient without standing up an SMTP server. Test helper — **not for production runtime.**

## Type
- `MemoryEmailSender struct { mu sync.Mutex; Messages []*email.Message }` (line 11)

## Methods
- `NewMemoryEmailSender()` (line 17)
- `Send(msg)` (line 22) — captures instead of sending
- `LastMessage()` (line 30) — most recent or nil
- `MessageCount()` (line 40)
- `Reset()` (line 47) — clears captured messages
- `MessagesTo(to)` (line 54) — filter by recipient address

## Imports of note
- `sync`
- `internal/email`

## Used by
- `internal/api` handler tests asserting "magic link email got sent"
- `internal/auth` tests for verify-email flows

## Notes
- Test helper — **not for production runtime.**
- All methods take the mutex so tests can assert from a different goroutine than the one that triggered `Send`.
