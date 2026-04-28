# SharkAuth Launch & Battle-Test Checklist (P0/P1 Gaps)

## 1. API & Server Smoke (P0)
- [ ] `./shark serve` starts in < 200ms on clean DB.
- [ ] `--dev` mode correctly captures emails to `internal/admin/dev-email`.
- [ ] Server handles SIGINT/SIGTERM gracefully (closes SQLite handles).
- [ ] Concurrent request test (100 parallel calls to `/healthz`) passes without 5xx.
- [ ] Database migrations run cleanly on fresh install (no 00028 duplicate).

## 2. OAuth 2.1 Flows (P0)
- [ ] PKCE flow: code_challenge verification enforced.
- [ ] Client Credentials: valid `client_id`/`client_secret` issues token.
- [ ] Refresh tokens: rotation works, old token invalidated.
- [ ] Scopes: downscoping on token request honored.
- [ ] Redirect URIs: exact match enforced (no wildcard bypass).
- [ ] Passkeys: registration + login works on Chrome/Safari.
- [ ] Magic link: token expires after 15m, single-use only.

## 3. Delegation Moat (CRITICAL)
- [ ] **MOAT-1**: `may_act_grants.max_hops` enforced in `exchange.go`.
- [ ] **MOAT-2**: `may_act_grants.scopes` intersection enforced (no widening).
- [ ] Token exchange: `subject_token` and `actor_token` type validation.
- [ ] Delegation chain visualization: Edge labels show correct scopes.
- [ ] Cascade Revoke: Revoking a root grant invalidates all downstream tokens.
- [ ] Audit correlation: `grant_id` present in every delegated token event.
- [ ] Hop count: Audit log correctly counts hops in complex chains.
- [ ] Recursion check: Prevent infinite delegation loops.

## 4. DPoP (P0)
- [ ] JTI Replay: Re-using same JTI within 60s window returns 400.
- [ ] Key binding: Token issued for Key-A rejected when used with Key-B.
- [ ] `htu`/`htm` validation: Token rejected if used on wrong path/method.
- [ ] Error messages: Clear 401 response when DPoP header is missing.

## 5. Admin UI & SDK (P1)
- [ ] Sidebar: "Identity Hub" and "Delegation" icons visible and clickable.
- [ ] User Connect: Vault connections UI shows real-time status.
- [ ] SDK Parity: Python and TS both support `auth.check`.
- [ ] TS SDK: Fix `VaultClient.fetchToken` route path.
- [ ] SDK Docs: Honest coverage % matches actual exported methods.
- [ ] SharkProvider: Audit for stray parens/syntax errors.

## 6. Resilience & Install (P1)
- [ ] Binary: Cross-compile to Linux/Darwin/ARM64.
- [ ] README: Quickstart commands (install -> serve -> first user) work on blank machine.
- [ ] Restart: DPoP JTI cache persists or fails safely across restart.
