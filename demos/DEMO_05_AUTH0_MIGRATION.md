# DEMO 05 — Auth0 Escape Hatch
## Migrate 50K Users from Auth0 to Self-Hosted Shark in a Weekend, Zero Downtime

---

## 1. The Story — Sam's Friday Afternoon

**Persona:** Sam Chen, VP Engineering at FintechCo  
**Company:** 85-person Series B fintech, 47K monthly active users, Auth0 Enterprise tenant  
**The bill:** $11,200/mo — Auth0 Enterprise with custom domains, org tiers, log streaming, and advanced attack protection  
**The deadline:** Procurement reviewed the contract renewal on Tuesday. CFO sent a Slack: "Stop the Auth0 bleeding before EOQ (5 weeks out)."  
**The fear:** Sam has tried to leave before. Last time, they got 3 weeks in and discovered:
  1. Auth0 exports bcrypt hashes at $2b cost-10. Their DB schema stored `password_hash` as a raw string — no hash-type column. Importing means every user resets their password.
  2. Auth0 issues JWTs with `iss: https://fintechco.us.auth0.com`. Every microservice (12 of them) validates that issuer. Rotating to a new issuer means coordinated deploys across every service, same weekend.
  3. Their "block-bots" Auth0 Rule is 200 lines of JS living in Auth0's UI with no version history and 3 hardcoded API keys inside it. They don't fully understand it.

**Friday 4:30pm:** Sam opens the Shark admin. By Sunday night, FintechCo is running on Shark. Auth0 contract cancelled Monday morning.

---

## 2. Architecture — The Migration Flow

```
FRIDAY EVENING
──────────────
Auth0 Management API
  └─ GET /api/v2/users (paginated)
  └─ GET /api/v2/roles
  └─ GET /api/v2/organizations
  └─ GET /api/v2/logs (last 30 days)
        │
        ▼
auth0-export-50k.json         auth0-roles.csv        auth0-orgs.csv
        │                           │                      │
        └───────────────────────────┴──────────────────────┘
                                    │
                    shark migrate auth0 --input auth0-export-50k.json
                                    │  --map-roles auth0-roles.csv
                                    │  --hash-policy preserve-bcrypt
                                    │
                         SSE progress stream (live in terminal)
                         ✓ 50,000 users   → Shark DB  (bcrypt hash preserved, hash_type="bcrypt")
                         ✓ 12 roles       → Shark RBAC
                         ✓ 8 orgs         → Shark orgs
                         ✓ 3 SAML conns   → Shark SSO
                                    │
SATURDAY — DUAL-ISSUER WINDOW
──────────────────────────────
     Auth0 JWKS still trusted    Shark JWKS also trusted
     https://fintechco.auth0.com  https://auth.fintechco.com
             │                           │
             └─────────┬─────────────────┘
                       │
              Each microservice's JWT validator
              configured with jwks_uris: [auth0, shark]
              (standard multi-issuer JWKS merge — see §5)
                       │
              New logins → Shark issues JWTs
              Existing sessions (Auth0 tokens) still valid
              until expiry (max 24h)
                       │
SUNDAY AFTERNOON — ATOMIC CUTOVER
───────────────────────────────────
              DNS CNAME: auth.fintechco.com → Shark
              (was proxied via Shark in dual mode already — zero gap)
              Remove Auth0 from jwks_uris
              Audit log shows 100% traffic on Shark
                       │
              CANCEL AUTH0 CONTRACT
              $11,200/mo → $0/mo
```

---

## 3. Shark Feature Map

| Migration Step | Shark Feature | Status Today |
|---|---|---|
| Parse Auth0 user export JSON | `POST /api/v1/migrate/auth0` handler | **STUBBED** (`notImplemented`, `router.go:488`) |
| Import users preserving bcrypt hash | `auth.NeedsRehash()` + `hash_type` column | **REAL** — `auth_handlers.go:270-278` |
| Lazy rehash on next login | bcrypt → argon2id upgrade at login time | **REAL** — `auth_handlers.go:270-278` |
| RBAC roles import | `storage.CreateRole`, `AssignRole` | **REAL** — roles CRUD in storage |
| Org import | `storage.CreateOrganization` | **REAL** — org CRUD in storage |
| JWKS endpoint | `GET /.well-known/jwks.json` | **REAL** — `well_known_handlers.go:73` |
| Dual-issuer JWKS (trust both) | Shark proxy `trusted_issuers` config | **REAL** — proxy resolver (`proxy_resolvers.go`) |
| CLI `shark migrate auth0` command | `cmd/shark/cmd/migrate.go` | **DOES NOT EXIST** — no migrate.go in cmd/ |
| SSE progress stream | SSE pattern exists (webhook events) | **STUBBED** — needs migrate handler |
| SAML connection import | SSO connections exist in storage | **REAL** (partially) — SSO CRUD |
| Audit log replay (30-day webhook re-emit) | `webhook_emit.go` + audit log export | **STUBBED** — no replay command |
| Admin "Migrations" tab | `empty_shell.tsx.md` route scaffolded | **STUBBED** — UI shell only |
| `GET /api/v1/migrate/{id}` (job status) | `router.go:489` | **STUBBED** (`notImplemented`) |

**Summary:** The password-handling pipeline is the real magic and it exists today. Everything else is infrastructure that needs to be built for the demo — the migrate handler, the CLI command, the SSE stream, and the progress UI.

---

## 4. Migration Playbook — The Actual Weekend Sequence

### Friday Evening: Export from Auth0

```bash
# Step 1: Get a Management API token
export AUTH0_DOMAIN=fintechco.us.auth0.com
export AUTH0_MGMT_TOKEN=$(curl -s -X POST \
  "https://${AUTH0_DOMAIN}/oauth/token" \
  -d grant_type=client_credentials \
  -d client_id=$AUTH0_CLIENT_ID \
  -d client_secret=$AUTH0_CLIENT_SECRET \
  -d audience="https://${AUTH0_DOMAIN}/api/v2/" \
  | jq -r .access_token)

# Step 2: Export all users (paginated, convert ndjson → json array)
# Auth0 export format: ndjson with fields:
#   user_id, email, email_verified, name, nickname, picture,
#   created_at, updated_at, last_login, logins_count,
#   identities[].connection, app_metadata, user_metadata,
#   multifactor (totp/sms)
# NOTE: password_hash is NOT included in Management API export —
# must use the Auth0 Import/Export Extension or Data Export add-on.
# Auth0 bcrypt hashes: $2a$10$... or $2b$10$... format.

node demos/auth0_migration/export.js > auth0-export-50k.json
# Output: 50,000 users, ~85MB JSON array

# Step 3: Export roles and org memberships
curl -s "https://${AUTH0_DOMAIN}/api/v2/roles" \
  -H "Authorization: Bearer $AUTH0_MGMT_TOKEN" > auth0-roles.json

curl -s "https://${AUTH0_DOMAIN}/api/v2/organizations" \
  -H "Authorization: Bearer $AUTH0_MGMT_TOKEN" > auth0-orgs.json

# Step 4: Export last 30 days of audit logs for webhook replay
curl -s "https://${AUTH0_DOMAIN}/api/v2/logs?q=type:s&per_page=100&from=$(date -d '30 days ago' +%s)000" \
  -H "Authorization: Bearer $AUTH0_MGMT_TOKEN" > auth0-audit-30d.json
```

### Friday Night: Import to Shark

```bash
# Gate 0: Verify Shark is running and healthy
shark health --url https://auth.fintechco.com
# ✓ healthy | DB: ok | version: 1.x.x

# Gate 1: Run import (THIS IS THE DEMO MOMENT)
shark migrate auth0 \
  --url https://auth.fintechco.com \
  --admin-key $SHARK_ADMIN_KEY \
  --input auth0-export-50k.json \
  --roles auth0-roles.json \
  --orgs auth0-orgs.json \
  --hash-policy preserve-bcrypt \
  --batch-size 500 \
  --sse-progress

# Expected output (live SSE stream):
# [00:00] Starting import: 50,000 users, 12 roles, 8 orgs
# [00:03] ████████░░░░░░░░░░░░  5,000/50,000 users  (1,667/s)
# [00:23] ██████████████████░░ 45,000/50,000 users  (1,958/s)
# [01:28] ████████████████████ 50,000/50,000 users  ✓
#         Roles mapped: 12/12
#         Orgs created: 8/8
#         SAML connections: 3/3
#         Failed: 0
#         Duration: 88s

# Gate 2: Spot-check a migrated user
shark user get --email sarah.kim@fintechco-customer.com
# ✓ user found | hash_type: bcrypt | verified: true | roles: [admin, user]

# Gate 3: Verify login works with original Auth0 password
curl -X POST https://auth.fintechco.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"sarah.kim@fintechco-customer.com","password":"OriginalAuth0Password1!"}'
# ✓ 200 OK | session created
# (On next login, bcrypt hash silently upgraded to argon2id — no user impact)
```

### Saturday: Configure Dual-Issuer JWKS

```bash
# Generate and distribute Shark's JWKS endpoint to all microservices
# Auth0 JWKS:  https://fintechco.us.auth0.com/.well-known/jwks.json
# Shark JWKS:  https://auth.fintechco.com/.well-known/jwks.json

# Update each microservice JWT validator config:
# Before (single issuer):
#   jwks_uri: "https://fintechco.us.auth0.com/.well-known/jwks.json"
#   issuer: "https://fintechco.us.auth0.com/"

# After (dual issuer — trusts both during cutover window):
#   jwks_uris:
#     - "https://fintechco.us.auth0.com/.well-known/jwks.json"
#     - "https://auth.fintechco.com/.well-known/jwks.json"
#   trusted_issuers:
#     - "https://fintechco.us.auth0.com/"
#     - "https://auth.fintechco.com/"

# Verify dual-issuer verification works
node demos/auth0_migration/dual-jwks-test.ts
# ✓ Auth0 JWT verified against Auth0 JWKS
# ✓ Shark JWT verified against Shark JWKS
# ✓ Auth0 JWT rejected by Shark JWKS (expected)
# ✓ Shark JWT rejected by Auth0 JWKS (expected)
# ✓ Both JWTs accepted by dual-issuer validator

# Shark proxy configured to route /api/* — new logins get Shark JWTs
# Existing Auth0 sessions (24h max TTL) drain naturally
```

### Saturday Evening: Traffic Cutover

```bash
# Point application's auth.fintechco.com DNS at Shark
# (Shark proxy has been running in parallel since Friday — zero warmup needed)

# Verify traffic is flowing through Shark
shark admin audit-logs --last 1h --event auth.login | head -20
# ✓ 847 logins in last hour, all issuer=auth.fintechco.com

# Gate: confirm 0 Auth0 logins in last 15min
# When clean: remove Auth0 from trusted_issuers
```

### Sunday: Audit Replay + Contract Cancel

```bash
# Replay 30-day auth.login events to SIEM (continuity of audit pipeline)
shark migrate replay-audit \
  --input auth0-audit-30d.json \
  --webhook-url https://siem.fintechco.internal/auth-events \
  --event-type auth.login \
  --dry-run  # verify first
shark migrate replay-audit \
  --input auth0-audit-30d.json \
  --webhook-url https://siem.fintechco.internal/auth-events \
  --event-type auth.login

# Cancel Auth0 contract
# $11,200/mo → $0/mo
```

---

## 5. Demo Script — 10-Minute Live Walkthrough

**Setup before demo:** Shark running locally with empty DB, 50K user fixture pre-generated but NOT yet imported, Auth0 export JSON on disk.

### Minute 0:00 — The Pain (60s)

> "It's Friday 4:30pm. I'm Sam, VP Eng at FintechCo. Auth0 invoice just hit: eleven thousand, two hundred dollars. For the month. That's a hundred and thirty-four thousand dollars a year. Our CFO wants it killed before EOQ, which is in five weeks. The problem? Every time we've tried to leave Auth0 before, we hit the same three walls: password resets for fifty thousand users, JWKS rotation breaking twelve microservices, and our bot-blocking Rule living as a JS blob in Auth0's UI that nobody fully understands. Today I'm going to show you the exit."

### Minute 1:00 — Export (90s)

Show the Auth0 user export JSON (already on disk — don't wait for API calls live):

```bash
wc -l auth0-export-50k.json   # 50,000 lines
head -3 auth0-export-50k.json
```

Show a single record with `password_hash: "$2b$10$..."` — Auth0's bcrypt format.

> "Standard Auth0 export. Fifty thousand users. Every one has a bcrypt hash. The thing everyone tells you is: you have to make users reset their passwords because no two auth systems store bcrypt the same way. That's wrong. Watch."

### Minute 2:30 — The Import (3 min)

```bash
shark migrate auth0 \
  --input auth0-export-50k.json \
  --roles auth0-roles.json \
  --orgs auth0-orgs.json \
  --hash-policy preserve-bcrypt \
  --sse-progress
```

The live SSE progress bar runs. 50K users in ~90 seconds. Terminal shows:
- User count climbing in real time
- Role mappings completing
- Org creation
- SAML connection imports

> "Ninety seconds. Fifty thousand users. All bcrypt hashes preserved exactly as Auth0 stored them. No password reset email sent. No user friction. Let me prove it."

### Minute 5:30 — The Login Test (90s)

```bash
curl -X POST https://localhost:8443/api/v1/auth/login \
  -d '{"email":"sarah.kim@demo.com","password":"OriginalAuth0Pass1!"}'
```

`200 OK`. Session created. Show the JWT — `iss: https://auth.fintechco.com`.

> "Sarah never knew we migrated. Her Auth0 password works. And on her next login, Shark silently upgrades her bcrypt hash to argon2id — our stronger default — without her doing a thing."

### Minute 7:00 — Dual-JWKS Cutover (2 min)

Open `dual-jwks-test.ts` in the terminal. Run it:

```bash
npx tsx demos/auth0_migration/dual-jwks-test.ts
```

Show: Auth0 JWT still verifies. Shark JWT also verifies. Both issuers trusted.

> "During the cutover window — Saturday to Sunday — every microservice trusts both Auth0 JWKS and Shark JWKS. Existing sessions drain. New logins get Shark JWTs. No coordinated deploys. No weekend war room."

Switch to the Shark admin UI: Settings → Keys → JWKS. Show the public key.

### Minute 9:00 — The Bill (60s)

Show a mock invoice graphic: **$11,200/mo → $0/mo**.

> "Single binary. Runs on a $40/mo VPS or your existing Kubernetes cluster. No per-MAU fees. No MAU tiers. No enterprise add-ons. One binary, your data, your server. The Auth0 escape hatch."

---

## 6. Implementation Plan — Files to Build

### `demos/auth0_migration/auth0-fixture-50k.json` (generator)

A Go or Node script that generates a realistic 50K-user Auth0 export fixture:
- Users with realistic names, email domains (gmail, yahoo, corp domains)
- Mix of `password_hash` (bcrypt $2b$10$, 80% of users) and social-only identities (20%)
- App metadata with fintech-realistic fields: `plan`, `kyc_status`, `org_id`
- Role assignments: `admin` (0.5%), `compliance` (5%), `user` (94.5%)
- 8 orgs with realistic names
- 3 SAML connection entries (enterprise customers)

### `demos/auth0_migration/migrate.sh`

Full end-to-end shell script:
1. Health check gate
2. Run `shark migrate auth0` with all flags
3. Spot-check 5 random users via `GET /api/v1/users`
4. Test login for 3 sample users (curl + assert 200)
5. Run dual-jwks-test.ts
6. Print pass/fail summary

### `demos/auth0_migration/dual-jwks-test.ts`

TypeScript test using `jose` library:
- Fetches Auth0 JWKS (real: `https://dev-xxx.us.auth0.com/.well-known/jwks.json`)
- Fetches Shark JWKS (`http://localhost:8443/.well-known/jwks.json`)
- Mints a fake Auth0-signed JWT (using a local key pair that mimics Auth0 format)
- Mints a real Shark JWT (via login)
- Verifies each JWT against a dual-issuer validator (merged JWKS, two trusted issuers)
- Asserts: Auth0 JWT valid ✓, Shark JWT valid ✓, cross-validation correctly rejects ✓

---

## 7. The "Wow" Moments (Ranked)

**Wow #1 — The Live Counter**
50,000 users importing in real time with an SSE progress bar. The terminal shows `49,847/50,000 ✓ (1,923 users/sec)`. Then it stops. Done. Under 2 minutes.

**Wow #2 — The Password That Still Works**
Without any password reset, a user whose bcrypt hash came straight from Auth0 logs in successfully. The audience has been told "you can't do this without password resets" for years. This breaks that assumption live on stage.

**Wow #3 — The $11K Bill Going to Zero**
A single screen: Auth0 invoice ($11,200) crossed out, Shark self-host cost ($0 auth fees — just compute). Series B founders in the audience know this number. It's visceral.

**Wow #4 — The 90s vs. the Weekend**
Auth0's own migration docs say plan for multiple weekends and expect user friction. Shark does it in 90 seconds, Friday evening, and users never notice.

---

## 8. The Sellable Angle

**One-liner:** "Migrate off Auth0 in a weekend, zero downtime, no password resets — and never pay a per-MAU bill again."

### Three Customer Types

1. **Series A-C startups burning $5K-$15K/mo on Auth0 Enterprise**
   Auth0 Enterprise pricing scales brutally past 10K MAU with org tiers, log streaming, and advanced attack protection as add-ons. A Series B with 50K MAU and SSO requirements is routinely at $10-15K/mo. Shark eliminates the bill on Day 1. ROI case writes itself in the CFO deck.

2. **Regulated industries trapped on Auth0 Enterprise for compliance reasons**
   Banks, fintechs, healthtechs that chose Auth0 for SOC2/HIPAA. Now they're learning "compliant" just meant "someone else's shared infrastructure." Shark self-hosted means their user data never leaves their own VPC. Compliance team prefers it. No more "ask Auth0 for a BAA."

3. **AI startups whose Auth0 bill outgrew their revenue**
   AI infrastructure costs scale differently than MAU. A startup with 500K registered users (mostly free tier, low MAU) can still hit Auth0's limits on API calls, log streaming, and M2M tokens — the hidden line items. Shark's M2M API keys, audit log streaming, and SSO are all included. No surprise invoice at the end of a viral month.

---

## 9. UAT Checklist (Pre-Demo)

- [ ] 1. Generate 50K user fixture with expected distribution (40K bcrypt, 10K social-only)
- [ ] 2. `shark migrate auth0` exits 0 on fixture import
- [ ] 3. `GET /api/v1/users?limit=1` returns 50,000 total count
- [ ] 4. 10 randomly sampled users all have `hash_type: "bcrypt"` in DB
- [ ] 5. Login with known bcrypt-hashed test credential returns 200 + valid JWT
- [ ] 6. JWT from successful login has correct `iss`, `sub`, `email` claims
- [ ] 7. On second login, `hash_type` upgrades to `argon2id` (lazy rehash confirmed)
- [ ] 8. All 12 roles present in `GET /api/v1/roles`
- [ ] 9. Role assignments correct: spot-check 3 admin users have `admin` role
- [ ] 10. All 8 orgs present in `GET /api/v1/organizations`
- [ ] 11. SAML connections visible in admin SSO tab
- [ ] 12. Shark JWKS endpoint returns valid JSON with at least one key
- [ ] 13. `dual-jwks-test.ts` passes all 4 assertions
- [ ] 14. SSE progress stream shows real-time counts (verify with `curl -N --no-buffer`)
- [ ] 15. `GET /api/v1/migrate/{id}` returns job status with counts
- [ ] 16. Social-only users (no `password_hash`) imported with `password_hash: null`
- [ ] 17. Re-run import with `--upsert` flag — no duplicate users created
- [ ] 18. Import of malformed record fails gracefully, logged in `failed[]`, import continues
- [ ] 19. Audit log replay dry-run prints expected event count without POSTing
- [ ] 20. `migrate.sh` end-to-end script passes on clean Shark instance

---

## 10. Rollback Plan

Migration is non-destructive to Auth0 (we never delete from Auth0). Rollback is therefore a DNS flip, not a data recovery operation.

### If migration fails mid-import (Friday night)

```
Condition: import job errors out at 30K/50K users
Action:    shark migrate rollback --job-id <id>
           (truncates all users from this job by job_id tag)
           Auth0 still active, zero user impact
Timeframe: 2 minutes
```

### If login is broken after import (Friday night test)

```
Condition: spot-check login returns 401
Action:    Check hash_type in DB — if blank, re-run with --hash-policy preserve-bcrypt
           OR: shark user reset-hash --user-id <id> --force-bcrypt-recheck
           Auth0 still active during investigation
Timeframe: 15 minutes diagnosis, re-import if needed
```

### If dual-JWKS breaks a microservice (Saturday)

```
Condition: A service rejects Shark JWTs after dual-issuer config
Action:    Revert that service's jwks_uris to Auth0-only
           Shark proxy continues running — new logins still work for services that accept both
           Fix the validator config, re-deploy that service
           Auth0 sessions still valid as fallback
Timeframe: Per-service revert is a single config change + redeploy (~5 min)
```

### If cutover DNS flip causes errors (Sunday)

```
Condition: Error rate spikes after pointing auth.fintechco.com at Shark
Action:    Revert DNS CNAME to Auth0 endpoint
           All services fall back to Auth0 within TTL (60s for demo, 5min in prod)
           All 50K users still exist in Shark DB — cutover can be retried
Timeframe: DNS revert: 60 seconds
```

### Nuclear rollback

```
Condition: Everything is broken, board meeting in 2 hours
Action:    Revert all DNS → Auth0
           Auth0 tenant is untouched — nobody deleted anything
           Users can log in via Auth0 as if migration never happened
           Shark import can be retried after post-mortem
Timeframe: < 5 minutes
Note:      This is why we never delete the Auth0 tenant until 30 days post-cutover.
```

---

## 11. What Needs to Be Built for the Demo

### Must-build (blocking demo)

| File | What | Estimate |
|---|---|---|
| `internal/api/migrate_handlers.go` | `POST /api/v1/migrate/auth0` — parse Auth0 JSON, batch insert users with `hash_type` preservation, stream SSE progress, return job ID | 1-2 days |
| `internal/api/migrate_handlers.go` | `GET /api/v1/migrate/{id}` — job status (counts, errors, state) | 0.5 days |
| `cmd/shark/cmd/migrate.go` | `shark migrate auth0` CLI command — wraps the HTTP endpoint, renders progress bar from SSE stream | 1 day |
| `demos/auth0_migration/auth0-fixture-50k.json` | Generator script for realistic 50K user fixture | 0.5 days |
| `demos/auth0_migration/migrate.sh` | End-to-end demo runner | 0.5 days |
| `demos/auth0_migration/dual-jwks-test.ts` | Dual-issuer JWKS verification test | 0.5 days |

### Nice-to-have (for full demo completeness)

| File | What | Estimate |
|---|---|---|
| `cmd/shark/cmd/migrate.go` | `shark migrate replay-audit` sub-command | 1 day |
| Admin UI: Migrations tab | Wire empty shell to real job status API | 1 day |
| `internal/api/migrate_handlers.go` | Rollback endpoint: `DELETE /api/v1/migrate/{id}` | 0.5 days |

### Already works today (no build needed)

- bcrypt hash preservation + lazy argon2id upgrade on next login (`auth_handlers.go:270-278`)
- RBAC role creation and assignment
- Org creation
- JWKS endpoint (`/.well-known/jwks.json`)
- Proxy with multi-issuer trust (proxy_resolvers.go)
- SSO SAML connection storage
- Audit log streaming
- Admin API key auth (gates migrate routes already)

---

## 12. Real Shark File References

| Claim | File | Lines |
|---|---|---|
| Migrate routes stubbed as `notImplemented` | `internal/api/router.go` | 488-489 |
| bcrypt lazy-rehash on login (REAL) | `internal/api/auth_handlers.go` | 270-278 |
| `auth.NeedsRehash()` function | `internal/auth/` (called at 270) | — |
| `hash_type` column in user storage | `internal/api/auth_handlers.go` | 275 |
| JWKS handler (REAL) | `internal/api/well_known_handlers.go` | 73-114 |
| Proxy multi-issuer resolver (REAL) | `internal/api/proxy_resolvers.go` | 33+ |
| Audit log export (REAL) | `internal/api/router.go` | 482 |
| Admin "Migrations" tab (shell only) | `documentation/inner_docs/admin/src/components/empty_shell.tsx.md` | — |
| No `migrate.go` in CLI | `cmd/shark/cmd/` | (absent) |
| Webhook emit (REAL, basis for replay) | `internal/api/webhook_emit.go` | — |
| README confirms bcrypt migration design | `README.md` (features section) | — |
