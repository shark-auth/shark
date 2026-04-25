# mfa_challenges.go

**Path:** `internal/authflow/mfa_challenges.go`  
**Package:** `authflow`  
**LOC:** 104  
**Tests:** `mfa_challenges_test.go`

## Purpose
In-memory MFA challenge store for Phase 6 flows. Issues time-limited challenge IDs, stores them, verifies TOTP code submissions. Thread-safe via mutex.

## Key types / functions
- `MFAChallenge` (struct, line 15) — UserID, FlowRunID, IssuedAt
- `ChallengeStore` (struct, line 29) — in-memory map + mutex
- `GlobalChallengeStore` (var, line 36) — package-level singleton
- `NewChallengeStore()` (func, line 39) — returns empty store
- `Issue(userID, flowRunID)` (func, line 47) — mint challenge ID, store it, sweep expired
- `Consume(challengeID, userID)` (func, line 69) — lookup, validate ownership + TTL, delete
- `Peek(challengeID)` (func, line 89) — read without consuming (test assertions)
- `sweepLocked()` (func, line 97) — remove expired entries (called lazily on Issue)

## Constants
- `challengeTTL = 5 * time.Minute` — challenge expiry window

## Imports of note
- `crypto/rand`, `encoding/hex` — random ID generation
- `sync` — mutex for thread safety
- `time` — expiry logic

## Wired by
- `require_mfa_challenge` step in authflow/steps.go calls Issue/Consume
- Global singleton used in Engine; tests inject NewChallengeStore for isolation

## Notes
- In-memory acceptable for single-instance self-host
- CLOUD.md §1: must migrate to Redis for multi-tenant Cloud fork before GA
- Challenges lost on restart; user must attempt auth again (known/documented failure mode)
- ID format: `mfac_` + 16 bytes hex
- Sweep: lazy (triggered on each Issue); expired entries removed
- Thread-safe for concurrent Issue/Consume/Peek calls

