# Token Vault Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Managed third-party OAuth token storage — agents request tokens for Google/Slack/GitHub/etc through Shark, never handling raw credentials directly.

**Architecture:** Reuses existing social OAuth client infrastructure. Two new tables (`vault_providers`, `vault_connections`). Tokens encrypted at rest with AES-256-GCM (existing `FieldEncryptor`). Auto-refresh on retrieval. Rich dashboard UI for provider management.

**Tech Stack:** Go + SQLite + AES-256-GCM encryption + React dashboard

**Spec Reference:** `AGENT_AUTH.md` — Token Vault section

**Depends on:** Phase 5 (OAuth 2.1 Agent Auth) — agents must exist to request vault tokens

---

## File Structure

```
internal/vault/
├── vault.go                # Manager: connect, retrieve, refresh lifecycle
├── vault_test.go           # Unit tests
├── providers.go            # Pre-built provider templates (Google, Slack, etc.)
└── providers_test.go       # Provider template tests

internal/storage/
├── vault.go                # Vault CRUD (providers, connections)
└── vault_test.go           # Storage tests

internal/api/
├── vault_handlers.go       # /api/v1/vault/* HTTP handlers
└── vault_handlers_test.go  # Handler tests

cmd/shark/migrations/
└── 00011_vault.sql         # vault_providers + vault_connections tables

admin/src/components/
└── vault_manage.tsx        # Vault dashboard page (replace stub)

scripts/
└── smoke_vault.sh          # Vault smoke tests
```

---

## Tasks

### Task 1: Migration + Storage
- [ ] Create `00011_vault.sql` (schema per AGENT_AUTH.md Token Vault section)
- [ ] Define entity types in `internal/storage/vault.go`
- [ ] Extend Store interface with vault methods
- [ ] Implement SQLite CRUD in `internal/storage/vault_sqlite.go`
- [ ] Write storage tests
- [ ] Commit

### Task 2: Vault Manager
- [ ] Implement `internal/vault/vault.go` — connect flow (OAuth), retrieve+auto-refresh, disconnect
- [ ] Token encryption: use existing `FieldEncryptor` for access_token_enc, refresh_token_enc
- [ ] Auto-refresh: on GetToken(), if expired → refresh → store new tokens → return fresh
- [ ] Error handling: refresh failure → mark connection as `needs_reauth` → return specific error
- [ ] Write tests (mock HTTP for OAuth exchange, test refresh logic, encryption round-trip)
- [ ] Commit

### Task 3: Pre-built Provider Templates
- [ ] Create templates for: Google (Calendar/Drive/Gmail), Slack, GitHub, Microsoft, Notion, Linear, Jira
- [ ] Each template: auth_url, token_url, default scopes, icon URL, display name
- [ ] Admin creates provider by selecting template + adding client_id/secret
- [ ] Write tests
- [ ] Commit

### Task 4: API Handlers
- [ ] `POST /api/v1/vault/providers` — register provider (admin)
- [ ] `GET /api/v1/vault/providers` — list providers
- [ ] `PATCH /api/v1/vault/providers/{id}` — update
- [ ] `DELETE /api/v1/vault/providers/{id}` — remove
- [ ] `GET /api/v1/vault/connect/{provider}` — start OAuth flow (session auth)
- [ ] `GET /api/v1/vault/callback/{provider}` — OAuth callback, store tokens
- [ ] `GET /api/v1/vault/{provider}/token` — get fresh token (OAuth agent auth — validates delegation)
- [ ] `GET /api/v1/vault/connections` — list user's connections (session auth)
- [ ] `DELETE /api/v1/vault/connections/{id}` — disconnect
- [ ] Mount in router
- [ ] Write tests
- [ ] Commit

### Task 5: Dashboard UI
- [ ] Create `vault_manage.tsx` — split grid (providers list + connection detail)
- [ ] Provider setup wizard (select template → enter credentials → save)
- [ ] Connection status indicators (active/needs_reauth/expired)
- [ ] Token usage audit (last retrieved, by which agent)
- [ ] Register in App.tsx + layout.tsx, remove stub
- [ ] Build admin assets, verify in browser
- [ ] Commit

### Task 6: Smoke Tests
- [ ] Provider CRUD via admin API
- [ ] OAuth connect flow (mock provider)
- [ ] Token retrieval + auto-refresh
- [ ] Disconnect + cleanup
- [ ] Agent requests vault token via OAuth (delegation validation)
- [ ] Audit events fire for all vault operations
- [ ] Commit

---

## Security Requirements
- Third-party tokens NEVER returned in API responses except via `/vault/{provider}/token`
- Tokens encrypted at rest (AES-256-GCM)
- Agent must have valid OAuth token with user delegation to access vault tokens
- Vault token endpoint validates: agent is authorized + user has connection + scope allows vault access
- Audit: every token retrieval logged with agent_id, provider, user_id
