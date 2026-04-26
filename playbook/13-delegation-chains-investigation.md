# Delegation Chains Investigation — 2026-04-26

## Symptom
- /admin/delegation-chains page List view: empty
- /admin/delegation-chains page Canvas view: empty
- /admin/delegation-chains drawer: nothing
- Agent drawer Delegations tab: nothing
- tools/agent_demo_tester.py ran all 11 steps successfully, yet no chains appear.

---

## Pipeline trace

### 1. Backend audit emission (file:line evidence)

- Token-exchange handler at: `internal/oauth/exchange.go:36` — `func (s *Server) HandleTokenExchange`
- Audit log emitted? **NO**
- Lines 205–212 only call `slog.Info("oauth.token.exchanged", ...)` — this is a structured log line to stdout/stderr, **not** a call to `s.AuditLogger.Log(ctx, &storage.AuditLog{...})`.
- No `AuditLogger` field is referenced anywhere in `exchange.go`.
- The audit infrastructure (`internal/audit/audit.go:Logger.Log`) exists and works; other handlers (agent.created, agent.deactivated, user.deleted, etc.) use it correctly and their rows appear in the DB.
- **Verdict: MISSING** — token-exchange never writes to the `audit_logs` table.

### 2. act_chain population

- `act_chain` / `actClaim` is built correctly at `internal/oauth/exchange.go:129` via `buildActClaim(actingAgent.ClientID, subjectAct)`.
- It is marshalled to JSON at line 206: `actChainJSON, _ := json.Marshal(actClaim)` — but only passed to `slog.Info`, never to an `AuditLog.Metadata` field.
- `storage.AuditLog` struct (`internal/storage/storage.go:538`) has **no** dedicated `act_chain` column. It has a `Metadata string` field (JSON blob).
- There is no `act_chain` column in the `audit_logs` table — confirmed by looking at the storage struct and the live DB.
- **Verdict: MISSING** — `act_chain` is never persisted. Even if audit was emitted, the JSON would need to be placed in `Metadata`.

### 3. Audit list API

- File: `internal/api/audit_handlers.go:31` — `handleListAuditLogs`
- Response shape: `{ data: []*storage.AuditLog, next_cursor: string, has_more: bool }`
- Each event returns: `id, actor_id, actor_type, action, target_type, target_id, org_id, session_id, resource_type, resource_id, ip, user_agent, metadata, status, created_at`.
- No top-level `act_chain` field — it would need to live inside the `metadata` JSON string.
- Supports `?action=<string>` and `?actor_id=<string>` filters. Both work correctly.
- **Verdict: OK** (API works; problem is upstream — nothing to return).

### 4. Frontend data fetching — delegation_chains.tsx

- File: `admin/src/components/delegation_chains.tsx:726-735`
- URL: `/audit-logs?action=oauth.token.exchanged&limit=50` (plus optional `since=` param)
- Also fetches: `/audit-logs?action=vault.token.retrieved&limit=50`
- Action filter: `"oauth.token.exchanged"` — **matches** exactly what `exchange.go:207` passes to slog, but since no audit row is ever written with that action, the API returns `data: []`.
- Response parsing (line 765): `exchangeData?.items || exchangeData?.audit_logs || exchangeData?.data || []` — correctly handles the `{ data: [...] }` envelope.
- `act_chain` parsing (line 63-67 in `normalizeEntry`): looks at `e.act_chain` first, then `e.oauth?.act`. The `AuditLog` struct has neither — `act_chain` would need to be in `metadata` JSON and parsed out server-side or client-side. Client-side path in `normalizeEntry` does NOT parse `e.metadata` as JSON to extract `act_chain`.
- **Verdict: TWO drifts** — (a) no rows exist; (b) even if they did, `normalizeEntry` does not parse `act_chain` out of `metadata` JSON string (`e.metadata` is a raw string, not an object).

### 5. Frontend per-agent canvas — agents_manage.tsx Delegations tab

- File: `admin/src/components/agents_manage.tsx:1530-1531`
- URLs: `GET /api/v1/audit-logs?action=oauth.token.exchanged&actor_id={agent.id}&limit=100` and `GET /api/v1/audit-logs?action=oauth.token.exchanged&limit=200`
- act_chain parsing (line 1538): `ev.act_chain || ev?.metadata?.act_chain || ev?.meta?.act_chain || []`
- `ev.metadata` is a **string** (raw JSON), not a parsed object, so `ev?.metadata?.act_chain` is always `undefined`. The `||` falls through to `[]`.
- **Verdict: TWO drifts** — (a) no rows; (b) `metadata` is a string, not an object, so act_chain lookup always returns `[]`.

### 6. Run-time evidence

Live DB query after tester run (`sqlite3 shark.db`):

```
SELECT id, action FROM audit_logs WHERE action LIKE '%exchange%' OR action LIKE '%token%' OR action LIKE '%delegat%';
-- Result: 0 rows
```

All audit rows present are: `agent.created`, `agent.deactivated`, `agent.tokens_revoked`, `user.deleted_with_token_revocation`, `admin.user.create`. Zero `oauth.token.exchanged` rows exist.

---

## Root causes (ordered by severity)

1. **`HandleTokenExchange` never calls `s.AuditLogger.Log`** — `internal/oauth/exchange.go:205-212`. The token exchange completes, issues a signed JWT, stores the OAuth token, then exits after a bare `slog.Info`. No audit row is ever written. This is the primary cause of the empty views.

2. **`act_chain` is never placed in `AuditLog.Metadata`** — `internal/oauth/exchange.go:206`. `actChainJSON` is computed but only passed to `slog`. Even if the audit call were added, `act_chain` data would not be included unless explicitly embedded in the Metadata JSON field.

3. **Frontend treats `ev.metadata` as a parsed object but it is a raw JSON string** — `admin/src/components/agents_manage.tsx:1538` and `admin/src/components/delegation_chains.tsx:normalizeEntry`. `ev?.metadata?.act_chain` always evaluates to `undefined` because `metadata` is a string.

---

## Fix plan (DO NOT IMPLEMENT — just describe per cause)

- **Cause 1:** In `internal/oauth/exchange.go`, after line 212, add a call to `s.AuditLogger.Log(ctx, &storage.AuditLog{Action: "oauth.token.exchanged", ActorID: actingAgent.ID, ActorType: "agent", TargetType: "agent", TargetID: subjectSub, Metadata: <see cause 2>, Status: "success"})`. The `Server` struct in `internal/oauth/server.go` must expose an `AuditLogger *audit.Logger` field and it must be wired at startup.

- **Cause 2:** In `internal/oauth/exchange.go:206`, build the metadata JSON to include `act_chain`. Change the `Metadata` field of the emitted `AuditLog` to: `{"act_chain": <actClaim JSON>, "scope": "...", "subject": "...", "acting_agent": "..."}`. Ensure `actClaim` is nested recursively (it already is via `buildActClaim`).

- **Cause 3:** In `admin/src/components/agents_manage.tsx:1538` and `delegation_chains.tsx:normalizeEntry`, parse `metadata` as JSON before accessing nested fields:
  ```js
  const meta = (typeof ev.metadata === 'string') ? JSON.parse(ev.metadata || '{}') : (ev.metadata || {});
  const chain = ev.act_chain || meta.act_chain || [];
  ```

---

## Test gap

- What smoke would have caught this: `tests/smoke/` — missing test `test_delegation_chain_audit_persisted` — assertion that after `POST /oauth/token` with `grant_type=urn:ietf:params:oauth:grant-type:token-exchange`, `GET /api/v1/audit-logs?action=oauth.token.exchanged` returns `data` with length >= 1 and `data[0].metadata` contains `act_chain`.

---

## RESOLVED — 2026-04-26

All 3 root causes fixed in a single diff. Commit: "fix(delegation-chains): emit audit event on token-exchange + parse metadata in frontend".

### Fix 1 + Fix 2 — Backend audit emission + act_chain in metadata

**`internal/oauth/server.go`** — Added `AuditLogger *audit.Logger` field to `Server` struct; added `"github.com/sharkauth/sharkauth/internal/audit"` import.

**`internal/api/router.go`** — After `s.AuditLogger` is initialized (line ~173), wire it to `s.OAuthServer.AuditLogger` when OAuthServer is non-nil.

**`internal/oauth/exchange.go`** — After the existing `slog.Info("oauth.token.exchanged", ...)` block (line ~207), added:
```go
if s.AuditLogger != nil {
    metaMap := map[string]any{
        "act_chain": actClaim,         // map[string]interface{} — marshals as nested JSON object
        "scope":     strings.Join(grantedScopes, " "),
        "client_id": actingAgent.ClientID,
        "subject":   subjectSub,
    }
    metaJSON, _ := json.Marshal(metaMap)
    _ = s.AuditLogger.Log(ctx, &storage.AuditLog{
        Action: "oauth.token.exchanged", ActorID: actingAgent.ID,
        ActorType: "agent", TargetID: subjectSub, TargetType: "token",
        Status: "success", Metadata: string(metaJSON),
    })
}
```

### Fix 3 — Frontend metadata parse

**`admin/src/components/delegation_chains.tsx`** — Added `parseMeta(ev)` helper above `normalizeEntry`; `normalizeEntry` now reads `meta.act_chain` as fallback when `e.act_chain` is empty.

**`admin/src/components/agents_manage.tsx`** — DelegationsTab filter (line ~1538) now JSON-parses `ev.metadata` when it is a string before accessing `.act_chain`.

### Test added

**`tests/smoke/test_delegation_audit_emission.py`** — Creates 2 agents, performs token exchange, asserts `GET /api/v1/audit-logs?action=oauth.token.exchanged` returns >= 1 event with `actor_type=="agent"` and metadata JSON containing `act_chain`.

### Builds

- Backend: `go build -o shark.exe ./cmd/shark` — PASS (zero errors)
- Admin: `npx vite build` — PASS (275 modules, built in 13s)
