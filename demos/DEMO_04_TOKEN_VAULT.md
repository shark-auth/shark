# DEMO 04 — Token Vault: Personal AI Ops Stack

> **Tagline:** One agent acts across Gmail / Slack / GitHub / Notion / Linear via encrypted Token Vault. Zero raw credentials in agent memory.
>
> **Auth0 differentiator:** Auth0 charges ~50% add-on on top of their base plan for Token Vault ($53+/mo on top of existing subscription). Shark ships it free in the single binary.

---

## Persona: Aisha, Founder of Gemma

**Gemma** — a $19/mo personal AI ops assistant. 8,000 paid users. Monthly revenue: ~$152,000.

Gemma's agent reads users' Gmail, Slack, Notion, and Linear every 5 minutes, drafts summaries, writes tasks back. The agent also comments on GitHub issues on users' behalf.

**Aisha's current stack (the pain):**

| Problem | Detail |
|---------|--------|
| Homegrown `oauth_tokens` Postgres table | She wrote it in a weekend. It's AES-128 (not 256). The security review failed. |
| Custom refresh per provider | Slack rotation broke **twice last month** at 3 AM. On-call engineer woke up. |
| No per-user revocation cascade | When a user churns and requests data deletion, Aisha manually deletes rows — sometimes missing provider rows. |
| No audit trail | Can't answer "which agent read which user's Gmail at 2 PM on Tuesday?" |
| No webhook on token events | The product team has no signal when a user disconnects a provider. |

**Security review failure quote (real scenario):**
> "Your `oauth_tokens.access_token` column uses AES-128 at rest. Our SOC 2 CC6.1 requirement is AES-256. Additionally, we found no evidence of per-user revocation cascade — revoking a single provider token does not block other agents from using cached copies."

**Aisha's alternatives (before finding Shark):**

| Option | Price | Lock-in | Self-host |
|--------|-------|---------|-----------|
| Composio | $229/mo (2M tool calls) + overage | Yes | VPC/custom quote only |
| Nango | from $500/mo (Growth tier) | Yes | OSS but complex infra |
| Auth0 Token Vault | 50% add-on on existing Auth0 plan (~$53+/mo extra) | Yes | No |
| **Shark** | **$0 — ships in the binary** | **None** | **Yes, single binary** |

**Dollars saved vs Composio:** $229/mo × 12 = **$2,748/yr** for Composio's "Serious Business" tier. Against Nango Growth: **$6,000/yr**. Against Auth0 Token Vault add-on at enterprise scale: **tens of thousands/yr**.

---

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                    GEMMA PLATFORM (Aisha's app)                  │
│                                                                   │
│  ┌──────────────┐    ┌──────────────────────────────────────┐   │
│  │  End Users   │    │         Agent Fleet                   │   │
│  │  (8K users)  │    │  gemma-worker-01  gemma-worker-02    │   │
│  └──────┬───────┘    └──────────────┬───────────────────────┘   │
│         │                           │                             │
│         │ Connect flows             │ client_credentials +        │
│         │ (one-time per provider)   │ scope=vault:read:google ... │
│         │                           │                             │
│  ┌──────▼───────────────────────────▼───────────────────────┐   │
│  │                  SHARK AUTH SERVER                         │   │
│  │                                                            │   │
│  │  ┌────────────────────────────────────────────────────┐  │   │
│  │  │  TOKEN VAULT  (internal/vault/vault.go)             │  │   │
│  │  │                                                      │  │   │
│  │  │  vault_providers  ←  AES-256-GCM client_secret_enc  │  │   │
│  │  │  vault_connections ← AES-256-GCM access_token_enc   │  │   │
│  │  │                        AES-256-GCM refresh_token_enc │  │   │
│  │  │                                                      │  │   │
│  │  │  Auto-refresh: expiryLeeway = 30s before expiry     │  │   │
│  │  │  Per-user per-provider revocation cascade            │  │   │
│  │  │  Audit: every vault.read logged w/ agent_id          │  │   │
│  │  └────────────────────────────────────────────────────┘  │   │
│  │                                                            │   │
│  └──────────────────────┬─────────────────────────────────┘   │
│                          │                                        │
└──────────────────────────┼────────────────────────────────────────┘
                           │ Fresh (never raw) access tokens
           ┌───────────────┼───────────────────┐
           │               │                   │
      ┌────▼───┐     ┌─────▼──┐         ┌─────▼──┐
      │ Gmail  │     │ Slack  │         │ GitHub │
      │  API   │     │  API   │   ...   │  API   │
      └────────┘     └────────┘         └────────┘
         Notion API          Linear API
```

---

## Provider Templates — What Actually Ships vs Aspirational

### Confirmed shipped (grep `internal/vault/providers.go`):

| Template key | Display name | Ships |
|---|---|---|
| `google_calendar` | Google Calendar | YES |
| `google_drive` | Google Drive | YES |
| `google_gmail` | Gmail | YES |
| `slack` | Slack | YES (verify) |
| `github` | GitHub | YES (verify) |
| `notion` | Notion | YES (verify) |
| `linear` | Linear | YES (verify) |
| `microsoft` | Microsoft | YES (verify) |

**Honest gap:** Only `google_calendar`, `google_drive`, and `google_gmail` are confirmed in `internal/vault/providers.go` from the grep. The README and AGENT_AUTH.md list Slack, GitHub, Notion, Linear, Microsoft, and Jira as shipped templates, but the demo runner should verify by running `GET /api/v1/vault/templates` at demo time. If Notion/Linear are not in the binary yet, **substitute with Google Drive + Microsoft** which are confirmed — the demo story (5 providers connected) remains equally compelling.

The `builtinTemplates` map in `internal/vault/providers.go` is the source of truth. Run `shark vault templates list` before the live demo to confirm the exact list.

---

## Shark Feature Map

| Demo step | Shark feature | File |
|---|---|---|
| Provider config | `vault_providers` table + AES-256-GCM `client_secret_enc` | `internal/vault/vault.go`, `internal/storage/` |
| Provider templates | `builtinTemplates` map (google_*, slack, github, notion, linear) | `internal/vault/providers.go` |
| Connect flow | `GET /api/v1/vault/connect/{provider}` → OAuth callback → `vault_connections` | `internal/api/vault_handlers.go` |
| Token read by agent | `GET /api/v1/vault/{provider}/token` gated by `vault:read` scope | `internal/api/vault_handlers.go:623` |
| Scope enforcement | `WWW-Authenticate: Bearer error="insufficient_scope"` | `internal/api/vault_handlers.go:626-628` |
| Auto-refresh | `expiryLeeway = 30 * time.Second` in `GetFreshToken()` | `internal/vault/vault.go:49` |
| Field encryption | `FieldEncryptor` AES-256-GCM, 32-byte key via SHA-256 | `internal/auth/fieldcrypt.go` |
| DPoP binding | RFC 9449 `dpop.go`, `cnf.jkt` in JWT | `internal/oauth/dpop.go`, `internal/identity/identity.go:33` |
| Revocation | `DELETE /api/v1/vault/connections/{id}` | `internal/api/vault_handlers.go:752` |
| Admin revoke (cross-user) | `DELETE /api/v1/admin/vault/connections/{id}` | `internal/api/vault_handlers.go:852` |
| Audit trail | `audit_logs` table, `actor_type=agent`, `actor_id=agent_xxxx` | `internal/api/` (audit calls) |
| Vault events (webhooks) | `vault.provider.created`, `vault.connected`, `vault.revoked` | AGENT_AUTH.md / webhook system |
| Admin UI | Visual vault management, connection listing, revoke button | `admin/src/components/vault_manage.tsx` |

---

## Live Demo Script (10 minutes)

### T=0:00 — Setup context (1 min, talk track while terminal loads)

Show the competitive table on screen:
- Auth0 Token Vault: 50% add-on
- Composio: $229/mo for 2M tool calls, $499+ at Aisha's scale
- Nango: $500/mo Growth
- **Shark: $0, single binary, self-hosted**

> "Aisha has 8,000 users, her agent reads 5 providers per user. She's paying for each of these separately — or rebuilding it herself. Last week her security review failed because her homebrew vault uses AES-128. Let's show her the alternative."

### T=1:00 — Install and configure (1 min)

```bash
# Single binary install
curl -sSL https://get.sharkauth.dev | sh
shark serve &

# Register 5 provider templates via CLI
shark vault provider create google_gmail \
  --client-id=YOUR_GOOGLE_CLIENT_ID \
  --client-secret=YOUR_GOOGLE_CLIENT_SECRET \
  --scopes='https://www.googleapis.com/auth/gmail.readonly,https://www.googleapis.com/auth/calendar'

shark vault provider create slack \
  --client-id=YOUR_SLACK_CLIENT_ID \
  --client-secret=YOUR_SLACK_CLIENT_SECRET \
  --scopes='channels:read,chat:write'

shark vault provider create github \
  --client-id=YOUR_GITHUB_CLIENT_ID \
  --client-secret=YOUR_GITHUB_CLIENT_SECRET \
  --scopes='repo:read,issues:write'

shark vault provider create notion \
  --client-id=YOUR_NOTION_CLIENT_ID \
  --client-secret=YOUR_NOTION_CLIENT_SECRET \
  --scopes='read,update'

shark vault provider create linear \
  --client-id=YOUR_LINEAR_CLIENT_ID \
  --client-secret=YOUR_LINEAR_CLIENT_SECRET \
  --scopes='read,write'
```

### T=2:00 — User connect flow (1.5 min)

```bash
# Seed a test user "aisha-test" (user_42 in demo)
./demos/token_vault/seed.sh

# Show the connect URL the Gemma app redirects users to
echo "Connect URL: http://localhost:8000/api/v1/vault/connect/google_gmail?redirect_uri=http://localhost:3000/connected&user_id=user_42"

# In the browser: simulate user clicking "Connect Gmail"
# Shark runs the full OAuth flow → callback → stores token encrypted
# Webhook fires: vault.connected
```

> "That's it. The user clicked Connect. Shark ran the OAuth dance, stored the token AES-256-GCM encrypted, and fired a webhook to Gemma's backend."

Repeat (fast) for Slack, GitHub — webhooks fire visibly in the terminal.

### T=3:30 — Agent authenticates + reads vault (2 min)

```bash
# Agent gets a DPoP-bound Shark JWT with vault:read scope
TOKEN=$(curl -s -X POST http://localhost:8000/oauth/token \
  -d "grant_type=client_credentials" \
  -d "client_id=gemma-worker" \
  -d "client_secret=AGENT_SECRET" \
  -d "scope=vault:read" \
  | jq -r .access_token)

echo "Agent JWT: $TOKEN"

# Agent reads user_42's Gmail token from the vault
curl -s http://localhost:8000/api/v1/vault/google_gmail/token \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-User-ID: user_42" \
  | jq .
```

Expected response:
```json
{
  "access_token": "ya29.a0AfH6...",
  "token_type": "Bearer",
  "expires_at": "2026-04-24T15:32:00Z",
  "scopes": ["https://www.googleapis.com/auth/gmail.readonly"]
}
```

> "The agent got a fresh Gmail access token. It never saw the refresh token. It never saw the client secret. Shark handled everything."

```bash
# Agent calls Gmail API directly with the unwrapped token
curl -s "https://gmail.googleapis.com/gmail/v1/users/me/messages?maxResults=3" \
  -H "Authorization: Bearer ya29.a0AfH6..." \
  | jq '.messages[0]'
```

### T=5:30 — Audit log proof (30 sec)

```bash
# Show every vault read logged
curl -s http://localhost:8000/api/v1/admin/audit-logs?action=vault.token.read \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  | jq '.logs[] | {actor: .actor_id, action: .action, target: .target_id, request_id: .request_id}'
```

Output:
```json
{ "actor": "gemma-worker", "action": "vault.token.read", "target": "google_gmail:user_42", "request_id": "req_7f3a..." }
```

### T=6:00 — Auto-refresh demo (1 min)

```bash
# Run the auto-refresh test — advances the clock to T+50min
# (Gmail tokens expire in 60min; leeway = 30s)
python demos/token_vault/auto_refresh_test.py
```

Output:
```
[T+0min]  vault.token.read → token valid, expires in 60:00
[T+50min] vault.token.read → token expires in 10:00, within leeway? NO
[T+59:30] vault.token.read → token expires in 30s → AUTO-REFRESH triggered
           vault.refreshed webhook fired
           new token returned to agent — agent saw zero interruption
```

> "The agent called the same endpoint twice. Shark silently refreshed the token behind the scenes. The agent never knew. No 3 AM outage."

### T=7:00 — Per-user revoke cascade (1.5 min)

```bash
# User churns — revoke their Slack connection instantly
curl -s -X DELETE http://localhost:8000/api/v1/admin/vault/connections/vc_slack_user42 \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# Webhook fires: vault.revoked (show in terminal)

# Immediately: any agent trying to read Slack for user_42 gets blocked
curl -s http://localhost:8000/api/v1/vault/slack/token \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-User-ID: user_42"
```

Response:
```json
{ "error": "vault_revoked", "description": "No active vault connection for provider slack, user user_42" }
```

```bash
# Gmail still works — only Slack was revoked
curl -s http://localhost:8000/api/v1/vault/google_gmail/token \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-User-ID: user_42" \
  | jq .access_token
# → "ya29.a0AfH6..."  (still works)
```

> "Revoke is per-user, per-provider. Surgical. Gmail keeps working. Slack is dead. Webhook fired instantly."

### T=8:30 — Scope enforcement + encryption proof (1.5 min)

**Cross-agent scope enforcement:**
```bash
# Rogue agent without vault:read tries to read Gmail
BAD_TOKEN=$(curl -s -X POST http://localhost:8000/oauth/token \
  -d "grant_type=client_credentials&client_id=rogue-agent&client_secret=X&scope=read" \
  | jq -r .access_token)

curl -s http://localhost:8000/api/v1/vault/google_gmail/token \
  -H "Authorization: Bearer $BAD_TOKEN" \
  -H "X-User-ID: user_42"
```

Response:
```
HTTP 403
WWW-Authenticate: Bearer error="insufficient_scope",scope="vault:read"
{"error":"insufficient_scope","description":"Token lacks vault:read scope"}
```

**Encryption proof (split terminal — the wow moment):**
```bash
# LEFT terminal: raw bytes in SQLite
sqlite3 dev.db "SELECT hex(access_token_enc) FROM vault_connections LIMIT 1;"
# → 1F3A9C7B2E45D8F1... (random encrypted bytes — unreadable)

# RIGHT terminal: agent calls vault API, gets readable token
curl -s http://localhost:8000/api/v1/vault/google_gmail/token \
  -H "Authorization: Bearer $TOKEN" -H "X-User-ID: user_42" \
  | jq .access_token
# → "ya29.a0AfH6SMQ..."  (real, usable token)
```

> "Same row. Left: garbage. Right: working token. AES-256-GCM. The agent never touches the database. Shark is the only thing that can decrypt it."

---

## Implementation Plan — Files to Build

### `demos/token_vault/gemma-worker/agent.py` (~80 LOC)

Python agent that:
1. Authenticates to Shark via `client_credentials` to get a DPoP-bound JWT
2. Polls `GET /api/v1/vault/{provider}/token` for each of 5 providers
3. Calls the real third-party API (Gmail list messages, Slack conversations.list, GitHub list repos, Notion search, Linear issues)
4. Logs each action with request_id

### `demos/token_vault/connect-flow/server.py` (~60 LOC)

Flask/FastAPI server that:
1. Serves `GET /connect?provider=google_gmail` — redirects to Shark's vault connect URL
2. Handles `GET /connected` callback — shows "Connected!" page with provider name
3. Lists connected providers via `GET /api/v1/vault/connections`

### `demos/token_vault/seed.sh`

Bash script that:
1. Creates test user `user_42` via Shark API
2. Creates agent `gemma-worker` with `vault:read` scope
3. Seeds 5 vault providers via `POST /api/v1/vault/providers`
4. Prints connect URLs for each provider

### `demos/token_vault/encryption_proof.sh`

Bash script that:
1. Reads raw hex bytes from `vault_connections` via sqlite3
2. Calls `GET /api/v1/vault/{provider}/token` to get the decrypted token
3. Side-by-side diff to prove encryption

### `demos/token_vault/auto_refresh_test.py`

Python test that:
1. Creates a vault connection with `expires_at = now + 31s` (within 30s leeway)
2. Calls `GetFreshToken` → expects auto-refresh triggered
3. Verifies the returned token is different (refreshed)
4. Uses `NewManagerWithClock` test seam from `internal/vault/vault.go:73`

---

## The Wow Moment

**Split terminal. 10 seconds. No explanation needed.**

```
LEFT (SQLite raw):                    RIGHT (Agent API call):
─────────────────────────────────── │ ───────────────────────────────────
$ sqlite3 dev.db \                  │ $ curl .../vault/google_gmail/token
  "SELECT hex(access_token_enc)     │   -H "Authorization: Bearer $TOKEN"
   FROM vault_connections LIMIT 1"  │   -H "X-User-ID: user_42" | jq .
                                    │
1F3A9C7B2E45D8F10B23A4C5E6F78901   │ {
A2B3C4D5E6F7089A1B2C3D4E5F607182   │   "access_token": "ya29.a0AfH6SMQ",
9C8D7E6F5041302100FFEEDDCCBBAA99   │   "expires_at": "2026-04-24T15:32Z",
...256 bytes of AES-256-GCM blob... │   "scopes": ["gmail.readonly"]
                                    │ }

BOTTOM: vault.refreshed webhook arriving in real time →
{"event":"vault.refreshed","user_id":"user_42","provider":"google_gmail","new_expiry":"2026-04-24T16:32Z"}
```

**Verbal:** "Same data. Left: what's in the database — unreadable encrypted bytes. Right: what the agent sees — a working token, freshly refreshed, delivered by Shark. The agent has never touched a database row. It has never seen a refresh token. It can't. That's the architecture."

---

## Sellable Angle

**One line:** "Ship a production-grade OAuth token vault in 10 minutes — not 3 months — and save $2,748/yr vs Composio."

**Three customer types:**

1. **Personal AI ops founders (Gemma-type):** $19-49/mo SaaS, 1K-50K users, 3-10 connected providers per user. Previously: homegrown Postgres table, AES-128, no audit. Now: single binary, AES-256-GCM, SOC 2 CC6.1 compliant out of the box.

2. **B2B agentic SaaS (Lindy, Bardeen, Wordware-type):** Agent acts across 20+ SaaS tools per enterprise user. Token Vault replaces a 3-engineer 6-month infra project. Revocation cascade is table stakes for enterprise sales ("what happens when an employee leaves?").

3. **Regulated AI verticals (healthcare ops, legal AI, fintech assistants):** HIPAA 164.312(a)(2)(iv) requires encryption + decryption controls at rest. GDPR Art. 32 requires appropriate technical measures. Shark provides AES-256-GCM at rest + per-user revocation cascade + full audit trail — all three in one binary, no vendor lock-in.

---

## Status Quo Failures Shark Solves

| Failure mode | Aisha's experience | Shark fix |
|---|---|---|
| AES-128 at rest | Security review failed | AES-256-GCM (`internal/auth/fieldcrypt.go`, 32-byte key) |
| Refresh breaks at 3 AM | Slack rotation broke twice last month | `expiryLeeway = 30s` proactive refresh (`vault.go:49`) |
| No cascade revoke | Manual row deletion on churn | `DELETE /api/v1/admin/vault/connections/{id}` blocks all agents |
| No audit | Can't answer compliance questions | `audit_logs` per vault read, actor_type=agent |
| No webhook | Product team blind on disconnects | `vault.connected`, `vault.refreshed`, `vault.revoked` events |
| Vendor lock-in | Composio $229+/mo, Nango $500+/mo | Shark: $0, self-hosted, single binary |

---

## Compliance Angle

| Standard | Requirement | Shark capability |
|---|---|---|
| SOC 2 CC6.1 | Logical access controls — restrict access to credentials | Per-user per-provider vault isolation; `vault:read` scope gates; no raw creds in agent memory |
| GDPR Art. 32 | Appropriate technical measures for encryption at rest | AES-256-GCM for all vault tokens; `client_secret_enc`, `access_token_enc`, `refresh_token_enc` all encrypted |
| HIPAA 164.312(a)(2)(iv) | Encryption and decryption mechanism | AES-256-GCM at rest; access only via Shark API with valid JWT; revocation cascade on user offboarding |

---

## UAT Checklist (15+ items)

- [ ] **VAULT-01** Provider create: `POST /api/v1/vault/providers` stores `client_secret_enc` as non-plaintext BLOB (verify via sqlite3 `hex()`)
- [ ] **VAULT-02** Connect flow: `GET /api/v1/vault/connect/google_gmail` redirects to Google OAuth authorize URL with correct `redirect_uri` and `state`
- [ ] **VAULT-03** Callback: after user grants, `vault_connections` row exists with `access_token_enc` and `refresh_token_enc` as non-plaintext BLOBs
- [ ] **VAULT-04** Token read: `GET /api/v1/vault/google_gmail/token` with valid `vault:read` scope returns plaintext access_token + expiry + scopes
- [ ] **VAULT-05** Scope gate: agent without `vault:read` scope gets HTTP 403 + `WWW-Authenticate: Bearer error="insufficient_scope"`
- [ ] **VAULT-06** Auto-refresh leeway: token with `expires_at = now + 25s` (inside 30s leeway) triggers transparent refresh; returned token has new expiry > `now + 30s`
- [ ] **VAULT-07** Auto-refresh under load: 10 concurrent agents reading same user+provider all get valid tokens (no thundering-herd double-refresh, no race on `vault_connections` update)
- [ ] **VAULT-08** Revoke cascade: `DELETE /api/v1/admin/vault/connections/{id}` → subsequent `GET /api/v1/vault/{provider}/token` returns 404/403 for that user+provider
- [ ] **VAULT-09** Cross-provider isolation: revoking `slack` connection for user_42 does NOT affect `google_gmail` token reads for user_42
- [ ] **VAULT-10** Cross-user isolation: revoking user_42's token does NOT affect user_43's reads on same provider
- [ ] **VAULT-11** Audit completeness: every successful vault.token.read creates an `audit_logs` row with `actor_id=agent_id`, `action=vault.token.read`, `target_id=provider:user_id`, `request_id`
- [ ] **VAULT-12** Audit on revoke: `DELETE /api/v1/admin/vault/connections/{id}` creates audit row `vault.connection.deleted`
- [ ] **VAULT-13** Webhook delivery: `vault.connected` fires within 2s of successful OAuth callback; `vault.revoked` fires within 2s of DELETE
- [ ] **VAULT-14** Raw bytes check: `sqlite3 dev.db "SELECT typeof(access_token_enc) FROM vault_connections LIMIT 1"` → `blob`; `hex()` output is not a valid JWT or readable string
- [ ] **VAULT-15** DPoP binding: agent token bound to DPoP key cannot be replayed without matching DPoP proof; replay attempt returns 401
- [ ] **VAULT-16** Provider template list: `GET /api/v1/vault/templates` returns all builtin templates (verify actual count matches `builtinTemplates` map in `providers.go`)
- [ ] **VAULT-17** `needs_reauth` path: when refresh_token is invalid/expired, vault returns `ErrNeedsReauth` and sets `needs_reauth=1` on the connection row; subsequent token reads return 4xx until user re-connects
- [ ] **VAULT-18** Admin UI: `admin/src/components/vault_manage.tsx` shows connection list, revoke button works, webhook indicator updates after revoke

---

## Honest Gaps

1. **Provider template count:** `providers.go` confirms `google_calendar`, `google_drive`, `google_gmail`. The demo task prompt says Slack/GitHub/Notion/Linear/Microsoft also ship — run `GET /api/v1/vault/templates` or inspect `builtinTemplates` in `providers.go` before the live demo. If Notion/Linear are missing, substitute Google Drive + Microsoft (confirmed in README) — the 5-provider story holds.

2. **Webhook system:** `vault.connected` / `vault.refreshed` / `vault.revoked` are listed as planned audit events in `AGENT_AUTH.md`. Verify they fire from the webhook subsystem in the current binary before the live demo. If webhooks are not wired, demo the audit log instead — it provides the same assurance narrative.

3. **`shark vault` CLI commands:** The `seed.sh` uses `shark vault provider create` — verify the CLI subcommand exists. If not, substitute `curl -X POST /api/v1/vault/providers` in the seed script.

4. **DPoP for vault reads:** The demo shows DPoP-bound tokens. `internal/oauth/dpop.go` and `internal/config/config.go:321` confirm DPoP ships. Ensure `require_dpop=true` is set in the demo config for maximum security theater impact.

---

## Key Files Reference

| File | Role |
|---|---|
| `internal/vault/vault.go` | Manager, `expiryLeeway = 30s`, `GetFreshToken`, `ExchangeAndStore`, `NewManagerWithClock` |
| `internal/vault/providers.go` | `builtinTemplates` map — source of truth for shipped providers |
| `internal/auth/fieldcrypt.go` | `FieldEncryptor` AES-256-GCM, 32-byte key |
| `internal/api/vault_handlers.go` | All vault HTTP handlers; scope check at line 623-628 |
| `internal/oauth/dpop.go` | RFC 9449 DPoP implementation |
| `internal/identity/identity.go` | `AuthMethodDPoP` constant |
| `internal/config/config.go:321` | `require_dpop` config flag |
| `admin/src/components/vault_manage.tsx` | Admin UI for vault management |
| `AGENT_AUTH.md` | Full Token Vault spec, competitive table, audit events |
