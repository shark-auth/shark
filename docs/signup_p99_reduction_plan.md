# Engineering Review & Plan: Sub-400ms Signup p99

## Executive Summary
Recent architectural updates have successfully optimized the login flow (hitting ~330 RPS with memory/cache bounds). However, the signup flow is experiencing significant p99 variance (up to 3.62s) due to burst contention. As correctly identified, the bottleneck is not merely an Argon2 hardware ceiling, but a compound issue of queueing, CPU bursts starving the Go scheduler, and serialized disk commits behind CPU tasks.

This plan details the steps to transition the system to a "two-speed identity system" by explicitly capping CPU-heavy identity creation and aggressively smoothing the database and memory overhead.

## Root Cause Analysis
Based on a codebase investigation, the following architectural issues contribute to the p99 instability:
1. **Unused Atomic Transactions:** `handleSignup` (`internal/api/auth_handlers.go`) performs multiple sequential DB operations (create user, create session, etc.) rather than leveraging the available `SignupAtomic` implementation in `internal/storage/sqlite.go`. This amplifies WAL writes and fsync pressure.
2. **Naive Semaphore for Argon2:** `internal/auth/password.go` uses a basic semaphore for concurrency control. Under burst, this causes queuing and Go scheduler starvation rather than smooth isolation.
3. **Synchronous Audit/Webhook Writes:** Webhook emission (`Dispatcher.Emit` in `internal/webhook/dispatcher.go`) and audit logs are written synchronously during the signup critical path.
4. **GC Pressure:** The request-to-user mapping phase and synchronous logging cause unnecessary object allocations during the CPU-bound Argon2 bursts.

## Phase 1: Database & Disk Amplification (High Impact, Low Effort)
**Goal:** Consolidate disk writes to minimize WAL/fsync overhead behind the CPU burst.

1. **Refactor `handleSignup` to use `SignupAtomic`:**
   - **File:** `internal/api/auth_handlers.go`
   - **Action:** Replace individual calls to `CreateUser`, `CreateSession`, and initial hooks with a single `storage.SignupAtomic` transaction. This ensures only one disk sync occurs for the entire signup process.

2. **Async Webhooks & Audit Logs:**
   - **Files:** `internal/api/auth_handlers.go`, `internal/webhook/dispatcher.go`
   - **Action:** Decouple `Dispatcher.Emit` and audit log creation from the HTTP critical path. Implement an in-memory batcher/buffer (using a background goroutine and channels) to flush deliveries asynchronously.

## Phase 2: CPU Burst Isolation & Queue Smoothing (High Impact)
**Goal:** Prevent Argon2 hashing bursts from starving the Go scheduler and queueing other requests indefinitely.

1. **Implement Dedicated Argon2 Worker Pool:**
   - **File:** `internal/auth/password.go`
   - **Action:** Replace the `Hasher.sem` semaphore with a dedicated, fixed-size worker pool (e.g., matching the number of physical cores/vCPUs). Requests should be queued with a strict timeout, failing fast (e.g., `503 Service Unavailable` or `429 Too Many Requests`) if the queue exceeds a threshold, rather than blocking until p99 hits 3.6s.

2. **Explicit Concurrency Capping for Signup:**
   - **File:** `internal/api/auth_handlers.go`
   - **Action:** Wrap the signup handler in a specific rate-limiting/smoothing middleware or token bucket that operates independently of the global CPU limiter, smoothing the inbound bursts specifically for the expensive identity creation layer.

## Phase 3: Memory & GC Optimization (Medium Impact)
**Goal:** Reduce GC spikes caused by allocations occurring in tandem with Argon2.

1. **Pre-allocate and Reuse Buffers:**
   - **File:** `internal/api/auth_handlers.go`
   - **Action:** Audit JSON decoding and object mapping. Use `sync.Pool` for reusable structs or bytes buffers during the initial payload parsing to lower GC pressure while the system is CPU-bound.
2. **Single-flighting (If Applicable):**
   - **Action:** Ensure that duplicate signup requests (e.g., rapid double-clicks for the same email) are handled via `golang.org/x/sync/singleflight` to immediately reject or coalesce duplicate CPU work.

## Success Metrics
- **Login RPS:** Maintained at ~330 RPS.
- **Signup RPS:** Maintained or slightly increased to ~100-120 RPS.
- **Signup p99:** Reduced from ~3.62s to **< 400ms**.
