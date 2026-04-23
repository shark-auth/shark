# Visual Flow Builder Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Dashboard-based visual configuration for auth flows and proxy routes. Auth0 Actions-like customization without code. Proxy route → permission mapping via drag-and-drop.

**Architecture:** Two components sharing a visual editor foundation:
1. **Proxy Config UI** — route rules, path matching, requirement assignment (simpler, ships first)
2. **Auth Flow Builder** — customizable auth pipelines ("after signup → verify email → enroll MFA → redirect")

**Tech Stack:** React + custom drag-drop (no library — keep bundle small) + Go backend for flow storage/execution

**Depends on:** Phase 6 (Proxy) for proxy config UI, Phase 5 (OAuth 2.1) for auth flow steps

---

## Component A: Proxy Config UI (ships with proxy)

Already covered in proxy plan Task 5. Visual rule editor with:
- Path input + requirement dropdown + scope input
- Rule ordering (drag to reorder)
- Live preview of effective rules
- Test URL tool ("what happens when GET /api/foo hits?")

---

## Component B: Auth Flow Builder (separate component)

### Data Model

```sql
-- Migration: 00012_auth_flows.sql
CREATE TABLE auth_flows (
    id          TEXT PRIMARY KEY,           -- flow_xxxx
    name        TEXT NOT NULL,              -- "Default Signup Flow"
    trigger     TEXT NOT NULL,              -- signup | login | password_reset | magic_link | oauth_callback
    steps       TEXT NOT NULL DEFAULT '[]', -- JSON array of step definitions
    enabled     INTEGER NOT NULL DEFAULT 1,
    priority    INTEGER NOT NULL DEFAULT 0, -- higher = checked first (for multiple flows per trigger)
    conditions  TEXT DEFAULT '{}',          -- JSON: when this flow applies (e.g., email domain match)
    created_at  TIMESTAMP NOT NULL,
    updated_at  TIMESTAMP NOT NULL
);
```

### Step Types

| Step | What it does | Configurable |
|------|-------------|--------------|
| `require_email_verification` | Block until email verified | timeout, redirect |
| `require_mfa_enrollment` | Prompt MFA setup | skip_if_enrolled |
| `require_mfa_challenge` | Challenge existing MFA | -- |
| `require_password_strength` | Enforce password rules | min_length, require_special |
| `redirect` | Redirect to URL | url, delay |
| `webhook` | Call external URL | url, method, headers, timeout |
| `set_metadata` | Set user metadata | key, value |
| `assign_role` | Auto-assign role | role_id |
| `add_to_org` | Auto-add to org | org_id, role |
| `custom_check` | Call URL, block if non-200 | url, error_message |
| `delay` | Wait N seconds | seconds |
| `conditional` | If/else branch | condition, then_steps, else_steps |

### Flow Execution Engine

```go
// internal/authflow/engine.go
type Engine struct {
    store   storage.Store
    flows   map[string][]*Flow  // trigger → flows (cached, invalidated on change)
}

func (e *Engine) Execute(ctx context.Context, trigger string, user *storage.User) (*FlowResult, error) {
    // Find matching flow for trigger + conditions
    // Execute steps in sequence
    // Return result: continue | block | redirect | error
}
```

### Tasks

#### Task 1: Flow Data Model + Storage
- [x] Create migration `00012_auth_flows.sql`
- [x] Define entity types
- [x] Implement CRUD storage methods
- [x] Write storage tests
- [x] Commit

#### Task 2: Flow Execution Engine
- [x] Implement `internal/authflow/engine.go`
- [x] Step executor for each step type
- [x] Condition evaluator (email domain match, metadata checks, time-based)
- [x] Error handling (step failure → configurable: block or skip)
- [x] Write tests for each step type
- [x] Commit

#### Task 3: Integration Points
- [x] Hook into signup handler — after user creation, before response
- [x] Hook into login handler — after auth success, before session/JWT
- [x] Hook into OAuth callback — after user link, before redirect
- [x] Hook into password reset — after reset, before redirect
- [x] Hook into magic link verify — after verify, before session
- [x] Write integration tests
- [x] Commit

#### Task 4: API Endpoints
- [x] `POST /api/v1/admin/flows` — create flow (admin)
- [x] `GET /api/v1/admin/flows` — list flows
- [x] `GET /api/v1/admin/flows/{id}` — get flow
- [x] `PATCH /api/v1/admin/flows/{id}` — update flow
- [x] `DELETE /api/v1/admin/flows/{id}` — delete flow
- [x] `POST /api/v1/admin/flows/{id}/test` — dry-run flow with mock user
- [x] Mount in router
- [x] Write handler tests
- [x] Commit

#### Task 5: Dashboard Visual Builder
- [x] Create `flow_builder.tsx` — visual flow editor
- [x] Step palette (left sidebar) — drag step types onto canvas
- [x] Canvas: vertical flow with step cards, connecting lines
- [x] Step configuration panel (right sidebar) — edit step properties
- [x] Condition editor for flow triggers
- [x] Preview mode: simulate flow with test user
- [x] Export as JSON (matches `steps` column format)
- [x] Register in App.tsx + layout.tsx
- [x] Commit

#### Task 6: Smoke Tests
- [x] Create flow via API
- [x] Flow executes on signup trigger
- [x] Conditional flow (email domain match)
- [x] Webhook step calls external URL
- [x] Flow disabled → skipped
- [x] Priority ordering (higher priority flow wins)
- [x] Commit

---

## Phasing Note

Proxy Config UI ships with Phase 6 (Proxy).
Auth Flow Builder is Phase 7-level complexity — can ship after SDK if needed.
Both share visual editing patterns that should be built as reusable components.
