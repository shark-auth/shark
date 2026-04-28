# Known Gaps & Battle-Test Requirements

_Documented on 2026-04-27 based on Sonnet Subagent Audit_

## 1. SDK Coverage & Integrity

- **Claim vs Reality**: `documentation/sdk/index.md` claims 75% coverage. Actual coverage is ~56% for Python and ~55% for TypeScript.
- **Broken Wrap (TS)**: `VaultClient.fetchToken` calls non-existent route `/admin/vault/connections/{id}/token`. Should be `/api/v1/vault/{provider}/token`.
- **SDK Gaps**: Passkeys (7), SSO (9), Flow Builder (7), Vault provider/template CRUD (11), admin sessions/stats (~20).

## 2. The "Moat" Bugs (P0 Launch Blockers)

- **Max Hops Enforcement**: `may_act_grants.max_hops` is stored but NOT enforced in `internal/oauth/exchange.go:272`. Agents can delegate to arbitrary depths.
- **Scope Widening**: `may_act_grants.scopes` is NOT enforced. Agents can exchange for scopes wider than the grant allows if the subject token carries them.
- **DPoP Replay**: JTI cache is in-memory only. Replay window re-opens on server restart.

## 3. Developer Experience (DX) & Quickstart

- **Broken Quickstart**: README suggests `pip install shark-auth`, but the package isn't on PyPI and the workflow is missing.
- **Device Flow**: Routes are mounted (`/oauth/device*`) but `HandleToken` returns 501. Either un-mount or finish implementation.
- **React Syntax**: Potential stray paren in `SharkProvider.tsx:86`.

## 4. Dogfooding Requirements

- **Admin Self-Auth**: Replace bootstrap-key login with real OAuth 2.1 PKCE flow against its own server.
- **Release Agent**: Create a Python `release-agent` script that uses the delegation chain to post release notes to GitHub.

---

## Execution Plan: Launch Readiness

**Note:** PyPI release will be handled separately by the maintainer. A specialized subagent should update the README to make Shark viral, including images and moat-forward messaging. Pytest should be run fully after all subagents finish.

### WAVE 1 (Parallel, Low-Risk Polish)
- **[ ] W1_README**: Viral rewrite of `README.md`, including screenshots/images and moat-forward positioning.
- **[ ] W1_DOCS**: Update `documentation/sdk/index.md` with honest SDK coverage % and a gap inventory.
- **[ ] W1_REACT**: Audit and fix the potential stray paren in `SharkProvider.tsx:86`.
- **[ ] W1_DEVICE**: Cleanly un-mount device flow routes for v0.1.
- **[ ] W1_CHECKLIST**: Persist the `launch-checklist.md` to `.planning/launch-checklist.md` based on the 40-item P0/P1 checklist.

### WAVE 2 (Parallel, After W1)
- **[ ] W2_MOAT**: Enforce `may_act` `max_hops` + `scopes` in `internal/oauth/exchange.go`. Add Go unit tests and pytest smoke for both gates.
- **[ ] W2_PASSKEYS**: Implement Passkeys in Python + TS clients (7 endpoints) + smoke tests.
- **[ ] W2_VAULT**: Fix TS VaultClient path bug + implement provider CRUD + user connect (11 endpoints) + smoke tests.
- **[ ] W2_FLOW**: Implement Flow builder CRUD (7 endpoints) + smoke tests.
- **[ ] W2_DPOP**: Fix DPoP replay window issue (JTI cache memory only).

### WAVE 3 (Parallel, After W2)
- **[ ] W3_ADMIN_OAUTH**: Replace bootstrap-key login in Admin UI with real OAuth 2.1 PKCE flow against its own server (Dogfooding).
- **[ ] W3_RELEASE_AGENT**: Build `demo-release-agent/` Python script + grant setup + audit-chain screenshot (Dogfooding).

### WAVE 4 (Final, Sequential)
- **[ ] E1_BUILD**: Admin build + Go binary embed.
- **[ ] E2_TEST**: FULL pytest run (tests/smoke + sdk smoke + new tests).
- **[ ] E3_REPORT**: Ship report — coverage % delta, test pass count, binary path.