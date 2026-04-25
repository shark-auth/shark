# dpop.go

**Path:** `internal/oauth/dpop.go`
**Package:** `oauth`
**LOC:** 428
**Tests:** `dpop_test.go`

## Purpose
RFC 9449 DPoP proof validation + in-memory JTI replay cache. Returns the JWK thumbprint (`jkt`) used for `cnf.jkt` binding.

## RFCs implemented
- RFC 9449 OAuth 2.0 DPoP
- RFC 7638 JWK Thumbprint (for `jkt`)

## Key types / functions
- `dpopWindow` (const, line 24) — 60-second iat skew window.
- `allowedDPoPAlgs` (var, line 28) — whitelist of asymmetric algs; HS* explicitly forbidden per §4.3.
- `DPoPJTICache` (struct, line 48) — `sync.Mutex`-guarded `map[string]time.Time`.
- `NewDPoPJTICache` / `MarkSeen` (line 54 / 60) — replay protection with on-demand pruning.
- `ValidateDPoPProof` (func, line 94) — full §4.3 validation: parses header, checks `typ=dpop+jwt`, verifies alg/jwk, validates signature, checks iat skew, htm method match, htu URL match, optional `ath` claim, marks JTI seen, returns thumbprint.
- `jwkToPublicKey`, `htuMatches` (later) — helpers.

## Imports of note
- `github.com/golang-jwt/jwt/v5`
- `crypto/ecdsa`, `crypto/rsa` for jwk reconstruction

## Wired by
- `internal/oauth/handlers.go:64` — HandleToken DPoP intercept.
- `internal/api/middleware` resource-server validation.

## Used by
- All DPoP-bound flows (`cnf.jkt` claim threading).

## Notes
- In-memory cache ⇒ horizontal scaling needs Redis/shared backend (see SCALE.md).
- `WithoutClaimsValidation` is intentional — iat is validated manually with the tighter 60s window.
- Symmetric algorithms blocked at line 28 — never accept HS256 even if a client tries.
