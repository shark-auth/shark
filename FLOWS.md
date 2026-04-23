# Auth Flow Builder

> Auth0 Actions-style visual editor for post-auth pipelines. Wire custom steps onto signup, login, OAuth callback, password reset, and magic link verify — no code.

**Status:** Shipped in Phase 6 (`claude/admin-vendor-assets-fix`, commits `a2d8bba..9663dfd`).

---

## TL;DR

```bash
# Create a flow via dashboard (Flows → + New flow) or API:
curl -X POST http://localhost:8080/api/v1/admin/flows \
  -H "Authorization: Bearer $ADMIN_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Default Signup Flow",
    "trigger": "signup",
    "steps": [
      {"type": "require_email_verification"},
      {"type": "require_mfa_enrollment", "config": {"skip_if_enrolled": true}}
    ],
    "enabled": true,
    "priority": 10
  }'
```

Every subsequent signup runs through these steps. If email isn't verified, signup returns `403 flow_blocked` with a reason. Agents see this immediately; dashboards can show it as a banner. No app-code changes required.

---

## Why Flows

Auth pipelines drift. Adding "require MFA for admin signups after Nov" means editing handler code, adding a feature flag, coordinating a deploy. Auth Flows move that logic into the dashboard:

- "After signup, require email verification before issuing a session" → one flow
- "Webhook our CRM on every login, block if CRM rejects" → `webhook` step
- "If user's email domain is `acme.com`, auto-assign admin role" → `conditional` + `assign_role`

Business rules live where business people can edit them. Dev deploys stay focused on code.

---

## Triggers

Flows fire at five integration points:

| Trigger | When it fires | After which mutation |
|---|---|---|
| `signup` | `POST /api/v1/auth/signup` | User created in DB |
| `login` | `POST /api/v1/auth/login` | Password verified (before MFA challenge) |
| `oauth_callback` | Social OAuth callback | User linked/created |
| `password_reset` | `POST /api/v1/auth/password/reset` | Password rotation committed |
| `magic_link` | `GET /api/v1/auth/magic-link/verify` | Verification complete |

**Key behavior:** flows run AFTER the underlying mutation succeeds. If the flow blocks, the user row (or new password) is already in the database — only the session/JWT issuance is skipped. This is deliberate: a blocking flow is an authorization gate, not a rollback mechanism.

---

## Step Types

Twelve step types, grouped by family:

### Block (stops flow unless condition met)

| Type | Config | Status |
|---|---|---|
| `require_email_verification` | `redirect: "https://..."` (optional) | ✅ Wired |
| `require_mfa_enrollment` | `skip_if_enrolled: bool` | ✅ Wired |
| `require_mfa_challenge` | — | ⏳ Stubbed (F2.1) |
| `require_password_strength` | `min_length: 12`, `require_special: true` | ✅ Wired (signup only) |
| `custom_check` | `url: "..."`, `error_message: "..."` | ⏳ Stubbed (F2.1) |

### Prompt (user interaction required)

Reserved for future step types. None wired in MVP.

### Side effect (mutations, integrations)

| Type | Config | Status |
|---|---|---|
| `webhook` | `url`, `method`, `headers: {...}`, `timeout: 5` | ✅ Wired |
| `set_metadata` | `key`, `value` | ⏳ Stubbed — patches `Result.Metadata` but doesn't persist to user row (F2.1) |
| `assign_role` | `role_id` | ⏳ Stubbed (F2.1) |
| `add_to_org` | `org_id`, `role` | ⏳ Stubbed (F2.1) |
| `delay` | `seconds` | ⏳ Stubbed (F2.1) |

### Branch (control flow)

| Type | Config | Status |
|---|---|---|
| `redirect` | `url` (required), `delay` (optional) | ✅ Wired |
| `conditional` | `condition` (JSON string), `then: [...]`, `else: [...]` | ✅ Wired |

**Stubbed steps** are dispatched at runtime — they log a warning and return Continue. Flows containing stubs still work end-to-end; just certain effects don't persist yet. Use `set_metadata` freely for staging state between steps within one flow; it gets surfaced in `Result.Metadata` today.

---

## Step outcomes

Each step returns one of:

| Outcome | Meaning | HTTP effect |
|---|---|---|
| `continue` | Step passed, run next step | Handler proceeds |
| `block` | Step denied; flow short-circuits | 403 `{"error":"flow_blocked","message":"<reason>"}` |
| `redirect` | Send user to URL | 302 with `Location` header + JSON `{"redirect_url":"..."}` body |
| `error` | Runtime error (webhook timeout, bad config) | **Non-fatal** — logged, handler proceeds as if no flow ran |

The `error` → non-fatal default is intentional. A webhook outage MUST NOT brick your auth endpoints. If you want errors to block, model them as `block` outcomes via `conditional` + `custom_check`.

---

## Conditions

Flows can gate themselves with top-level `conditions`. Only matching flows are considered for a trigger; highest priority wins.

```json
{
  "name": "Enterprise signup flow",
  "trigger": "signup",
  "priority": 100,
  "conditions": {
    "all_of": [
      {"email_domain": "acme.com"},
      {"has_metadata": "invite_code"}
    ]
  },
  "steps": [...]
}
```

### Predicates

| Predicate | Type | Semantics |
|---|---|---|
| `email_domain` | string | `user.Email` ends with `@value` |
| `has_metadata` | string | key present in Context.Metadata or user.Metadata |
| `metadata_eq` | map (one k/v) | metadata key equals value |
| `trigger_eq` | string | Context.Trigger equals value (redundant with top-level `trigger` but useful in branches) |
| `user_has_role` | string | role in user's role list |
| `all_of` | array | AND of nested predicates |
| `any_of` | array | OR of nested predicates |
| `not` | nested predicate | NEGATE |
| `{}` | — | always matches |

Unknown predicates return an error at evaluation time — the flow is skipped (with a logged warning), the next-priority candidate is tried.

### Conditional step

Same DSL inside a `conditional` step's `condition` field. `condition` is a JSON string; parse it as a map:

```json
{
  "type": "conditional",
  "condition": "{\"email_domain\": \"acme.com\"}",
  "then": [
    {"type": "assign_role", "config": {"role_id": "role_admin"}}
  ],
  "else": [
    {"type": "redirect", "config": {"url": "/onboarding/standard"}}
  ]
}
```

---

## Priority + selection

Multiple flows can target the same trigger. Engine picks by:

1. Fetch all enabled flows for trigger, ordered `priority DESC, created_at ASC`
2. Evaluate each flow's `conditions` against the execution context
3. First flow whose conditions match → execute
4. Bad conditions (JSON error, unknown predicate) → log + skip
5. No flow matches → `outcome: continue` (default — auth proceeds normally)

---

## Dry-run preview

The Flow Builder's Preview tab POSTs to `/api/v1/admin/flows/{id}/test` with a mock user payload. No `auth_flow_runs` row is persisted. The response includes a `timeline` array so the UI can render step-by-step:

```json
{
  "outcome": "block",
  "reason": "email verification required",
  "redirect_url": "",
  "blocked_at_step": 0,
  "timeline": [
    {
      "index": 0,
      "type": "require_email_verification",
      "outcome": "block",
      "reason": "email verification required",
      "started_at": "2026-04-19T20:00:00Z",
      "duration_ns": 120000
    }
  ],
  "metadata": {}
}
```

Four preset mock users ship in the dashboard (stored in `localStorage` under `sharkauth.flow.mocks`):
- **Fresh signup** — `email_verified: false`, no metadata
- **Existing user** — verified, one role
- **OAuth user** — oauth_callback trigger, verified via provider
- **Org admin** — verified + `user_has_role: admin`

Edit, save, reuse.

---

## History

Every Execute call (not DryRun) persists an `auth_flow_runs` row with the full timeline in `metadata`. Query via:

```bash
curl http://localhost:8080/api/v1/admin/flows/{id}/runs?limit=50
```

Dashboard History tab displays: Started · User · Outcome · Blocked step · Duration · Reason. Runs cascade-delete when the flow is deleted (FK).

---

## Admin API

All endpoints admin-keyed (Bearer in `Authorization`).

| Method | Path | Purpose |
|---|---|---|
| POST | `/api/v1/admin/flows` | Create flow |
| GET | `/api/v1/admin/flows[?trigger=X]` | List (optionally filter by trigger) |
| GET | `/api/v1/admin/flows/{id}` | Get single flow |
| PATCH | `/api/v1/admin/flows/{id}` | Partial update (name, steps, enabled, priority, conditions, trigger) |
| DELETE | `/api/v1/admin/flows/{id}` | Delete (cascades runs) |
| POST | `/api/v1/admin/flows/{id}/test` | Dry-run with mock user |
| GET | `/api/v1/admin/flows/{id}/runs?limit=N` | Run history (default 20, max 100) |

### Validation

- `name` non-empty
- `trigger` ∈ {signup, login, password_reset, magic_link, oauth_callback}
- `steps` at least 1, each with valid `type`
- Nested `then`/`else` validated recursively
- Invalid payload → 400 with descriptive error

### Sanitization

Flow JSON emits all public fields (id, name, trigger, steps, enabled, priority, conditions, timestamps). No secrets today; metadata is opaque JSON the caller supplied.

Flow run JSON emits: id, flow_id, user_id, trigger, outcome, blocked_at_step, reason, metadata, started_at, finished_at.

---

## Dashboard

Navigate to `/admin/flows` (keyboard: check sidebar for key).

### Flows list

Table view with filter (all/signup/login/etc.) and search. Click row → editor. `+ New flow` seeds a starter flow and jumps into editing.

### Flow editor

Three-pane layout:

- **Palette** (200px left): step types grouped by family. Click to insert after current selection.
- **Canvas** (fluid center): vertical flow, trigger pseudo-node at top, steps stacked, done pseudo-node at bottom. Click node → select. Hover → `⋮` menu (move/duplicate/delete). Conditional steps display linearly with indented then/else branches (forked visualization deferred to F4.1).
- **Config** (320px right): typed fields per step type — URL inputs, method dropdowns, headers key/value editor, condition textarea, etc. Generic JSON fallback for unknown shapes.

Tabs above panes: **Steps · Trigger conditions · Preview · History**.

Header: inline-editable name, trigger picker, enabled toggle (confirm on enable), Save/Revert buttons (shown only when dirty).

### Deferred (F4.1)

- Drag-drop step reordering (click-to-insert only today)
- Forked-canvas conditional visualization (linear-indented today)
- Canvas keyboard shortcuts (Delete, ↑↓, Enter)

---

## Webhook payload

`webhook` steps POST a JSON body with:

```json
{
  "trigger": "signup",
  "user": {
    "id": "usr_abc",
    "email": "alice@example.com",
    "email_verified": false,
    "name": "Alice"
  },
  "metadata": { "any_accumulated_state": "..." }
}
```

**Never included:** `password_hash`, `mfa_secret`, or any encrypted field. Webhook bodies are sanitized unconditionally. If your CRM or audit pipeline needs more data, model the flow with a preceding `set_metadata` step (stubbed today; raw result metadata available in `Result.Metadata`).

Default timeout 5s, capped at 30s via `config.timeout`. Non-2xx responses yield `outcome: error` (logged, non-fatal). Headers from config are merged with `Content-Type: application/json`.

---

## Operational notes

- **Non-fatal failures**: engine errors never brick auth. Flows that panic/error log a warning and let the handler proceed.
- **Single-flow execution**: only one flow runs per trigger invocation. No cascading across multiple matching flows.
- **Priority tiebreak**: same priority → older `created_at` wins (stable).
- **MVP login hook position**: fires AFTER password verify, BEFORE MFA challenge. Post-MFA hook placement deferred (F3.1).
- **Password reset**: hook fires after the password row updates. Blocking here means the user has their new password but can't log in yet.
- **Run persistence is best-effort**: a failed `CreateAuthFlowRun` insert doesn't block the response. Don't treat `auth_flow_runs` as authoritative audit — use it for dashboard history only.

---

## Data model

### `auth_flows`

```sql
id          TEXT PRIMARY KEY        -- flow_xxxxxxxxxxxxxxxxxxxx
name        TEXT NOT NULL
trigger     TEXT NOT NULL           -- enum
steps       TEXT NOT NULL DEFAULT '[]'  -- JSON array of step definitions
enabled     INTEGER NOT NULL DEFAULT 1
priority    INTEGER NOT NULL DEFAULT 0
conditions  TEXT NOT NULL DEFAULT '{}'  -- JSON predicate map
created_at  TIMESTAMP NOT NULL
updated_at  TIMESTAMP NOT NULL
```

Indexed on `trigger` + `priority DESC`.

### `auth_flow_runs`

```sql
id              TEXT PRIMARY KEY        -- fr_xxxxxxxxxxxxxxxxxxxx
flow_id         TEXT NOT NULL REFERENCES auth_flows(id) ON DELETE CASCADE
user_id         TEXT                    -- nullable (pre-signup)
trigger         TEXT NOT NULL
outcome         TEXT NOT NULL           -- continue | block | redirect | error
blocked_at_step INTEGER                 -- nullable
reason          TEXT
metadata        TEXT NOT NULL DEFAULT '{}'
started_at      TIMESTAMP NOT NULL
finished_at     TIMESTAMP NOT NULL
```

Indexed on `(flow_id, started_at DESC)` and `(user_id, started_at DESC)`.

---

## Testing

```bash
# Storage + engine + conditions — 35 pass
go test ./internal/authflow/... ./internal/storage/... -count=1 -run AuthFlow -v

# HTTP handlers + integration — 20 pass
go test ./internal/api/... -run Flow -count=1 -v

# Smoke tests (sections 50-54)
./smoke_test.sh
```

---

## Reference

- Plan: `docs/superpowers/plans/2026-04-18-visual-flow-builder.md`
- Code: `internal/storage/auth_flows*`, `internal/authflow/` (engine, steps, conditions), `internal/api/flow_handlers.go`, `admin/src/components/flow_builder.tsx`
- Tests: `internal/authflow/engine_test.go`, `internal/storage/auth_flows_sqlite_test.go`, `internal/api/flow_handlers_test.go`
