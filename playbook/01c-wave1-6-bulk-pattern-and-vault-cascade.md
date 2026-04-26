# Wave 1.6 — Layers 4 + 5 (Bulk-Pattern Revoke + Vault Cascade)

**Budget:** 7h CC · **Optional for Monday launch · Mandatory before Thursday Video B + YC application**

## Context

Wave 1.5 ships Layer 3 (customer-fleet cascade). Wave 1.6 ships Layers 4 and 5 of the depth-of-defense model. Together with Layers 1-2 (already shipped), all five layers are present and the YC pitch ("five layers, one mental model") is honest.

| Layer | Status after Wave 1.5 | After Wave 1.6 |
|---|---|---|
| 1. Per-token RFC 7009 | ✅ ships | ✅ |
| 2. Per-agent revoke-all | ✅ ships | ✅ |
| 3. Per-customer cascade | ✅ Wave 1.5 | ✅ |
| 4. Per-agent-type bulk by pattern | ❌ missing | ✅ Wave 1.6 |
| 5. Per-vault-credential cascade | ❌ missing | ✅ Wave 1.6 |

## Why ship by Thursday, not Monday

The Monday launch can credibly claim "five layers" if 3 ship at launch + the launch post explicitly says "layers 4 and 5 ship within 2 weeks." This is honest scoping that HN respects more than vaporware.

But the YC application video (Thursday-Friday recording per Wave 4.5) and the YC application itself (Saturday submission) are stronger if all five layers actually ship by then. Wednesday-Thursday is the right window.

## Edit 1 — Bulk revoke by client_id pattern (Layer 4)

**Threat:** agent-type v3.2 ships a bug across all customers (think: OpenAI Custom GPT template with a credential leak). Need to kill all instances of v3.2 without touching customers.

**File:** `internal/storage/oauth_sqlite.go`

```go
// RevokeOAuthTokensByClientIDPattern revokes all tokens whose client_id matches
// a SQLite GLOB pattern. Pattern syntax: * matches any sequence, ? matches one char.
// Returns number of tokens revoked.
func (s *SQLiteStore) RevokeOAuthTokensByClientIDPattern(ctx context.Context, pattern string) (int64, error) {
    res, err := s.db.ExecContext(ctx,
        `UPDATE oauth_tokens SET revoked_at = ? WHERE client_id GLOB ? AND revoked_at IS NULL`,
        time.Now().UTC().Format(time.RFC3339), pattern)
    if err != nil {
        return 0, err
    }
    return res.RowsAffected()
}
```

**File:** `internal/api/admin_oauth_handlers.go` (new file or extend existing)

```go
// POST /api/v1/admin/oauth/revoke-by-pattern
// Body: { "client_id_pattern": "shark_agent_v3.2_*", "reason": "v3.2 buggy 2026-04-26" }
// Returns: { "revoked_count": 142, "audit_event_id": "..." }
// Auth: admin API key only.
```

**Audit event:** `oauth.bulk_revoke_pattern` with metadata `{ pattern, revoked_count, reason }`.

**UI surface:** add a "Bulk Revoke by Pattern" button in `admin/src/components/agents_manage.tsx` (or a new `incident_response.tsx` component). Confirmation modal showing pattern preview + count of tokens that would match BEFORE confirming.

**Effort:** ~3h CC.

## Edit 2 — Vault disconnect cascades to agents (Layer 5)

**Threat:** customer's external Gmail OAuth gets phished. Customer revokes at Gmail. Today: SharkAuth doesn't know. Agents with cached vault tokens auto-refresh and keep working.

**File:** `internal/api/vault_handlers.go`

When a vault connection is deleted (DELETE `/api/v1/vault/connections/{id}` or admin path):
1. Emit audit event `vault.disconnected` with `{ provider_id, provider_name, user_id }` metadata
2. Query all agents that have ever retrieved from this vault connection (via audit log lookup `where action=vault.token.retrieved AND target=connection_id`)
3. For each agent in that set, revoke its current tokens via `RevokeOAuthTokensByClientID`
4. Emit audit event `vault.disconnect_cascade` with metadata `{ vault_connection_id, revoked_agent_ids, revoked_token_count }`

**Storage helper needed:** `ListAgentsByVaultRetrieval(connectionID string) ([]Agent, error)` — queries audit log to find which agents have ever fetched from this vault.

**Why this matters in the demo:** the user's own observation about "secure unhackable agents" requires that an external compromise (phished Gmail) propagates BACK to invalidate agents holding scope on it. Without this, an attacker who steals the customer's Gmail OAuth can keep using SharkAuth's cached tokens after the customer thinks they revoked at Gmail.

**Effort:** ~4h CC.

## Definition of done for Wave 1.6

- Bulk-pattern revoke endpoint + UI button + audit event
- Vault disconnect cascade + audit events
- Smoke tests for both:
  - `tests/smoke/test_bulk_pattern_revoke.py`: register 3 agents matching pattern, 2 not matching, run pattern revoke, assert 3 revoked, 2 unaffected
  - `tests/smoke/test_vault_disconnect_cascade.py`: agent retrieves from vault, vault disconnected, agent's next request 401s
- Smoke suite remains 375+ PASS
- Updates to README, HN follow-up post, and YC application: "all five layers shipped"

## Demo screencast for Wave 1.6 (use in Wed-Thu recording)

Two extra 30s clips for Twitter / Video B:
- **Bulk pattern revoke:** show admin running `POST /admin/oauth/revoke-by-pattern { client_id_pattern: "demo_v3.2_*" }` killing 50 token records in 1 second, audit event preview
- **Vault disconnect cascade:** show customer disconnecting Gmail → audit event fires → 3 agents holding `vault:gmail:read` scope all 401 on next request → audit shows the cascade event

These clips are Twitter-thread material for Friday-Saturday — closes the launch-week campaign with "all five layers shipped."

## Skip condition

If energy is genuinely on the floor by Wednesday:
- Ship just bulk-pattern revoke (Layer 4) — ~3h, lower risk, smaller surface
- Defer vault cascade (Layer 5) to W+2 with a public roadmap commitment
- YC application says "4 of 5 layers shipped, 5th lands W+2" — still credible, still differentiated
