# Demo #3 — Headless Coding Agent on a CI Runner
## RFC 8628 Device Flow + DPoP-Bound Scoped Token, No Browser, No Service Account Sprawl

**Audience:** Platform leads, DevSecOps, AI tooling teams  
**Runtime:** 10 minutes live  
**Format:** Terminal split-screen + phone QR scan + Slack side-panel  

---

## Persona

**Marcus, Platform Lead at DriftCo**

DriftCo runs 30 self-hosted GitHub Actions runners across 4 AWS regions (us-east-1, eu-west-1, ap-southeast-1, sa-east-1). Their AI deploy agent — a custom Devin-style pipeline — does five things per CI run:

1. Reads the target repo to plan a build
2. Builds a Docker image
3. Pushes to staging ECR
4. Runs smoke tests against `staging.drift.co`
5. Posts a PR comment with deploy summary + link

**The postmortem story (IR followup, happened last quarter):**

> A misconfigured CloudWatch log group was shipping raw environment variables to a third-party log aggregator. Runner env included `DRIFTCO_DEPLOY_PAT` — a GitHub PAT with `admin:org` scope, baked into every runner image. The aggregator had a public S3 export bucket. Time to discovery: 11 days. Blast radius: full org-level GitHub access — every repo, every secret, every package. Remediation required rotating every secret in the org and auditing 3 months of commit history.

**The IR followup requirement:** Replace PAT-based CI auth with per-run, scoped, short-lived, user-bound tokens with full audit trail per deploy. Deadline: before next quarterly pentest. Marcus has 2 weeks.

---

## Status Quo Failures

### 1. GitHub Personal Access Tokens (PATs)
- **Scope problem:** Minimum useful PAT for a deploy agent is `repo` + `write:packages` — that's full repo read/write on every repo the user owns. No granular "write only to `org/api` repo" option until fine-grained PATs, which many orgs haven't migrated to.
- **Lifetime problem:** Classic PATs are non-expiring by default. Fine-grained PATs max at 1 year. A PAT baked into a runner image from 18 months ago is still valid.
- **Audit problem:** GitHub audit log shows PAT usage at the org level, but not which CI run, which runner, or which agent invocation. Zero per-deploy traceability.
- **Blast radius:** One PAT leak = every resource that user could access. Per GitHub's own 2023 security transparency report, stolen tokens (PATs + OAuth tokens) account for ~46% of unauthorized access incidents on the platform.
- **Rotation pain:** Rotating a PAT means updating it in every runner, every secret store, every CI config. Teams defer rotation indefinitely.

### 2. GitHub Actions OIDC Tokens (current best-in-class for cloud CI)
- **What it does well:** Issues short-lived OIDC tokens that AWS/GCP/Azure can federate — no long-lived cloud credentials needed.
- **The gap:** OIDC tokens are cloud-vendor-specific. They cannot be used to authenticate arbitrary AI tooling, internal APIs, staging deploy services, or any system that isn't an AWS/GCP/Azure IAM endpoint. They're also bound to the GitHub Actions environment — useless on self-hosted runners outside the Actions ecosystem, on Devin/OpenHands/Aider running in a Docker container, or on kiosk/POS terminals.
- **No human binding:** OIDC tokens identify the workflow, not a human approver. There's no "Marcus approved this deploy" — just "workflow `deploy.yml` ran."
- **No scope narrowing to resources:** You get a token for a cloud role, not a token scoped to `repo:write:driftco/api` vs `repo:write:driftco/infra`.

### 3. Shared API Keys / Service Account Tokens
- Every agent shares the same credential → no per-agent audit trail
- Keys are static → rotation is manual → never rotated
- No human-in-the-loop approval → compliance nightmare for regulated industries
- Blast radius identical to PATs

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         DriftCo CI Runner (headless)                        │
│                                                                             │
│  agent.py                                                                   │
│  ┌─────────────────────────────────────────────────────┐                   │
│  │ 1. DPoPProver.generate()  → fresh ECDSA P-256 keypair│                  │
│  │    (private key NEVER leaves this runner process)    │                  │
│  │                                                      │                  │
│  │ 2. POST /oauth/device                               │                   │
│  │    client_id=ci-deploy-agent                         │                  │
│  │    scope=repo:write:driftco/api deploy:staging       │                  │
│  │    resource=https://deploy.drift.co     [RFC 8707]   │                  │
│  │                                                      │                  │
│  │ 3. Print QR + user_code to runner log               │                   │
│  │    Polls POST /oauth/token every 5s  [RFC 8628 §3.5] │                  │
│  └─────────────────────────────────────────────────────┘                   │
└───────────────────────────────┬─────────────────────────────────────────────┘
                                │ POST /oauth/device  (RFC 8628 §3.1)
                                │ POST /oauth/token   (RFC 8628 §3.5 + RFC 9449)
                                ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         SharkAuth (single binary)                           │
│                                                                             │
│  internal/oauth/device.go    — RFC 8628 device authorization endpoint       │
│  internal/oauth/dpop.go      — RFC 9449 DPoP proof validation + jkt bind    │
│  internal/oauth/audience.go  — RFC 8707 resource indicator enforcement      │
│  internal/oauth/store.go     — refresh family tracking + reuse detection    │
│  internal/api/webhook_emit.go — async webhook fanout                        │
│  admin/src/hosted/routes/    — branded approval page                        │
│                                                                             │
│  Device codes: SHA-256 hashed at rest, 15-min lifetime                      │
│  User codes: XXXX-XXXX format, unambiguous charset (no I/O/0/1)             │
│  Polling interval: 5s, slow_down on violation                               │
│  Access token: ES256 JWT, 15 min TTL, cnf.jkt bound                        │
│  Refresh token: opaque, family-tracked, rotation on every use               │
└───────┬────────────────────────────────────┬────────────────────────────────┘
        │ verification_uri_complete           │ webhook: agent.device_authorized
        │ (QR scan → phone)                  ▼
        ▼                        ┌─────────────────────────┐
┌─────────────────────────┐      │   ops Slack #deploys    │
│  Marcus's phone         │      │                         │
│  /oauth/device/verify   │      │  "device authorized:    │
│  Branded DriftCo page   │      │   marcus@driftco for    │
│  Shows exact scopes     │      │   ci-deploy-agent       │
│  [Approve] [Deny]       │      │   scope=repo:write:...  │
└──────────┬──────────────┘      │   expires_at=..."       │
           │ POST approve        └─────────────────────────┘
           ▼
  device code → authorized
  (agent poll returns token)
        │
        ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                    staging deploy API  (deploy-target/server.py)            │
│                                                                             │
│  POST /deploy                                                               │
│  Authorization: DPoP eyJ...                                                 │
│  DPoP: eyJ...   (proof signed by agent's ECDSA key, ath=SHA256(access_tok)) │
│                                                                             │
│  Middleware verifies:                                                       │
│    - JWT signature (ES256, JWKS from /.well-known/jwks.json)               │
│    - aud == https://deploy.drift.co  (RFC 8707)                            │
│    - scope includes deploy:staging                                          │
│    - cnf.jkt == SHA256(DPoP public key thumbprint)  (RFC 9449)             │
│    - DPoP proof htm/htu/iat/jti                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Shark Feature Map

| Demo Step | RFC | Shark File | What It Does |
|-----------|-----|------------|--------------|
| ECDSA keypair generation | RFC 9449 §2 | `sdk/python/shark_auth/dpop.py` | `DPoPProver.generate()` — P-256 key, never leaves runner |
| Device authorization request | RFC 8628 §3.1 | `internal/oauth/device.go` `HandleDeviceAuthorization` | Issues `device_code` + `user_code`, stores SHA-256 hash |
| User code format (XXXX-XXXX) | RFC 8628 §6.1 | `internal/oauth/device.go` `generateUserCode()` | Unambiguous charset: no I, O, 0, 1 |
| Verification URI + QR | RFC 8628 §3.2 | `internal/oauth/device.go` deviceAuthResponse struct | `verification_uri_complete` = base + `?user_code=` |
| Branded approval page | — | `admin/src/hosted/routes/` | Per-tenant branding, scope display, approve/deny |
| Polling + slow_down | RFC 8628 §3.5 | `internal/oauth/device.go` `HandleDeviceToken` | Returns `authorization_pending` / `slow_down` / token |
| DPoP proof at token endpoint | RFC 9449 §4.3 | `internal/oauth/dpop.go` `ValidateDPoPProof` | Validates alg, htm, htu, iat window, JTI replay |
| cnf.jkt in JWT | RFC 9449 §6 | `internal/oauth/handlers.go` (DX1 enrichment block) | Embeds `cnf.jkt` thumbprint in access token claims |
| Audience binding | RFC 8707 | `internal/oauth/audience.go` | Token only valid for `resource=https://deploy.drift.co` |
| Scope narrowing | OAuth 2.1 §5 | `internal/oauth/device.go` scope intersection | Granted scopes ⊆ registered allowed scopes |
| Local token verify | RFC 7519 | `sdk/python/shark_auth/decode_agent_token()` | JWKS fetch+cache, sig verify, exp/aud/iss check — no round-trip |
| Refresh token rotation | OAuth 2.1 §4.3 | `internal/oauth/store.go` family_id tracking | New token per use; reuse triggers family revoke |
| Webhook on approval | — | `internal/api/webhook_emit.go` | `agent.device_authorized` fires async to Slack |
| DPoP theft rejection | RFC 9449 §11.1 | `internal/oauth/dpop.go` `ValidateDPoPProof` | Different runner = different key = jkt mismatch → 401 |
| Admin Device Flow tab | — | `admin/src/components/device_flow.tsx` | Live pending flows, approve/deny, status |
| Audit row per deploy | — | `internal/audit/audit.go` | `actor`, `on_behalf_of`, `action`, `resource` |

---

## Live Demo Script

### Pre-flight (30 seconds before audience arrives)

```bash
# Terminal left: SharkAuth running
./sharkauth serve --config sharkauth.yaml

# Terminal right: Slack webhook mock
python demos/headless_ci/slack-webhook-mock.py --port 9999

# Terminal center: blank — this is the demo runner
```

### Step 1 — Seed the client (seed.sh, 20 sec)

```bash
bash demos/headless_ci/seed.sh
```

`seed.sh` does:
```bash
#!/usr/bin/env bash
# Register ci-deploy-agent via DCR
curl -s -X POST http://localhost:8080/oauth/register \
  -H "Content-Type: application/json" \
  -d '{
    "client_name": "ci-deploy-agent",
    "grant_types": ["urn:ietf:params:oauth:grant-type:device_code","refresh_token"],
    "scope": "repo:write:driftco/api deploy:staging",
    "token_endpoint_auth_method": "client_secret_basic",
    "logo_uri": "https://driftco.example/logo.png",
    "tos_uri": "https://driftco.example/tos",
    "contacts": ["marcus@driftco.example"]
  }' | tee /tmp/agent_creds.json

# Configure Slack webhook
curl -s -X POST http://localhost:8080/api/v1/admin/webhooks \
  -H "X-Shark-Admin-Key: $SHARK_ADMIN_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "http://localhost:9999/slack",
    "events": ["agent.device_authorized","agent.token_revoked"],
    "secret": "demo-hmac-secret"
  }'

echo "[seed] done — client_id=$(jq -r .client_id /tmp/agent_creds.json)"
```

### Step 2 — Runner startup: keypair + device request (30 sec)

```bash
python demos/headless_ci/runner/agent.py \
  --auth-url http://localhost:8080 \
  --client-id ci-deploy-agent \
  --client-secret "$(jq -r .client_secret /tmp/agent_creds.json)" \
  --scope "repo:write:driftco/api deploy:staging" \
  --resource https://deploy.drift.co \
  --target http://localhost:7777/deploy
```

Runner output (exactly as audience sees it):

```
[ci-deploy-agent] startup @ 2026-04-24T18:30:00Z
[crypto] generated ECDSA P-256 keypair — private key in process memory only
[device] POST /oauth/device ...
[device] user_code     : WDJB-MJHT
[device] expires_in    : 900s
[device] scan this QR or visit:
         http://localhost:8080/oauth/device/verify?user_code=WDJB-MJHT

██████████████████████████████
█ ▄▄▄▄▄ █▀ █▀▀▀█▀ ▄▄▄▄▄ █
█ █   █ █▄▀ ▄▀█ ▀ █   █ █
█ █▄▄▄█ █ ▄▄▄ █ ▀ █▄▄▄█ █
█▄▄▄▄▄▄▄█ ▀ █ ▀ █▄▄▄▄▄▄▄█
... (qr_print.py renders inline)

[poll] waiting for approval (interval=5s)...
```

### Step 3 — Marcus scans on phone (20 sec)

Marcus scans with phone camera. Sees DriftCo-branded page:

```
┌─────────────────────────────────────────────┐
│  🦈  DriftCo                                │
│                                             │
│  ci-deploy-agent is requesting access       │
│                                             │
│  Permissions requested:                     │
│  • repo:write:driftco/api                  │
│    Write access to the driftco/api repo     │
│  • deploy:staging                           │
│    Deploy to staging environment            │
│                                             │
│  Token expires: 15 minutes after approval   │
│  Resource: https://deploy.drift.co          │
│                                             │
│  ┌──────────┐    ┌──────────┐              │
│  │  Approve │    │   Deny   │              │
│  └──────────┘    └──────────┘              │
└─────────────────────────────────────────────┘
```

Marcus taps **Approve**.

### Step 4 — Slack fires simultaneously (5 sec)

Slack side-panel (visible on screen) shows webhook POST arriving:

```json
{
  "event": "agent.device_authorized",
  "timestamp": "2026-04-24T18:30:22Z",
  "data": {
    "client_id": "ci-deploy-agent",
    "client_name": "ci-deploy-agent",
    "authorized_by": "marcus@driftco.example",
    "scope": "repo:write:driftco/api deploy:staging",
    "audience": "https://deploy.drift.co",
    "expires_at": "2026-04-24T18:45:22Z",
    "runner_ip": "10.0.1.45",
    "dpop_jkt": "0ZcOCORZNYy-DWpqq30jZyJGHTN0d2HglBV3uiguA4I"
  }
}
```

### Step 5 — Terminal blasts (the wow moment, <12 sec total from QR scan)

```
[poll] authorization_pending...
[poll] authorization_pending...
[+++] token received!

--- ACCESS TOKEN CLAIMS ---
sub      : ci-deploy-agent
act      : {"sub": "marcus@driftco.example"}
scope    : repo:write:driftco/api deploy:staging
aud      : https://deploy.drift.co
iss      : http://localhost:8080
exp      : 1745522122  (15 min from now)
cnf.jkt  : 0ZcOCORZNYy-DWpqq30jZyJGHTN0d2HglBV3uiguA4I
token_type: DPoP
---

[deploy] POST http://localhost:7777/deploy
         Authorization: DPoP eyJhbGci...
         DPoP: eyJhbGci... (proof: htm=POST, htu=.../deploy, ath=SHA256(token))
[deploy] 200 OK — staging deploy accepted
[audit]  actor=ci-deploy-agent on_behalf_of=marcus@driftco.example
         action=deploy:staging resource=https://deploy.drift.co
```

### Step 6 — Attack scenario (2 min)

```bash
# Attacker copies JWT from runner log and tries from different machine
# (no DPoP key — can't make a valid proof)
curl -X POST http://localhost:7777/deploy \
  -H "Authorization: DPoP eyJhbGci..." \
  -H "DPoP: eyJhbGci...FAKE_PROOF..."

# Response:
# HTTP/1.1 401 Unauthorized
# {"error": "invalid_token", "error_description": "DPoP proof jkt does not match cnf claim"}
```

Audience sees: **the token is cryptographically bound to the runner that generated it. Replay from a different machine is impossible.**

### Step 7 — Refresh rotation (1 min)

```bash
# Agent auto-uses refresh token after 15 min (simulated with --fast-expire flag)
# Each rotation is logged — Marcus does NOT re-approve
[refresh] rotating token (family: tok_fam_abc123)
[+++] new access token issued, cnf.jkt unchanged (same keypair)
[audit] actor=ci-deploy-agent action=token_rotated family=tok_fam_abc123
```

### Step 8 — Revocation (1 min)

```bash
shark agent revoke ci-deploy-agent --user marcus@driftco.example

# Or via admin API:
curl -X POST http://localhost:8080/api/v1/admin/agents/ci-deploy-agent/revoke \
  -H "X-Shark-Admin-Key: $SHARK_ADMIN_KEY"

# Output:
# {"revoked": true, "family_tokens_revoked": 3, "event": "agent.token_revoked"}

# Slack side-panel fires:
# {"event": "agent.token_revoked", "client_id": "ci-deploy-agent", ...}

# Next agent poll or API call:
# HTTP/1.1 401 {"error": "invalid_token", "error_description": "token revoked"}
```

---

## Implementation Plan

### Files to create under `demos/headless_ci/`

#### `runner/agent.py` (~70 LOC)
Full demo CI agent. Responsibilities:
- Parse CLI args: `--auth-url`, `--client-id`, `--client-secret`, `--scope`, `--resource`, `--target`
- Instantiate `DPoPProver` from `shark_auth` — generates fresh ECDSA P-256 keypair on startup
- Call `DeviceFlow(auth_url, client_id, client_secret).start(scope=..., resource=...)` — returns `DeviceInit`
- Call `qr_print.print_qr(verification_uri_complete)` for terminal QR rendering
- Poll `device_flow.poll(prover=prover)` with 5s interval; handle `authorization_pending` / `slow_down`
- On token receipt: call `decode_agent_token()` to verify locally, print decoded claims
- Call target deploy API with `Authorization: DPoP <token>` + `DPoP: <proof>` headers
- On 401: print attack-scenario explanation

#### `deploy-target/server.py` (~50 LOC)
Mock staging deploy API. Responsibilities:
- `POST /deploy` handler with DPoP + Bearer middleware
- Fetch JWKS from `http://localhost:8080/.well-known/jwks.json` (cached)
- Verify JWT: signature, `exp`, `aud == https://deploy.drift.co`, `scope` contains `deploy:staging`
- Verify DPoP proof: parse `DPoP` header, check `htm=POST`, `htu` match, `iat` within 60s, `jti` not replayed
- Verify `cnf.jkt` in JWT claims matches SHA-256 thumbprint of DPoP public key
- Return `200 {"status": "deploy accepted", "env": "staging"}` or `401` with RFC 6750 error body
- Emit audit line: `actor= on_behalf_of= action=deploy:staging resource=`

#### `seed.sh`
Idempotent setup script:
- Register `ci-deploy-agent` via `POST /oauth/register` (DCR, RFC 7591)
- Configure allowed scopes: `repo:write:driftco/api`, `deploy:staging`
- Set `token_lifetime=900` (15 min) on the client record
- Configure `may_act` delegation (so act claim is populated)
- Register Slack webhook mock for `agent.device_authorized` + `agent.token_revoked`
- Set branding: DriftCo logo, colors, app name on the device verify page
- Save `client_id` + `client_secret` to `/tmp/agent_creds.json`

#### `slack-webhook-mock.py`
Simple HTTP server:
- Listens on port 9999
- `POST /slack` — verifies HMAC-SHA256 signature (demo-hmac-secret)
- Pretty-prints received payload to stdout with timestamp
- Keeps a `GET /events` SSE endpoint so a side-panel browser tab can stream events live

#### `qr_print.py`
Terminal QR renderer:
- Uses `qrcode` library (pure Python, no C deps)
- Renders as Unicode block characters (works in any modern terminal)
- Falls back to printing raw URL if terminal doesn't support Unicode
- Exported as `print_qr(url: str) -> None`

---

## Honest Gaps

| Gap | Detail | Timeline |
|-----|--------|----------|
| Rate limiter is in-memory | Device polling rate limiter (`10/min per client_id`, `slow_down` enforcement) is stored in a per-process sync.Map. A second SharkAuth replica will have its own counter — a client could double its allowed rate by load-balancing across replicas. | Redis-backed distributed rate limiter planned for Q3 2026 (see SCALE.md) |
| Single-replica only for this demo | The demo runs one SharkAuth process. HA/multi-replica deployment requires shared storage for DPoP JTI replay cache (currently in-memory `DPoPCache`). | Same Q3 2026 track |
| Device code storage | SHA-256 hashed device codes stored in SQLite. SQLite is single-writer. Under high concurrent device flows (>1000 simultaneous), write contention increases. | PostgreSQL backend on roadmap |
| QR code in runner log | Depends on terminal Unicode support. GitHub Actions log viewer strips some Unicode blocks. Fallback: print raw URL only. | Cosmetic; not a security issue |

---

## The Wow Moment

Phone camera → QR → branded DriftCo approval page → Marcus taps Approve → terminal erupts:

```
[+++] token received!
cnf.jkt: 0ZcOCORZNYy-... (bound to THIS runner's key)
[deploy] 200 OK — staging deploy accepted
```

All of this in **under 12 seconds** from QR scan to deploy confirmation. Simultaneously: Slack fires. Audit row lands. The private key never touched the network.

Then the attacker demo: copy-paste the JWT, try to replay → `401 invalid_token` in 200ms. The token is physically impossible to use without the runner's private key.

---

## Sellable Angle

**One-line pitch:** "Device flow gives your AI agent a user-approved, DPoP-bound, scope-narrowed token in under 12 seconds — no PAT, no service account, full audit."

| Customer Type | Their Pain | Shark Solve |
|---------------|-----------|-------------|
| **CI/CD platforms** (GitHub Actions, self-hosted runners, Buildkite) | Shared PATs baked into images, massive blast radius on leak | Per-run device flow: scoped, short-lived, human-approved, audit trail |
| **AI coding agents** (Claude Code, Devin, OpenHands, Aider, Cursor) | Long-lived API keys in env vars, no human approval loop, no audit per action | DPoP-bound tokens issued via device flow: agent acts on behalf of a named human, every action traceable |
| **Kiosk / POS / IoT fleets** | Devices can't run a browser; shared credentials across fleet | Device flow maps perfectly: device polls, operator approves on phone, token scoped to device ID |
| **Regulated industries** (fintech, healthtech, gov) | Compliance requires human-in-the-loop approval for privileged ops, named actor on every audit row | `act.sub = marcus@driftco.example` in every JWT; audit row links machine action to human approver |

---

## UAT Checklist

- [ ] **Device authorization response shape:** `POST /oauth/device` returns all 6 RFC 8628 §3.2 fields: `device_code`, `user_code`, `verification_uri`, `verification_uri_complete`, `expires_in`, `interval`
- [ ] **User code format:** User code matches `[A-Z2-9]{4}-[A-Z2-9]{4}` (no I, O, 0, 1)
- [ ] **Polling: authorization_pending:** Token endpoint returns `{"error":"authorization_pending"}` before user approves
- [ ] **Polling: slow_down compliance:** Polling faster than `interval` (5s) returns `{"error":"slow_down"}` with incremented interval
- [ ] **Polling: success after approval:** Token endpoint returns full token response after Marcus approves
- [ ] **DPoP proof validation:** Token issued as `token_type=DPoP` when DPoP header present at token endpoint
- [ ] **cnf.jkt in JWT:** Decoded access token contains `cnf.jkt` matching SHA-256 thumbprint of DPoP public key
- [ ] **Audience binding:** Token with `aud=https://deploy.drift.co` rejected by deploy API if `aud` doesn't match
- [ ] **Scope narrowing:** Requesting scope not in registered allowed scopes returns `invalid_scope`
- [ ] **Token-theft DPoP rejection:** JWT from runner A + DPoP proof signed by runner B's key → deploy API returns `401 invalid_token` with `cnf.jkt mismatch` detail
- [ ] **Local verify (no introspection):** `decode_agent_token()` verifies signature + `exp`/`aud`/`iss` without calling `/oauth/introspect`
- [ ] **Refresh rotation:** Each refresh token use issues a new access + refresh pair; old refresh token is revoked
- [ ] **Refresh reuse detection:** Presenting an already-rotated refresh token revokes the entire token family
- [ ] **Webhook HMAC verify:** `agent.device_authorized` webhook payload HMAC-SHA256 matches configured secret
- [ ] **Webhook fires on approval:** Slack mock receives `agent.device_authorized` event within 2s of Marcus tapping Approve
- [ ] **Revocation propagates:** After `shark agent revoke`, next token use (at deploy API or introspect) returns `401`
- [ ] **Token family revocation:** Revoking one token in a family revokes all siblings (verified via audit log + API calls)
- [ ] **Branded approval page:** `/oauth/device/verify` renders DriftCo logo, colors, and scopes in human-readable form
- [ ] **Device code expiry:** Approving a code after `expires_in` seconds returns `expired_token` error
- [ ] **Audit row completeness:** Each deploy call produces audit row with `actor`, `on_behalf_of`, `action`, `resource`, `timestamp`
- [ ] **Rate limit on device endpoint:** More than 10 device requests/min per `client_id` returns `429`
- [ ] **QR renders in terminal:** `qr_print.py` outputs scannable QR to stdout; fallback URL prints on exception

---

## Key File References

| File | Role |
|------|------|
| `internal/oauth/device.go` | `HandleDeviceAuthorization` (POST /oauth/device), `HandleDeviceToken` (polling), `generateUserCode()`, `generateDeviceCode()`, slow_down logic |
| `internal/oauth/dpop.go` | `ValidateDPoPProof()`, `DPoPCache` (JTI replay), thumbprint computation |
| `internal/oauth/audience.go` | RFC 8707 resource indicator extraction + enforcement |
| `internal/oauth/handlers.go` | DX1 block: `cnf.jkt` injection into JWT session |
| `internal/oauth/store.go` | `oauth_tokens` table schema, `family_id` column, rotation + reuse detection |
| `internal/api/webhook_emit.go` | `agent.device_authorized` event dispatch |
| `internal/audit/audit.go` | Per-action audit rows |
| `admin/src/components/device_flow.tsx` | Admin: live pending device flows, approve/deny |
| `admin/src/components/agents_manage.tsx` | Admin: agent table with `device_code` grant type |
| `admin/src/hosted/routes/` | Branded hosted pages (device verify, login, etc.) |
| `sdk/python/shark_auth/device_flow.py` | `DeviceFlow`, `DeviceInit`, `TokenResponse` |
| `sdk/python/shark_auth/dpop.py` | `DPoPProver.generate()`, `make_proof()` |
| `sdk/python/shark_auth/__init__.py` | `decode_agent_token()` |

---

*Demo #3 of 5 — SharkAuth Launch Series*  
*Target runtime: 10 minutes | Requires: SharkAuth binary, Python 3.11+, qrcode package, a phone with camera*
