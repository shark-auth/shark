# Handoff — Dashboard Gap Fix Branch

Branch: `claude/admin-vendor-assets-fix`
Smoke baseline at handoff: **375 PASS / 0 FAIL**
Cumulative growth: 244 → 375 (+131 assertions)

## What shipped on this branch (14 commits ahead of origin at handoff)

```
989c319 fix(dashboard): un-gate proxy + flow builder (shipped in Phase 6)
922f045 docs: mark Wave D + 3 + 4 done in DASHBOARD_GAPS
ac0dcb2 feat(dashboard): wave 3 — glue features (cmd+k, deep links, quick create, empty states, keybinds)
ffdf540 feat: wave D proxy rules CRUD + wave 4 smoke coverage backfill
43f2c82 fix(oauth): return requester with ErrInactiveToken on revoked refresh
6ffaa5c feat(api+dashboard): wave F — RBAC reverse lookup + email preview
4b772e3 feat(api+dashboard): wave D deferred + wave E — admin consents + device queue
5be21d1 feat(api+dashboard): wave C — admin vault connections list
743a49f feat(api+dashboard): wave B — users list filters (auth_method, org_id)
bf25a7f fix(dashboard): wave A — signing key rotate, overview mocks removed
48dfb63 fix(dashboard+api): wave 2 — silent fail polish + counter accuracy
86ec7ce feat(api): wave 1 — ship 7 missing admin routes
20e1d35 fix(dashboard): wave 0 webhook + user + health correctness
e27912b fix(oauth): PKCE persistence + token rotation collision
```

Read `DASHBOARD_GAPS.md` for the full audit + per-wave checkboxes. Wave summary:

- Wave -1 PKCE fix
- Wave 0 critical correctness (webhook list, MFA flag, last_login, health shape, test-fire)
- Wave 1 missing admin routes (webhook replay, org CRUD, MFA disable)
- Wave 2 silent fails (empty catches, audit filter, stats accuracy)
- Wave A frontend polish (overview mocks removed)
- Wave B users filters (auth_method, org_id)
- Wave C admin vault connections
- Wave D proxy rules CRUD (shipped via recovery branch)
- Wave E admin consents + device queue
- Wave F RBAC reverse lookup + email preview
- Wave 3 dashboard glue (Cmd+K, QuickCreate, Notifications, deep links, keybinds, PhaseGate)
- Wave 4 smoke coverage backfill (sections 64-69)
- fix: fosite refresh-token reuse panic (returned nil requester with ErrInactiveToken)

## What is NOT shipped — open work for next agent

### Wave 4 partials (smoke `note` lines remain)
- **G1 MFA TOTP full flow** — current smoke covers MFA toggle + admin disable; no end-to-end enroll → challenge → verify → recovery code consume. Needs TOTP code generator (`oathtool` or python `pyotp`).
- **G2 Passkey/WebAuthn flow** — only structural assertions on /begin endpoints. Full sign/verify needs virtual authenticator. Candidate for Go test, not bash smoke.
- **F4 Token Exchange happy path** — RFC 8693, agent-to-agent `act` chain. Need to seed two agents, mint subject_token + actor_token, exchange, decode resulting JWT, assert `act.sub`.
- **F5 DPoP full flow** — needs DPoP proof JWT construction (ECDSA P-256). Pure bash is gnarly; use python3 `cryptography` lib or move to Go test.
- **F6 Vault token retrieval** — needs in-process mock OAuth upstream. Stub OK; full flow blocked until mock server lands.
- **F7 Cache-Control headers** — backend may not set Cache-Control on `/.well-known/openid-configuration`, `/.well-known/jwks.json`, `/admin/health`. Verify + fix if missing.

### Settings page — currently read-only
File `admin/src/components/settings.tsx` displays Server config (Base URL, CORS, JWT mode, environment, session mode) as read-only. Has action buttons for: test email, purge sessions, purge audit, danger zone delete. Editing config requires:
- Backend: `PATCH /admin/config` handler that writes `sharkauth.yaml` + triggers hot-reload OR returns 503 with "restart required"
- Storage: yaml writer that preserves comments + ordering (use `yaml.v3` Node API)
- Hot-reload path: chi router + middleware re-init under mutex, OR simpler — restart-on-write with a process supervisor
- Frontend: convert read-only ConfigRow to editable inputs with save button

User indicated this is acceptable as Phase 8/9 work, not blocking.

### Phase gating (intentional placeholders)
Sidebar items with `ph:` markers in `admin/src/components/layout.tsx`:
- API Explorer / Session Debugger / Event Schemas — `ph: 5` (Phase 5 SDK; sub-features not built — render `EmptyShell` from `empty_shell.tsx`)
- OIDC Provider — `ph: 8` (us being IdP for other apps; SSO consumer flow is built and lives at `sso.tsx`)
- Impersonation / Compliance / Migrations / Branding — `ph: 9`

`PhaseGate` component (`admin/src/components/PhaseGate.tsx`) and `CURRENT_PHASE = 6` constant control gating logic. Bumping `CURRENT_PHASE` un-gates everything ≤ N.

### Wave D deferral notes (now shipped, leaving for context)
Original Wave D deferral worried about Engine reload + breaker capture. Final shipped design: `Engine.SetRules` swaps the rule set under a mutex; YAML stays as bootstrap, DB rules override. Migration `00015_proxy_rules.sql`. Frontend at `admin/src/components/proxy_config.tsx`. Smoke section 63 covers full CRUD.

## Critical lessons (don't repeat)

### Parallel agents need worktrees
Spawning 2+ background agents on the same repo without `isolation: "worktree"` causes catastrophic interference:
- `smoke_test.sh` — each agent appends own sections, last writer wins
- `internal/admin/dist/` — each rebuilds with new hash, deletes prior
- `dev.db` + port `:8080` — concurrent smoke runs collide (000/401 cascades)
- One agent stashes another's "conflict" files

**Mitigation:** ALWAYS pass `isolation: "worktree"` to the Agent tool when launching >1 parallel sub-agent on this repo. After agents complete, merge worktree branches back sequentially.

### Latent fosite refresh-token reuse panic
Pre-fix `internal/oauth/store.go` returned `(nil, ErrInactiveToken)` from `GetRefreshTokenSession` when token was revoked. fosite's `RefreshTokenGrantHandler.handleRefreshTokenReuse` (`flow_refresh.go:190`) calls `req.GetID()` on that nil requester → panic → server crash → cascading 000s on all subsequent smoke sections. Fix: return `(req, ErrInactiveToken)` so fosite can revoke the rotation family.

This bug was masked until Wave -1 fixed PKCE (refresh tokens were never issued before that). Don't drop the assertion at smoke section 33 lines 798-802.

### gopls in this workspace is broken
Diagnostics show false `undefined: SQLiteStore`, `undefined: writeJSON`, etc. Trust `go build` output (silent = pass), not gopls.

### Smoke server lifecycle gotchas
`smoke_test.sh` boots/stops the server multiple times: bootstrap, then explicit stop+restart at sections 7 (key rotation) and 8 (apps CLI). `trap stop_server EXIT` cleans up. If smoke leaves orphan shark processes (e.g. after Ctrl+C), kill them: `taskkill //F //IM shark.exe`.

`dev.db` and `server.log` are not auto-cleaned between runs. The smoke `bootstrap: fresh DB` section removes them at start. Manual runs should `rm -f dev.db dev.db-* server.log` first.

## How to continue

1. **Re-baseline:** `bash smoke_test.sh` → expect 375 PASS / 0 FAIL.
2. **Pick next item:** open `DASHBOARD_GAPS.md` and `ATTACK.md`. Wave 7 (SDK) is next per `ATTACK.md`. Wave 4 partials are quick wins for smoke completeness.
3. **HARD RULE for any backend fix:** ship paired smoke assertion in `smoke_test.sh`. Never fix without test.
4. **Frontend changes:** `cd admin && npm run build && cd ..` then rebuild Go (assets are `go:embed`'d into `internal/admin/`). Stage the renamed dist asset hash.
5. **Commits:** small, scoped, descriptive. One feature per commit. Run smoke before each commit.
6. **Memory:** `C:\Users\raulg\.claude\projects\C--Users-raulg-desktop-projects-shark\memory\` — `MEMORY.md` is the index. Phase 6.5 done is logged. Update when you ship Phase 7.
