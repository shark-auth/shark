# dispatcher.go

**Path:** `internal/webhook/dispatcher.go`
**Package:** `webhook`
**LOC:** 448
**Tests:** `dispatcher_test.go`

## Purpose
Durable webhook delivery: HMAC-SHA256 signing, exponential backoff retry ([1m, 5m, 30m, 2h, 12h] over 5 attempts), dead-letter failure, real-time broadcast.

## Key types / functions
- `Dispatcher` (struct, line 57) — store, worker pool (default 4), job queue, http.Client, goroutines, real-time subscribers
- `EventEnvelope` (struct, line 71) — event name, created_at timestamp, data payload
- `Dispatcher.Start()` (line 150) — launches worker goroutines + retry scheduler
- `Dispatcher.Stop()` (line 161) — waits for in-flight deliveries (bounded by 10s timeout)
- `Dispatcher.Emit()` (line 171) — records pending deliveries, enqueues immediate send, broadcasts to subscribers
- `Dispatcher.Subscribe()` (line 80) — returns channel for real-time event broadcast (buffer 100)
- `Dispatcher.Unsubscribe()` (line 89) — removes subscriber channel
- `Dispatcher.broadcast()` (line 102) — sends to all subscribers; drops if subscriber is slow
- `New()` (line 135) — constructs Dispatcher with store, options
- `WithWorkers()` (line 119) — option to set worker count
- `BackoffSchedule` (line 39) — [1m, 5m, 30m, 2h, 12h]
- `MaxAttempts` (line 49) — 6 (initial + 5 backoffs)

## Imports of note
- `crypto/hmac`, `crypto/sha256` — HMAC-SHA256 signing
- `internal/storage` — WebhookDelivery persistence
- `sync.WaitGroup` — worker pool lifecycle

## Wired by
- Server.Build() constructs and starts Dispatcher
- `internal/audit/audit.go` (SetDispatcher wires for real-time emission)
- All Emit() callsites (auth events, rule changes, etc.)

## Used by
- Real-time SSE dashboard subscriptions
- Webhook admin endpoints to test delivery
- Audit logger for real-time emission

## Notes
- Signature format: X-Shark-Signature: t=<unix_ts>,v1=<hex(hmac)> (line 12, matches Stripe shape).
- Durable-first: delivery row created before HTTP call (line 200), survives process death.
- Async: Emit() returns immediately; actual HTTP happens on worker pool (line 9).
- Worker pool size default 4, tunable via WithWorkers (line 118).
- Job queue buffered (256), prevents unbounded growth (line 139).
- Retry scheduler runs continuously, enqueues next-attempt deliveries (line 157).
- HTTP timeout 10s per attempt (line 53).
- Slow subscribers dropped (default channel non-blocking) to prevent backpressure (line 109).
