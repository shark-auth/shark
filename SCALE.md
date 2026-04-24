# SharkAuth Scaling Model

**Short version:** SharkAuth v0.9.x is designed for a single-process deployment on SQLite. It runs well on a $5 VPS. Horizontal scale across multiple replicas is on the Q3 2026 roadmap and requires the work listed below.

If you are planning a multi-replica deployment today, read this document first. We would rather be honest about the boundary than surprise you in production.

---

## What works today (single replica)

- OAuth 2.1 authorization server with RFC 7591 DCR, RFC 8693 token exchange, RFC 9449 DPoP, RFC 8628 device flow, RFC 8707 resource indicators.
- Identity surface: password (argon2id), magic links, TOTP MFA with recovery codes, passkeys (WebAuthn), social login (Google/GitHub), session management with refresh rotation, account lockout.
- Platform: organizations, RBAC, reverse proxy with circuit breaker and health checks, audit log, webhooks (HMAC-signed with retry), first-boot setup, admin dashboard, CLI.
- SQLite with WAL mode handles thousands of requests per second on a single node for typical auth workloads.

All of the above is production-grade on a single replica with no caveats.

---

## What does NOT scale across replicas in v0.9.x

Four in-memory stores live inside the binary. They work correctly in a single process; they split-brain across replicas.

### 1. DPoP replay cache

- File: `internal/oauth/dpop.go` (around lines 48-78).
- What it holds: short-lived map of seen DPoP JTI values to prevent proof replay (RFC 9449 §11.1).
- Failure mode under horizontal scale: an attacker can replay the same DPoP-proofed request against a different replica and succeed. This is a **spec violation** of RFC 9449 §11.1, not just a performance issue.

### 2. Device flow rate limiter

- File: `internal/oauth/device.go` (around lines 33-55).
- What it holds: package-level map of device-flow request counts per client.
- Failure mode: rate limit multiplies by replica count. A 10 requests/min limit becomes N×10 across N replicas. Trivially bypassable.

### 3. Per-IP rate limiter

- File: `internal/api/middleware/ratelimit.go` (around lines 46-84).
- What it holds: token buckets keyed by client IP.
- Failure mode: same as #2. Limit multiplies by replica count.

### 4. Per-API-key rate limiter

- File: `internal/auth/apikey.go` (around lines 121-162).
- What it holds: token buckets keyed by API key.
- Failure mode: same as #2.

Additionally, SQLite WAL lock contention makes multiple processes writing to the same `dev.db` file unsafe across replicas on shared storage.

---

## Q3 2026 roadmap to horizontal scale

In order of landing:

1. **Atomic refresh-token rotation** (immediate). Replace the read-then-write pattern in `RotateRefreshToken` with a single `UPDATE ... WHERE revoked_at IS NULL RETURNING id` SQL statement. Closes a concurrent-refresh race that exists today even on a single replica under load.
2. **PostgreSQL support.** Add a Postgres driver alongside the existing SQLite driver. Migration runner already uses goose, which supports both. Estimated scope: 2-3 weeks.
3. **Shared-state cache for in-memory stores #1-#4.** Either (a) move to Postgres tables with time-windowed indexes, or (b) add an optional Redis driver. Decision deferred until partner conversations clarify whether Redis is welcome in the deployment story. Leaning (a) for zero-extra-dependency OSS story.
4. **Circuit-breaker state for the reverse proxy.** Currently per-instance in `internal/proxy/circuit.go`. Needs the same shared-state treatment.
5. **Kubernetes Helm chart.** Unlocks horizontal pod autoscaling with the above in place.

---

## Key-at-rest model

SharkAuth signs JWTs with an ES256 key generated at first boot. The private key is encrypted at rest with AES-GCM using a key derived from `SHARKAUTH_SERVER_SECRET` (env var, loaded by koanf). No KMS integration today.

This is acceptable for the current deployment model (single binary, operator-managed env) but does not meet enterprise-KMS requirements. KMS integration (AWS KMS, GCP KMS, HashiCorp Vault) is planned alongside the hosted-cloud tier in Q3 2026.

If your threat model includes "attacker dumps the binary memory or reads the env," plan to rotate `SHARKAUTH_SERVER_SECRET` and re-encrypt the signing key accordingly.

---

## Reporting scale issues

If you are running SharkAuth at a scale where any of the boundaries above bite, open an issue on GitHub tagged `scale` with your deployment shape (replica count, request rate, which limiter is multiplying). Real-world data shapes the roadmap prioritization.
