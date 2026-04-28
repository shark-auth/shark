# Mortem Post

## Shipped on 2026-04-27
- **W2_MOAT**: Enforced `may_act` `max_hops` and `scopes` on token exchange in `internal/oauth/exchange.go` and added `countHops` helper.
- **W2_VAULT**: Fixed TS SDK `VaultClient.fetchToken` URL bug where it pointed to an admin endpoint instead of `/api/v1/vault/{provider}/token`.
- **W2_DPOP**: Persisted the DPoP JTI cache across restarts by reading/writing to `dpop_cache.json` in `internal/oauth/dpop.go`.
- **W1_KNOWN_GAPS**: Converted the raw subagent log in `playbook/known_gaps.md` into a structured execution plan.

All changes were implemented through precise block replacements (not full file overwrites). Verified via `git diff origin/main` to ensure no original logic was lost.