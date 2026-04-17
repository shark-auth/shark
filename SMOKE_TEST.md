# Shark Phase 3 Smoke Test

Post-build verification. Exercises every Phase 3 feature against a real running binary.

## Prereqs

- `jq`, `curl`, `sqlite3` in `$PATH`
- Binary built: `go build -ldflags="-s -w" -trimpath -o shark ./cmd/shark`
- No server running on `:8080`
- Fresh DB (script handles this)

## Run

```bash
./smoke_test.sh
```

Exits 0 on all-pass, non-zero on first failure. Colored PASS/FAIL per section.

## What it covers

| # | Section | Verifies |
|---|---------|----------|
| 1 | Bootstrap | First-boot banners: admin API key + default app |
| 2 | JWT at signup | Response includes `token`; RS256; cookie set |
| 3 | Dual-accept middleware | Bearer /me 200, cookie /me 200, both 200, garbage Bearer + valid cookie → 401 (no-fallthrough) |
| 4 | JWKS endpoint | `/.well-known/jwks.json` returns 1 key; RSA/RS256/use=sig; kid matches token header |
| 5 | Revocation | `POST /api/v1/auth/revoke` → 200; under `check_per_request=true`, /me → 401 |
| 6 | Admin revoke | `POST /api/v1/admin/auth/revoke-jti` admin-gated; non-admin → 401 |
| 7 | Key rotation | `shark keys generate-jwt --rotate`; JWKS returns 2 keys; old token still validates |
| 8 | Apps CLI | `shark app create/list/show/rotate-secret/delete`; default-delete refused |
| 9 | Admin apps HTTP | POST/GET/PATCH/DELETE /api/v1/admin/apps/*; secret shown once |
| 10 | Redirect allowlist | Magic-link with allowed URL → 302; not allowlisted → 400; `javascript:` → 400; fragment → 400 |
| 11 | Org RBAC | Create org seeds 3 builtins; custom role grants access; revoke removes access; builtin delete → 409 |
| 12 | Audit logs | Mutations land rows in `audit_logs` with expected action strings |
| 13 | Regression | `/auth/logout`, `/healthz`, legacy cookie path |

## Notes

- Random suffix in emails allows repeat runs.
- Each test has `fail "<reason>"` on unexpected output.
- Revoke-check test edits sharkauth.yaml + restarts server — intrusive.
- No OAuth provider stub — redirect test uses magic-link path (same validator, §4.4 in PHASE3.md).

## When to run

- Before every release tag
- After any touch to `internal/auth/jwt`, `internal/api/middleware/auth.go`, `internal/auth/redirect`, `internal/rbac`, `internal/storage/applications.go`, `internal/server/server.go`
- After Phase 4+ merges

## Known gaps

- No external JWKS interop (use jwt.io + foreign-lang resource server — see newtests.md §6)
- No concurrent-boot race test
- No clock-skew / expired-token test (would need time travel)
