# Shark Bench — Expansion Plan (Phases B–H)

> Authored: 2026-04-27. Based on Phase A scaffold (commit `10ad4f5`), `playbook/16-benchmark-plan.md`,
> `internal/api/router.go`, and admin UI component survey.
> **DO NOT modify any file outside `bench/` when implementing.**

---

## Existing Phase A scenarios (already shipped)

| # | Name | One-liner |
|---|---|---|
| 1 | `signup_storm` | bcrypt cost=12 dominates; measures DB write throughput under random-email load |
| 2 | `login_burst` | pre-seeds 100 users; hammers `/api/v1/auth/login`; session-create hot path |
| 3 | `oauth_client_credentials` | DCR-registers a client, hammers `/oauth/token`; 179 RPS smoke baseline |

Phase A confirmed: token issuance hot path is healthy; bcrypt is the auth latency floor (~1.3s p50).

---

## Frontend feature inventory

Each top-level admin UI component → backend endpoints it exercises:

| Component | Backend endpoints hit |
|---|---|
| `agents_manage.tsx` | GET/POST/PATCH/DELETE `/api/v1/agents`, GET `/api/v1/agents/{id}/tokens`, POST `/{id}/tokens/revoke-all`, POST `/{id}/rotate-secret`, POST `/{id}/rotate-dpop-key`, GET `/{id}/audit`, GET/POST `/{id}/policies`, POST `/api/v1/admin/oauth/revoke-by-pattern`, GET `/api/v1/admin/oauth/consents`, GET `/api/v1/audit-logs?action=oauth.token.exchanged` |
| `vault_manage.tsx` | GET/POST/PATCH/DELETE `/api/v1/vault/providers`, GET `/api/v1/vault/templates`, GET `/api/v1/vault/connect/{provider}`, GET `/api/v1/vault/callback/{provider}`, GET/DELETE `/api/v1/vault/connections`, GET `/api/v1/vault/{provider}/token` |
| `delegation_chains.tsx` | GET `/api/v1/audit-logs?action=oauth.token.exchanged`, GET `/api/v1/audit-logs?action=vault.token.retrieved` (chain reconstruction from audit logs) |
| `users.tsx` | GET/DELETE/PATCH `/api/v1/users/{id}`, GET `/api/v1/users/{id}/roles`, GET `/api/v1/users/{id}/permissions`, GET `/api/v1/users/{id}/sessions`, DELETE `/api/v1/users/{id}/sessions`, GET `/api/v1/users/{id}/audit-logs`, POST `/api/v1/users/{id}/revoke-agents`, GET `/api/v1/admin/oauth/consents?user_id=`, DELETE `/api/v1/admin/oauth/consents/{id}`, POST `/api/v1/admin/users` |
| `organizations.tsx` | GET/POST/PATCH/DELETE `/api/v1/admin/organizations`, GET `/{id}/members`, POST `/{id}/invitations`, DELETE `/{id}/invitations/{id}` |
| `rbac.tsx` / `rbac_matrix.tsx` | GET/POST/PUT/DELETE `/api/v1/roles`, GET/POST/DELETE `/api/v1/permissions`, GET `/api/v1/admin/permissions/batch-usage`, POST `/api/v1/auth/check` |
| `sessions.tsx` | GET `/api/v1/admin/sessions`, DELETE `/api/v1/admin/sessions`, DELETE `/api/v1/admin/sessions/{id}`, POST `/api/v1/admin/sessions/purge-expired` |
| `session_debugger.tsx` | GET `/api/v1/admin/sessions`, GET `/api/v1/users/{id}/sessions` |
| `audit.tsx` | GET `/api/v1/audit-logs`, GET `/api/v1/audit-logs/{id}`, POST `/api/v1/audit-logs/export` |
| `webhooks.tsx` | GET/POST/PATCH/DELETE `/api/v1/webhooks`, GET `/api/v1/webhooks/events`, POST `/{id}/test`, GET `/{id}/deliveries`, POST `/{id}/deliveries/{deliveryId}/replay` |
| `consents_manage.tsx` | GET `/api/v1/admin/oauth/consents`, DELETE `/api/v1/admin/oauth/consents/{id}`, POST `/api/v1/admin/oauth/revoke-by-pattern` |
| `device_flow.tsx` | POST `/oauth/device`, POST `/oauth/token`, GET `/api/v1/admin/oauth/device-codes`, POST `/api/v1/admin/oauth/device-codes/{user_code}/approve`, POST `/{user_code}/deny` |
| `proxy_config_real.tsx` | GET `/api/v1/admin/proxy/status`, SSE `/api/v1/admin/proxy/status/stream`, GET/POST/PATCH/DELETE `/api/v1/admin/proxy/rules/db`, POST `/api/v1/admin/proxy/simulate`, POST `/api/v1/admin/proxy/start|stop|reload` |
| `branding.tsx` | GET/PATCH `/api/v1/admin/branding`, POST/DELETE `/api/v1/admin/branding/logo`, PATCH `/api/v1/admin/branding/design-tokens` |
| `sso.tsx` | GET/POST/PUT/DELETE `/api/v1/sso/connections` |
| `api_keys.tsx` | GET/POST/PATCH/DELETE `/api/v1/api-keys`, POST `/{id}/rotate` |
| `applications.tsx` | GET/POST/PATCH/DELETE `/api/v1/admin/apps`, POST `/{id}/rotate-secret`, GET `/{id}/snippet` |
| `signing_keys.tsx` | POST `/api/v1/admin/auth/rotate-signing-key`, GET `/.well-known/jwks.json` |
| `overview.tsx` / `identity_hub.tsx` | GET `/api/v1/admin/stats`, GET `/api/v1/admin/stats/trends`, GET `/api/v1/admin/health` |

---

## Backend route inventory by feature area

### Human Auth
| Route | Method | Notes |
|---|---|---|
| `/api/v1/auth/signup` | POST | bcrypt hash + session create |
| `/api/v1/auth/login` | POST | bcrypt verify + session create |
| `/api/v1/auth/logout` | POST | session destroy |
| `/api/v1/auth/me` | GET | session read |
| `/api/v1/auth/email/verify` | GET/POST | email token verify |
| `/api/v1/auth/password/send-reset-link` | POST | password reset flow |
| `/api/v1/auth/password/reset` | POST | password reset apply |
| `/api/v1/auth/password/change` | POST | authenticated change |

### Passkeys
| Route | Method | Notes |
|---|---|---|
| `/api/v1/auth/passkey/register/begin` | POST | WebAuthn ceremony start |
| `/api/v1/auth/passkey/register/finish` | POST | WebAuthn ceremony finish |
| `/api/v1/auth/passkey/login/begin` | POST | challenge generation |
| `/api/v1/auth/passkey/login/finish` | POST | assertion verify |
| `/api/v1/auth/passkey/credentials` | GET/DELETE/PATCH | credential management |

### Magic Links
| Route | Method |
|---|---|
| `/api/v1/auth/magic-link/send` | POST |
| `/api/v1/auth/magic-link/verify` | GET |

### MFA
| Route | Method |
|---|---|
| `/api/v1/auth/mfa/enroll` | POST |
| `/api/v1/auth/mfa/verify` | POST |
| `/api/v1/auth/mfa/challenge` | POST |
| `/api/v1/auth/mfa/recovery` | POST |
| `/api/v1/auth/mfa/recovery-codes` | GET |
| `/api/v1/auth/mfa` | DELETE |

### Sessions
| Route | Method |
|---|---|
| `/api/v1/auth/sessions` | GET (self) |
| `/api/v1/auth/sessions/{id}` | DELETE (self) |
| `/api/v1/admin/sessions` | GET/DELETE (admin) |
| `/api/v1/admin/sessions/{id}` | DELETE (admin) |
| `/api/v1/admin/sessions/purge-expired` | POST |
| `/api/v1/users/{id}/sessions` | GET/DELETE (admin) |

### OAuth / Token Issuance
| Route | Method |
|---|---|
| `/oauth/token` | POST | client_credentials, token exchange, device |
| `/oauth/authorize` | GET/POST | authorization code flow |
| `/oauth/register` | POST | DCR |
| `/oauth/register/{client_id}` | GET/PUT/DELETE | DCR management |
| `/oauth/introspect` | POST |
| `/oauth/revoke` | POST |
| `/oauth/device` | POST | device authorization grant |
| `/oauth/device/verify` | GET/POST | user approval |

### Agents
| Route | Method |
|---|---|
| `/api/v1/agents` | GET/POST |
| `/api/v1/agents/{id}` | GET/PATCH/DELETE |
| `/api/v1/agents/{id}/tokens` | GET |
| `/api/v1/agents/{id}/tokens/revoke-all` | POST |
| `/api/v1/agents/{id}/rotate-secret` | POST |
| `/api/v1/agents/{id}/rotate-dpop-key` | POST |
| `/api/v1/agents/{id}/audit` | GET |
| `/api/v1/agents/{id}/policies` | GET/POST |
| `/api/v1/users/{id}/agents` | GET |
| `/api/v1/users/{id}/revoke-agents` | POST |
| `/api/v1/me/agents` | GET |

### Vault
| Route | Method |
|---|---|
| `/api/v1/vault/providers` | GET/POST |
| `/api/v1/vault/providers/{id}` | GET/PATCH/DELETE |
| `/api/v1/vault/templates` | GET |
| `/api/v1/vault/connect/{provider}` | GET |
| `/api/v1/vault/callback/{provider}` | GET |
| `/api/v1/vault/connections` | GET/DELETE |
| `/api/v1/vault/{provider}/token` | GET |
| `/api/v1/admin/vault/connections` | GET/DELETE |

### RBAC
| Route | Method |
|---|---|
| `/api/v1/roles` | GET/POST/PUT/DELETE |
| `/api/v1/roles/{id}/permissions` | POST/DELETE |
| `/api/v1/permissions` | GET/POST/DELETE |
| `/api/v1/permissions/{id}/roles` | GET |
| `/api/v1/permissions/{id}/users` | GET |
| `/api/v1/users/{id}/roles` | POST/DELETE/GET |
| `/api/v1/auth/check` | POST |
| `/api/v1/admin/permissions/batch-usage` | GET |
| Org RBAC | GET/POST/PATCH/DELETE under `/api/v1/organizations/{id}/roles` |

### Organizations
| Route | Method |
|---|---|
| `/api/v1/organizations` | GET/POST |
| `/api/v1/organizations/{id}` | GET/PATCH/DELETE |
| `/api/v1/organizations/{id}/members` | GET/PATCH/DELETE |
| `/api/v1/organizations/{id}/invitations` | POST |
| `/api/v1/organizations/invitations/{token}/accept` | POST |
| `/api/v1/admin/organizations` | GET/POST/PATCH/DELETE |

### Audit Logs
| Route | Method |
|---|---|
| `/api/v1/audit-logs` | GET |
| `/api/v1/audit-logs/{id}` | GET |
| `/api/v1/audit-logs/export` | POST |
| `/api/v1/admin/audit-logs/purge` | POST |

### Webhooks
| Route | Method |
|---|---|
| `/api/v1/webhooks` | GET/POST/PATCH/DELETE |
| `/api/v1/webhooks/events` | GET |
| `/api/v1/webhooks/{id}/test` | POST |
| `/api/v1/webhooks/{id}/deliveries` | GET |
| `/api/v1/webhooks/{id}/deliveries/{id}/replay` | POST |

### Proxy
| Route | Method |
|---|---|
| `/api/v1/admin/proxy/status` | GET |
| `/api/v1/admin/proxy/status/stream` | GET (SSE) |
| `/api/v1/admin/proxy/rules` | GET |
| `/api/v1/admin/proxy/rules/db` | GET/POST/PATCH/DELETE |
| `/api/v1/admin/proxy/simulate` | POST |
| `/api/v1/admin/proxy/start|stop|reload` | POST |
| `/*` catch-all | reverse proxy handler |

### Branding / System
| Route | Method |
|---|---|
| `/api/v1/admin/branding` | GET/PATCH |
| `/api/v1/admin/branding/logo` | POST/DELETE |
| `/api/v1/admin/branding/design-tokens` | PATCH |
| `/api/v1/admin/stats` | GET |
| `/api/v1/admin/stats/trends` | GET |
| `/api/v1/admin/health` | GET |
| `/api/v1/admin/auth/rotate-signing-key` | POST |
| `/.well-known/jwks.json` | GET |
| `/.well-known/oauth-authorization-server` | GET |

### SSO
| Route | Method |
|---|---|
| `/api/v1/sso/connections` | GET/POST/PUT/DELETE |
| `/api/v1/sso/saml/{id}/metadata` | GET |
| `/api/v1/sso/saml/{id}/acs` | POST |
| `/api/v1/sso/oidc/{id}/auth` | GET |
| `/api/v1/sso/oidc/{id}/callback` | GET |

---

## Proposed bench expansion

> Phases follow `playbook/16-benchmark-plan.md` (A done, B→D defined). Expanded to B→H here.
> Total new scenarios: **29** (plus 3 existing = 32 total). Trim to ~30 by merging within phases as noted.

---

### Phase B — Human Auth + Sessions (4 scenarios)

**Prerequisite:** existing fixtures.go Bundle; extend with pre-hashed users.

---

#### B1 — `mfa_enroll_verify_throughput`

- **Goal metric:** RPS + p99 for the 3-step MFA flow (enroll → get TOTP secret → verify code)
- **Workload shape:** write-heavy, ramp 10→50 concurrency over 30s, sustained 30s
- **Fixtures:** 200 pre-seeded verified users with sessions
- **Routes:** `POST /api/v1/auth/mfa/enroll`, `POST /api/v1/auth/mfa/verify`
- **YC pitch:** "MFA enrollment completes under Xms p99 — full TOTP onboarding in a single request cycle"
- **Effort:** M
- **Phase:** B

---

#### B2 — `session_list_revoke_concurrent`

- **Goal metric:** p99 for session list + revoke under 100 concurrent users each revoking their own sessions; SQLITE_BUSY count
- **Workload shape:** read-heavy list, write burst revoke, 60s sustained
- **Fixtures:** 100 users × 5 sessions each (500 session rows)
- **Routes:** `GET /api/v1/auth/sessions`, `DELETE /api/v1/auth/sessions/{id}`, `DELETE /api/v1/admin/sessions/{id}`
- **YC pitch:** "session revocation is consistent sub-Xms at 100 concurrent users — zero ghost tokens"
- **Effort:** S
- **Phase:** B

---

#### B3 — `passkey_login_throughput`

- **Goal metric:** p99 for full passkey login (begin → finish) at sustained 50 rps
- **Workload shape:** mixed read-write (challenge gen + assertion verify), 30s ramp + 30s sustained
- **Fixtures:** 50 pre-registered passkey credentials (synthesize via admin API seeding or mock WebAuthn assertions)
- **Routes:** `POST /api/v1/auth/passkey/login/begin`, `POST /api/v1/auth/passkey/login/finish`
- **YC pitch:** "WebAuthn ceremonies under Xms p99 — drop-in passwordless without sacrificing speed"
- **Effort:** L (WebAuthn ceremony requires real crypto or mock stub)
- **Phase:** B

---

#### B4 — `admin_session_bulk_revoke`

- **Goal metric:** time-to-complete for revoking all sessions for 1000 concurrent-session tenant
- **Workload shape:** spike (single admin call), measure cascade write time
- **Fixtures:** 1000 active session rows pre-seeded
- **Routes:** `DELETE /api/v1/admin/sessions`, `POST /api/v1/admin/sessions/purge-expired`
- **YC pitch:** "compromise response: wipe 1000 sessions in under Xms — incident response without manual DB surgery"
- **Effort:** S
- **Phase:** B

---

### Phase C — Agent Auth + Delegation (the moat — 6 scenarios)

> These are the marquee scenarios. Prioritize for marketing run.

---

#### C1 — `oauth_dpop` ★ MARQUEE

- **Goal metric:** RPS + p99 vs `oauth_client_credentials` baseline; DPoP overhead delta
- **Workload shape:** sustained 200 concurrent, 60s; batched-reuse mode (default) + resign mode (comparison flag)
- **Fixtures:** 1 DCR-registered client with DPoP-bound key
- **Routes:** `POST /oauth/token` with DPoP proof header
- **YC pitch:** "DPoP-bound tokens at Xrps — same throughput as bearer with cryptographic sender-binding"
- **Effort:** S (scaffold exists in `dpop.go`, just wire scenario)
- **Phase:** C (originally Phase B in playbook — aligning here to moat grouping)

---

#### C2 — `token_exchange_chain` ★ MARQUEE

- **Goal metric:** p50/p99 at chain depth 1, 2, 3 — latency vs depth curve
- **Workload shape:** cascade=1 profile, runs to completion per depth; repeat 100× per depth for stable p99
- **Fixtures:** agent-A authorized by user U, agent-B authorized by agent-A, agent-C authorized by agent-B (depth-3 chain)
- **Routes:** `POST /oauth/token` with `grant_type=urn:ietf:params:oauth:grant-type:token-exchange`, `POST /api/v1/admin/consents` (setup)
- **YC pitch:** "depth-3 delegation chain resolves under Xms p99 — multi-hop agent orchestration without roundtrip hell"
- **Effort:** M
- **Phase:** C

---

#### C3 — `cascade_revoke_user_agents` ★ MARQUEE

- **Goal metric:** wall-clock ms to complete cascade revocation for 50 and 100 agents under one user
- **Workload shape:** cascade profile (1 conc, runs to completion); two fixture sizes
- **Fixtures:** 1 user with 50 agents (first run), then 100 agents
- **Routes:** `POST /api/v1/users/{id}/revoke-agents`
- **YC pitch:** "cascade-revoke 100 agents in under Xms — machine-to-machine token hygiene without O(n) admin calls"
- **Effort:** S
- **Phase:** C

---

#### C4 — `agent_rotate_dpop_key_load`

- **Goal metric:** RPS + p99 for DPoP key rotation under concurrent agent load
- **Workload shape:** write-heavy, 50 concurrent, 30s
- **Fixtures:** 50 registered agents
- **Routes:** `POST /api/v1/agents/{id}/rotate-dpop-key`
- **YC pitch:** "key rotation without downtime — agents re-key in under Xms while serving live traffic"
- **Effort:** S
- **Phase:** C

---

#### C5 — `agent_policy_read_hot`

- **Goal metric:** p99 for GET agent policies at 200 rps (policy enforcement read path)
- **Workload shape:** read-heavy, 100 concurrent, 60s sustained
- **Fixtures:** 100 agents each with 3-5 policies
- **Routes:** `GET /api/v1/agents/{id}/policies`
- **YC pitch:** "policy enforcement read path at Xrps — zero-overhead policy checks on every token grant"
- **Effort:** S
- **Phase:** C

---

#### C6 — `token_exchange_concurrent_depth2` ★ MARQUEE

- **Goal metric:** sustained RPS for parallel depth-2 exchanges; measures contention on consent table
- **Workload shape:** mixed, 100 concurrent, 60s sustained
- **Fixtures:** 100 user-agent-subagent triples, all pre-consented
- **Routes:** `POST /oauth/token` (token-exchange × 100 concurrent)
- **YC pitch:** "100 parallel delegation chains at Xrps — real orchestration workloads, not toy demos"
- **Effort:** M
- **Phase:** C

---

### Phase D — Vault + 3rd-party Token Brokering (4 scenarios)

---

#### D1 — `vault_read_concurrent` ★ MARQUEE

- **Goal metric:** RPS + p99 for vault token retrieval at 100 parallel requestors; AES-256 decryption cost
- **Workload shape:** read-heavy, 100 concurrent, 60s sustained
- **Fixtures:** 100 vault connections pre-seeded (one per user, `github` provider), tokens stored encrypted
- **Routes:** `GET /api/v1/vault/{provider}/token`
- **YC pitch:** "vault retrieval p99 under Xms at 100 concurrent — per-user encrypted token brokering at scale"
- **Effort:** M
- **Phase:** D

---

#### D2 — `cascade_revoke_vault_connections`

- **Goal metric:** wall-clock ms for vault disconnect cascade (50 pending retrievals, then delete connection)
- **Workload shape:** cascade profile, runs to completion
- **Fixtures:** 1 vault provider with 50 active connections across 50 users
- **Routes:** `DELETE /api/v1/vault/connections/{id}`, `DELETE /api/v1/admin/vault/connections/{id}`
- **YC pitch:** "vault provider revocation cascades across 50 users in under Xms — credential hygiene without user-by-user teardown"
- **Effort:** M
- **Phase:** D

---

#### D3 — `vault_provider_crud_throughput`

- **Goal metric:** RPS for provider create/read/delete cycle under admin load
- **Workload shape:** write-heavy, 20 concurrent, 30s
- **Fixtures:** none (creates + deletes its own)
- **Routes:** `POST /api/v1/vault/providers`, `GET /api/v1/vault/providers`, `DELETE /api/v1/vault/providers/{id}`
- **YC pitch:** "vault provider onboarding at Xrps — multi-provider catalogs without ops toil"
- **Effort:** S
- **Phase:** D

---

#### D4 — `vault_token_refresh_under_load`

- **Goal metric:** p99 for token re-fetch when cached token is expired (simulates refresh path)
- **Workload shape:** mixed, 50 concurrent, 30s sustained with forced-expired tokens
- **Fixtures:** 50 connections with expired `access_token` (seed as expired directly)
- **Routes:** `GET /api/v1/vault/{provider}/token` (triggers refresh branch)
- **YC pitch:** "transparent token refresh under load — agents never block on a stale credential"
- **Effort:** M (requires fixture tooling to seed expired tokens)
- **Phase:** D

---

### Phase E — RBAC + Organizations (4 scenarios)

---

#### E1 — `rbac_permission_check_hot`

- **Goal metric:** p99 for `POST /api/v1/auth/check` at 500 rps (the most-called admin path)
- **Workload shape:** read-heavy, 200 concurrent, 60s sustained
- **Fixtures:** 50 roles, 200 permissions, 1000 role assignments across users
- **Routes:** `POST /api/v1/auth/check`
- **YC pitch:** "permission enforcement at Xrps p99 under Xms — inline RBAC without a sidecar"
- **Effort:** S
- **Phase:** E

---

#### E2 — `rbac_role_assign_revoke_storm`

- **Goal metric:** RPS + SQLITE_BUSY for concurrent role grant/revoke (write contention test)
- **Workload shape:** write-heavy, 50 concurrent, 30s
- **Fixtures:** 100 users, 20 roles
- **Routes:** `POST /api/v1/users/{id}/roles`, `DELETE /api/v1/users/{id}/roles/{rid}`
- **YC pitch:** "role mutations at Xrps with zero read inconsistency — RBAC writes don't starve reads"
- **Effort:** S
- **Phase:** E

---

#### E3 — `org_member_invite_accept_flow`

- **Goal metric:** p99 for full invite→accept cycle under 50 concurrent orgs
- **Workload shape:** write-heavy, 50 concurrent, 30s
- **Fixtures:** 50 orgs, 50 pending invite tokens pre-seeded
- **Routes:** `POST /api/v1/admin/organizations/{id}/invitations`, `POST /api/v1/organizations/invitations/{token}/accept`
- **YC pitch:** "org invite flow at Xms p99 — B2B multi-tenancy onboarding without queue backlogs"
- **Effort:** M
- **Phase:** E

---

#### E4 — `batch_permission_usage_read`

- **Goal metric:** p99 for `/admin/permissions/batch-usage` with 100+ permissions (dashboard hot path)
- **Workload shape:** read-heavy, 20 concurrent, 30s
- **Fixtures:** 200 permissions, 50 roles, 1000 role-permission bindings
- **Routes:** `GET /api/v1/admin/permissions/batch-usage`
- **YC pitch:** "RBAC dashboard renders in under Xms even with 200+ permissions — no pagination pagination hacks"
- **Effort:** S
- **Phase:** E

---

### Phase F — Audit + Webhook delivery (3 scenarios)

---

#### F1 — `audit_emission_throughput` ★ MARQUEE

- **Goal metric:** audit rows/sec + lag from action→query-visible; overhead vs no-audit baseline
- **Workload shape:** write-heavy, 100 concurrent, 60s sustained; measure audit table row count delta
- **Fixtures:** 100 users doing login (each login emits audit row)
- **Routes:** `POST /api/v1/auth/login` (triggers audit emission), `GET /api/v1/audit-logs` (verify lag)
- **YC pitch:** "every auth event persisted to audit log at Xrps with under X% throughput overhead — SOC 2 ready at launch"
- **Effort:** M
- **Phase:** F

---

#### F2 — `webhook_fanout_latency`

- **Goal metric:** p99 delivery latency per endpoint under 10 subscriber webhooks; retry storm behavior
- **Workload shape:** sustained, 1 conc action trigger × 10 webhook endpoints (fanout), 60s; measure delivery queue depth
- **Fixtures:** 10 webhooks pre-registered (pointing at a local echo server started by the bench binary)
- **Routes:** `POST /api/v1/auth/login` (triggers fanout), `GET /api/v1/webhooks/{id}/deliveries`
- **YC pitch:** "webhook fanout to 10 endpoints in under Xms p99 — event-driven integrations without delay"
- **Effort:** L (needs a local echo HTTP server embedded in bench binary)
- **Phase:** F

---

#### F3 — `audit_log_export_concurrent`

- **Goal metric:** p99 + throughput for export under 10 concurrent admin queries on 100k row table
- **Workload shape:** read-heavy, 10 concurrent, 30s sustained
- **Fixtures:** 100k pre-inserted audit rows (bulk seed via fixtures)
- **Routes:** `POST /api/v1/audit-logs/export`, `GET /api/v1/audit-logs`
- **YC pitch:** "audit export at Xrps — compliance teams can pull 6-month exports without locking writes"
- **Effort:** M (requires bulk audit row seeding)
- **Phase:** F

---

### Phase G — OAuth surface (4 scenarios)

---

#### G1 — `oauth_introspect_hot`

- **Goal metric:** RPS + p99 for `POST /oauth/introspect` at 500 rps (resource server hot path)
- **Workload shape:** read-heavy, 200 concurrent, 60s sustained
- **Fixtures:** 1000 live access tokens pre-issued
- **Routes:** `POST /oauth/introspect`
- **YC pitch:** "token introspection at Xrps — resource servers validate bearer tokens without caching lag"
- **Effort:** S
- **Phase:** G

---

#### G2 — `dcr_register_rotate_load`

- **Goal metric:** RPS for DCR register + secret rotate cycle
- **Workload shape:** write-heavy, 50 concurrent, 30s
- **Fixtures:** none (self-seeding)
- **Routes:** `POST /oauth/register`, `POST /oauth/register/{id}/secret`, `DELETE /oauth/register/{id}`
- **YC pitch:** "DCR at Xrps — programmatic client registration for MCP and agent frameworks"
- **Effort:** S
- **Phase:** G

---

#### G3 — `device_code_poll_storm`

- **Goal metric:** p99 for device poll loop under 50 concurrent devices polling `POST /oauth/token`
- **Workload shape:** read-heavy (authorization_pending response), 50 concurrent, 30s
- **Fixtures:** 50 pending device codes pre-issued
- **Routes:** `POST /oauth/device`, `POST /oauth/token` with `grant_type=device_code`
- **YC pitch:** "device grant handles 50 concurrent polling clients at Xms p99 — IoT and CLI tool auth without thundering herd"
- **Effort:** M
- **Phase:** G

---

#### G4 — `oauth_revoke_bulk`

- **Goal metric:** RPS + time-to-complete for bulk token revocation pattern
- **Workload shape:** write-heavy, 20 concurrent, 30s; then single `revoke-by-pattern` call
- **Fixtures:** 1000 live access tokens across 100 clients
- **Routes:** `POST /oauth/revoke`, `POST /api/v1/admin/oauth/revoke-by-pattern`
- **YC pitch:** "tenant-scoped token revocation in under Xms — incident response without per-token teardown"
- **Effort:** S
- **Phase:** G

---

### Phase H — Cross-cutting: Proxy + Branding + Mixed Realistic (4 scenarios)

---

#### H1 — `proxy_auth_passthrough_load` ★ MARQUEE

- **Goal metric:** RPS + p99 for authenticated proxy passthrough (JWT-resolved identity, rule match, upstream forward)
- **Workload shape:** sustained, 200 concurrent, 60s; upstream is a local echo server
- **Fixtures:** 200 users with valid JWTs, 5 proxy rules loaded
- **Routes:** `/* catch-all` (proxy handler), `POST /oauth/token` (setup), upstream echo server
- **YC pitch:** "single-binary auth proxy at Xrps — replace your nginx + JWT verify sidecar with one process"
- **Effort:** L (needs local upstream echo + proxy config fixture)
- **Phase:** H

---

#### H2 — `branding_hot_reload`

- **Goal metric:** p99 for PATCH design tokens + asset serve under concurrent reads; verify no stale serve
- **Workload shape:** mixed (1 writer + 100 readers), 30s
- **Fixtures:** 1 branding config with logo uploaded
- **Routes:** `PATCH /api/v1/admin/branding/design-tokens`, `GET /assets/branding/*`
- **YC pitch:** "live branding updates under Xms — white-label tenants reskin without a deploy"
- **Effort:** S
- **Phase:** H

---

#### H3 — `mixed_realistic` (from playbook)

- **Goal metric:** sustained RPS over 60s under weighted blend; captures the full system under prod-like load
- **Workload shape:** 35% token issuance, 25% vault read, 20% login, 10% signup, 5% token exchange, 5% cascade revoke
- **Fixtures:** 1000 users, 200 agents, 100 vault connections, 50 roles/permissions
- **Routes:** all hot paths
- **YC pitch:** "Xrps sustained mixed workload on a 4GB single-binary — prod-like traffic without horizontal scaling"
- **Effort:** M
- **Phase:** H

---

#### H4 — `jwks_fetch_concurrency`

- **Goal metric:** RPS + p99 for `GET /.well-known/jwks.json` under 500 concurrent (resource server cold-start storm)
- **Workload shape:** read-heavy, 500 concurrent, 30s burst
- **Fixtures:** none
- **Routes:** `GET /.well-known/jwks.json`
- **YC pitch:** "JWKS endpoint handles 500 concurrent cold-start fetches under Xms — no stampede failures at scale-out"
- **Effort:** S
- **Phase:** H

---

## Coverage matrix

| Feature Area | Covered? | Phase | Scenario(s) |
|---|---|---|---|
| Signup / bcrypt | YES | A | `signup_storm` |
| Login / bcrypt | YES | A | `login_burst` |
| Client credentials token | YES | A | `oauth_client_credentials` |
| MFA enroll + verify | YES | B | `mfa_enroll_verify_throughput` |
| Session list + revoke | YES | B | `session_list_revoke_concurrent` |
| Passkey login | YES | B | `passkey_login_throughput` |
| Bulk session wipe | YES | B | `admin_session_bulk_revoke` |
| DPoP token issuance | YES | C | `oauth_dpop` |
| Token exchange chain | YES | C | `token_exchange_chain`, `token_exchange_concurrent_depth2` |
| Cascade revoke (agents) | YES | C | `cascade_revoke_user_agents` |
| Agent key rotation | YES | C | `agent_rotate_dpop_key_load` |
| Agent policy read | YES | C | `agent_policy_read_hot` |
| Vault token read | YES | D | `vault_read_concurrent` |
| Vault cascade revoke | YES | D | `cascade_revoke_vault_connections` |
| Vault provider CRUD | YES | D | `vault_provider_crud_throughput` |
| Vault token refresh | YES | D | `vault_token_refresh_under_load` |
| Permission check (RBAC) | YES | E | `rbac_permission_check_hot` |
| Role assign/revoke storm | YES | E | `rbac_role_assign_revoke_storm` |
| Org invite flow | YES | E | `org_member_invite_accept_flow` |
| Batch permission usage | YES | E | `batch_permission_usage_read` |
| Audit emission + lag | YES | F | `audit_emission_throughput` |
| Webhook fanout | YES | F | `webhook_fanout_latency` |
| Audit export | YES | F | `audit_log_export_concurrent` |
| Token introspection | YES | G | `oauth_introspect_hot` |
| DCR register + rotate | YES | G | `dcr_register_rotate_load` |
| Device code poll | YES | G | `device_code_poll_storm` |
| Bulk token revoke | YES | G | `oauth_revoke_bulk` |
| Proxy passthrough | YES | H | `proxy_auth_passthrough_load` |
| Branding hot reload | YES | H | `branding_hot_reload` |
| Mixed realistic | YES | H | `mixed_realistic` |
| JWKS cold-start | YES | H | `jwks_fetch_concurrency` |
| Magic links | NO | — | low-priority; latency dominated by email transport, not shark |
| Password reset flow | NO | — | email-gated; not benchmarkable in black-box mode |
| SSO SAML/OIDC | NO | — | requires external IdP stub; defer |
| Auth flows engine | NO | — | complex fixture setup; defer to post-launch |
| Signing key rotation | NO | — | one-shot admin op; not a throughput scenario |
| API keys CRUD | NO | — | commodity CRUD; covered implicitly by admin auth middleware |
| Email templates | NO | — | admin read path; not latency-critical |
| Application CRUD | NO | — | commodity CRUD |

---

## Headline metrics for YC pitch

These are the 7 marquee numbers. Run `--profile marketing` to validate. Honest targets based on Phase A smoke data:

1. **`oauth_client_credentials`** — 1000+ token grants/sec on a 4GB single-binary (Phase A already shows 179 RPS at 10 concurrency; 200-concurrency marketing run should approach this)
2. **`token_exchange_chain`** — depth-3 delegation resolves under 30ms p99 (depth-1 ~10ms baseline)
3. **`cascade_revoke_user_agents`** — 100-agent cascade completes under 1 second wall-clock
4. **`vault_read_concurrent`** — vault retrieval p99 under 50ms at 100 concurrent (AES-256 decryption in hot path)
5. **`oauth_dpop`** — DPoP overhead under 3ms p99 vs bearer baseline (batched-reuse mode)
6. **`rbac_permission_check_hot`** — permission check at 500+ RPS under 5ms p99 (pure read path, SQLite read-concurrency)
7. **`proxy_auth_passthrough_load`** — authenticated proxy at 200+ RPS p99 under 20ms (single process: auth + proxy + rule engine)

---

## Risks + caveats

**SQLite WAL bottlenecks:**
- SQLite allows only one writer at a time even under WAL mode. Write-heavy scenarios (signup, role assign, cascade revoke) will hit BUSY waits at high concurrency. Document BUSY count in output. Do NOT imply Postgres-equivalent write throughput.
- SQLite read concurrency is strong. Read-heavy scenarios (vault read, permission check, introspect) are the honest showcase.
- Recommend: frame public numbers as "single-binary on a 4GB Linode, SQLite-backed" — not "production distributed scale."

**bcrypt floor:**
- Signup and login p99 will always be 1–3s at cost=12. This is correct security posture. Frame it: "auth latency floor is bcrypt cost=12 by design; production deployments can tune cost."

**Vault refresh (D4):**
- Seeding expired tokens requires direct DB writes or a time-travel fixture. If that's not possible black-box, skip and note "refresh path untested."

**Webhook fanout (F2):**
- Requires embedding a local echo HTTP server in the bench binary. If complexity is high, demote to Phase H or simplify to measure dispatch queue depth only (no live delivery).

**Passkey (B3):**
- Full WebAuthn ceremony requires real CBOR-encoded assertions. May need a mock mode in the bench client. If blocked, downgrade to "register-begin only" throughput test (measures challenge generation, not full assertion verify).

**Proxy (H1):**
- Requires a running shark instance with proxy enabled + upstream configured. Add `--proxy-upstream` flag to `cmd/bench/main.go`. Document as optional scenario.

**Phase sequencing:**
- B and C are prerequisites for D (vault uses agent bearer tokens for retrieval).
- F requires scenarios from B (login triggers audit).
- H3 (`mixed_realistic`) should be last — it composites all previous scenarios.
- Do NOT run `stress` profile on CI. Marketing run is manual, documented with machine specs.

**What NOT to claim:**
- Do not compare to Auth0/Clerk/WorkOS throughput numbers. Different architecture (multi-region, Postgres, edge). Frame shark as "single-binary, self-hosted, no SaaS tax."
- Do not claim "infinite scale" — SQLite is the honest constraint. The moat is the feature set (delegation chains, vault, DPoP), not raw throughput.
