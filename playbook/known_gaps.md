# Known Gaps & Battle-Test Requirements
*Documented on 2026-04-27 based on Sonnet Subagent Audit*

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
