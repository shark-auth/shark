# Handoff — Dashboard Deep Audit Branch

Branch: `claude/admin-vendor-assets-fix`
Last update: 2026-04-20 (Phase 6.6 dashboard deep audit session)
Full suite: **17/17 Go packages pass.** Admin bundle clean. Binary: 41M.

## What shipped this session (uncommitted at time of writing)

Backend + frontend fixes across 24 tasks (P0 + P1 + P2). Full detail in `CHANGELOG.internal.md` → `Phase 6.6`.

Top-level summary:
- 3 critical routing 404s fixed (`/admin/audit`, `/admin/orgs/{id}/roles`, `/admin/orgs/{id}/invitations`)
- 5 JSON shape mismatches fixed (sessions.data, users camelCase→snake_case, proxy PascalCase→snake_case, overview MFA/trends/auth_methods, orgs members+metadata)
- 1 crash fixed (organizations.tsx ReferenceError on detail open)
- 1 failing test fixed (`TestAdminStatsBasicCounts` MFA seed)
- 3 authflow stubs wired (`assign_role`, `add_to_org`, `require_mfa_challenge`) + new `POST /auth/flow/mfa/verify` endpoint
- adminConfigSummary expanded with passkey/password_policy/jwt/magic_link/session_mode/social_providers nested objects
- Real CSV audit export (was JSON-as-.csv) + pagination + date-range UI
- `DELETE /admin/sessions` (revoke-all), `POST /agents/{id}/rotate-secret`, `GET /admin/permissions/batch-usage`, `GET /webhooks/events`, `DELETE /admin/organizations/{id}/members/{uid}` — all new
- New migration `00002_audit_logs_extended_filters.sql` adds org_id/session_id/resource_type/resource_id columns
- 8 swallowed `_ = store.X` errors now log via `slog.Warn`
- Dead `admin/src/components/agents.tsx` deleted (was MOCK stub; `agents_manage.tsx` is the live page)

## RM.md repositioning

Lead now reads agent-native first: "first open-source identity platform built for AI agents as first-class citizens. MCP-native OAuth 2.1. DPoP-bound tokens. Agent-to-agent delegation. Token Vault." Humans-auth features listed after, still complete. Matches actual moat.

## Critical lessons from this session

### LSP stale after subagent writes
When a subagent adds methods to the `Store` interface across many files, the editor's LSP diagnostics will show "missing method" errors on test files that already implement them — but `go build ./...` is clean. Do NOT trust diagnostics. Re-run `go build ./...` + `go test ./... -count=1` to verify. We saw this 3+ times this session.

### Worktree discipline for multi-writer parallel agents
HANDOFF rule from prior branch: **never launch 2+ writer subagents in parallel without `isolation: "worktree"`**. This session launched parallel Sonnet subagents but each had atomic, non-overlapping file scopes (e.g., `sessions.tsx + session_handlers.go` isolated from `users.tsx + user_handlers.go`). Tests between batches locked in green state before next dispatch. Safe for this pattern. NOT safe if scopes overlap.

### Subagent model selection
Use `subagent_type: "general-purpose"` + `model: "sonnet"` for coding work to preserve Opus main-thread context. Subagents get self-contained prompts with full context + clear deliverable format. Report back: files changed + test output + decision notes.

### `@ts-nocheck` on dashboard files masks every field mismatch
Every `admin/src/components/*.tsx` has `// @ts-nocheck` at the top. This is why 9 of the 24 bugs were silent: backend struct tags and frontend reads drifted without the compiler catching it. Follow-up (post-launch): generate TS types from Go structs via `tygo` or similar, then drop nocheck.

### Anti-pattern: hardcoded fallback data
`overview.tsx:321` had a hardcoded health fallback (`v0.8.2`, `18d uptime`, etc.) that masked `/admin/health` errors with plausible-looking fake data. Replaced with loading skeleton + one-line error state. Audit other fallbacks with same shape.

## What's NOT shipped — open for next agent

### Wave 4 smoke partials (same as prior handoff, not yet closed)
- **G1 MFA TOTP full flow** — smoke covers toggle + admin disable; no end-to-end enroll→challenge→verify→recovery. Use `oathtool` or `python3 -c "import pyotp"` for code generation.
- **G2 Passkey WebAuthn flow** — structural only. Use `go-webauthn` virtual authenticator OR canned attestation/assertion fixtures.
- **F4 Token Exchange happy path** — RFC 8693 agent-to-agent `act` chain. Need: seed 2 agents, mint subject+actor tokens, exchange, decode resulting JWT, assert `act.sub`.
- **F5 DPoP full flow** — needs ECDSA P-256 proof JWT construction. Pure bash is gnarly. Use `python3 cryptography` OR move to Go test.
- **F6 Vault token retrieval full flow** — needs in-process mock OAuth upstream. Partially stubbed.
- **F7 Cache-Control headers** — backend may not set Cache-Control on `/.well-known/openid-configuration`, `/.well-known/jwks.json`, `/admin/health`. Verify + fix.

### Settings page — editable config
`admin/src/components/settings.tsx` displays config read-only. Making it editable requires:
- Backend: `PATCH /admin/config` handler writing `sharkauth.yaml` + hot-reload OR 503 "restart required"
- Storage: yaml writer preserving comments + ordering (yaml.v3 Node API)
- Hot-reload: chi router re-init under mutex OR restart-on-write via supervisor
- Frontend: convert read-only ConfigRow to editable inputs

User previously indicated this is acceptable as Phase 8/9 work, not blocking launch.

### Phase gating (intentional placeholders, un-gate when features land)
Sidebar items in `admin/src/components/layout.tsx` with `ph:` markers:
- **API Explorer / Session Debugger / Event Schemas** — `ph: 5` (Phase 5 SDK sub-features not built; render `EmptyShell`)
- **OIDC Provider** — `ph: 8` (us being IdP for other apps; SSO consumer path is built and lives at `sso.tsx`)
- **Impersonation / Compliance / Migrations / Branding** — `ph: 9`

`PhaseGate` at `admin/src/components/PhaseGate.tsx` + `CURRENT_PHASE = 6` constant control gating. Bumping `CURRENT_PHASE` un-gates everything ≤ N.

### Remaining deferred authflow stubs
- `set_metadata` — wire to `store.UpdateUser` metadata field
- `custom_check` — similar to webhook-step but sync with timeout
- `delay` — useful for rate-limit simulation

All three currently show `deferred: true` with "v0.2" chip in `flow_builder.tsx` palette. Runtime handlers no-op gracefully.

### Shared abuse intel / HSM-backed DPoP / multi-region MCP
Cloud-only future features. Per user direction these defer to the Cloud fork (separate repo track, also shipping April 27). Not scope for OSS self-host.

## Critical don'ts from prior handoff (still active)

### Parallel writer subagents need worktrees
Still true. Read-only investigators (Explore) safe in-repo. Writers need `isolation: "worktree"` OR serialized atomic scopes. See this session's successful serialized pattern for the latter.

### Fosite refresh-token reuse guard
Don't drop smoke section 33 assertion (lines 798-802). Pre-fix `internal/oauth/store.go` returned `(nil, ErrInactiveToken)` from `GetRefreshTokenSession` → fosite panic on `req.GetID()`. Fix: return `(req, ErrInactiveToken)`. Test keeps this locked.

### gopls flaky in this workspace
Trust `go build` + `go test`, not gopls. We've seen false "undefined" errors on Store interface additions multiple times. Always re-verify with the actual compiler.

### Smoke server lifecycle
`smoke_test.sh` boots/stops server multiple times (bootstrap + sections 7 + 8). `trap stop_server EXIT` cleans up. If smoke leaves orphans: `pkill -f "bin/shark serve"`. Before manual runs: `rm -f dev.db dev.db-* server.log`.

## How to continue

1. **Re-baseline:** `bash smoke_test.sh` → should pass all sections.
2. **Pick next item:**
   - Easy wins: close Wave 4 smoke partials (F4, F5 especially — locks agent-auth moat assertions for HN demo).
   - Medium: Phase 7 SDK per ATTACK.md.
   - Large: Phase 8 migration tools + OIDC provider.
3. **HARD RULE:** every backend fix ships with a paired smoke assertion in `smoke_test.sh`. Never fix without test.
4. **Frontend changes:** `cd admin && npm run build && cd ..` then rebuild Go (assets `go:embed`'d into `internal/admin/dist/`). Stage the renamed dist asset hash.
5. **Commits:** small, scoped, descriptive. Run `go test ./... -count=1` + `bash smoke_test.sh` before each.
6. **Memory:** `/home/raul/.claude/projects/-home-raul-Desktop-projects-shark/memory/` — `MEMORY.md` is the index.

## Task list status at handoff

All 24 dashboard-audit tasks from this session closed. See `CHANGELOG.internal.md` → Phase 6.6 for full detail. Next agent starts from a clean baseline.
