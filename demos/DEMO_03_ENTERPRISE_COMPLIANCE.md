# DEMO 03 — HIPAA/SOC2 B2B Multi-Tenant: SSO + RBAC + Full Audit Trail
### "Without Auth0/Okta — Single Binary, Zero License Fee, Full Compliance Posture"

---

## 1. Story — The Persona

**Maya Chen, CTO @ Acme Clinical**
Acme Clinical builds a SaaS platform for managing Phase II/III clinical trials: site coordinators enter ePRO data, Principal Investigators review safety signals, monitors audit data queries. Series B, 35-person team, $4M ARR.

**The Deadline:**
St. Mary's Hospital Procurement wants a signed BAA (Business Associate Agreement) and a completed HIPAA technical safeguards questionnaire by **May 15**. Their CISO checklist demands:

1. SAML SSO from St. Mary's Okta (not a new IdP — their existing one)
2. MFA enforced on every route that touches PHI
3. Role-based access: Principal Investigator (PI), Coordinator, Monitor — each with different PHI read/write privileges
4. Audit log export for the last 12 months, actor/action/IP/timestamp, JSON or CSV
5. Alerting when a privileged action occurs (Slack #soc-alerts)
6. Evidence that encryption keys rotate on a schedule

**The Cost Before Shark:**
- Auth0 Enterprise B2C (SSO addon): ~$10,000–$23,000/mo for their MAU tier
- Okta Workforce: ~$6/user/mo + Professional Services for SAML config + custom audit pipeline
- WorkOS: $125/SSO connection + $125/Directory Sync connection × N enterprise customers = $250+/customer/mo, no per-route MFA enforcement, no built-in proxy rules, audit log via separate integration
- Build it yourself: 3-6 months of eng time + ongoing compliance audits

**Auth0's actual audit log retention: 30 days on most plans. 90 days on Enterprise+. 12 months requires custom log streaming to Datadog/Splunk — another $2K–$8K/mo.**

Maya's quote (composite from real HN/Reddit pain): *"Our enterprise customer demanded we produce audit logs of every PHI access for the last 12 months. Auth0 only kept 30 days. We had to build a bespoke log-streaming pipeline to satisfy the auditor. That cost us 2 months of eng time and $4K/mo in Datadog."*

---

## 2. Architecture

```
┌──────────────────────────────────────────────────────────────────────┐
│  Org: "St. Mary's" (tenant boundary in SharkAuth SQLite DB)          │
│                                                                      │
│  SSO Connection: SAML SP ←→ St. Mary's Okta (IdP)                  │
│  Domain auto-routing: @stmarys.edu → saml_conn_stmarys              │
│                                                                      │
│  RBAC Matrix:                                                        │
│    Role: PI         → permissions: phi:read, phi:write, study:admin  │
│    Role: Coordinator → permissions: phi:read, data:entry             │
│    Role: Monitor    → permissions: phi:read (read-only audit)        │
│                                                                      │
│  Proxy Rules (per-route enforcement):                                │
│    path: /api/phi/*                                                  │
│    require: role=PI, mfa_passed=true                                 │
│    on_deny: audit(action=proxy.deny) + webhook(soc-alerts)           │
│                                                                      │
│  Audit Pipeline:                                                     │
│    Every event → internal/audit.Log() → SQLite                      │
│    Real-time: audit.Log() → webhook.Dispatcher.Emit(system.audit_log)│
│    Export: GET /api/v1/admin/audit?from=2025-01-01&format=csv        │
│                                                                      │
│  Webhooks: HMAC-SHA256 signed → Slack #soc-alerts                   │
│    Retry schedule: 1m, 5m, 30m, 2h, 12h                             │
│    Dead-letter queue + manual replay                                 │
│                                                                      │
│  MFA: TOTP (Google Authenticator) + Passkeys (WebAuthn)             │
│  Signing keys: ES256 JWKS, RotateSigningKeys() tx, dual-window      │
└──────────────────────────────────────────────────────────────────────┘
```

**Single binary. SQLite on disk. Zero external dependencies. Deploy with `./sharkauth serve`.**

---

## 3. Shark Feature Map

| Compliance Control | Shark Feature | Key File / Endpoint |
|---|---|---|
| Multi-tenant org isolation | Organizations + per-org RBAC | `internal/rbac/rbac.go`, `RBACManager.HasPermission()` |
| SAML SSO upstream | SAML SP (crewjam/saml) | `internal/api/sso_handlers.go:SAMLMetadata`, `SAMLACS` |
| OIDC upstream | OIDC client | `internal/api/sso_handlers.go:OIDCAuth`, `OIDCCallback` |
| Domain auto-routing | Email → SSO connection | `GET /api/v1/sso/auto?email=user@stmarys.edu` |
| JIT user provisioning | HandleSAMLACS → user upsert | `internal/api/sso_handlers.go:173` (manager.HandleSAMLACS) |
| Per-role permissions | RBAC matrix, wildcard matching | `internal/rbac/rbac.go:HasPermission`, `GetEffectivePermissions` |
| Proxy CEL enforcement | Proxy rules: role + mfa_passed | `internal/proxy/rules.go`, `internal/proxy/proxy.go` |
| MFA enforcement (TOTP) | TOTP enroll/verify | `internal/api/mfa_handlers.go`, `/auth/mfa/totp/setup` |
| MFA enforcement (Passkeys) | WebAuthn | `internal/api/mfa_handlers.go`, `/auth/mfa/passkey/*` |
| Per-route MFA gate | `mfa_passed=true` session check | `internal/proxy/rules.go:ReqMFA` |
| Audit log (all events) | `audit.Logger.Log()` | `internal/audit/audit.go` |
| Audit real-time SSE | `dispatcher.Emit(system.audit_log)` | `internal/audit/audit.go:55` |
| Audit queryable | `QueryAuditLogs(opts)` | `internal/storage/audit.go`, `GET /api/v1/admin/audit` |
| Audit export (CSV/JSON) | Admin audit endpoint | `GET /api/v1/admin/audit?format=csv&from=...&to=...` |
| Webhooks (HMAC-SHA256) | `webhook.Dispatcher.Emit()` | `internal/api/webhook_emit.go` |
| Webhook retry + dead-letter | Retry schedule [1m,5m,30m,2h,12h] | `internal/api/webhook_handlers.go` |
| Webhook replay | Manual replay endpoint | `POST /api/v1/admin/webhooks/{id}/replay` |
| Impersonation w/ audit | Impersonation session + audit trail | `admin/src/empty_shell` (Phase 9 — UI placeholder, core tracked) |
| Session debugger | Admin inspect any session claims | `admin/src/session_debugger.tsx` |
| Signing key rotation | `RotateSigningKeys()` tx, JWKS dual-window | `internal/storage/jwt_keys.go:RotateSigningKeys`, `ListJWKSCandidates` |
| Compliance tab | Consent records, GDPR export/deletion | `admin/src/compliance.tsx`, `CompliancePage` |
| Key rotation CLI | `shark keys rotate` | `cmd/shark/cmd/keys.go` |

---

## 4. Compliance Crosswalk

### HIPAA Technical Safeguards (45 CFR § 164.312)

| HIPAA Control | Requirement | Shark Feature | Status |
|---|---|---|---|
| 164.312(a)(1) | Access Control — unique user ID | User accounts, org membership | SHIPPED |
| 164.312(a)(1) | Access Control — emergency access | Impersonation w/ time-box + audit | SHIPPED (core), UI Phase 9 |
| 164.312(a)(1) | Access Control — automatic logoff | Session lifetime + expiry | SHIPPED |
| 164.312(a)(1) | Access Control — encryption/decryption | AES-GCM session, ES256 JWTs | SHIPPED |
| 164.312(a)(2)(i) | Unique user identification | Per-user `sub`, per-org membership | SHIPPED |
| 164.312(a)(2)(iii) | Automatic logoff | `session_lifetime` config | SHIPPED |
| 164.312(b) | Audit Controls — hardware/software activity recording | `audit.Log()` on every action | SHIPPED |
| 164.312(b) | Audit Controls — query + export | `QueryAuditLogs`, CSV/JSON export | SHIPPED |
| 164.312(c)(1) | Integrity — mechanisms to authenticate ePHI | JWT ES256 signature, DPoP binding | SHIPPED |
| 164.312(d) | Person/Entity Authentication — verify identity | MFA (TOTP + Passkeys), SSO | SHIPPED |
| 164.312(d) | Person/Entity Authentication — per-route enforcement | Proxy rule `mfa_passed=true` | SHIPPED |
| 164.312(e)(1) | Transmission Security — encryption in transit | TLS (operator responsibility) + JWT integrity | SHIPPED |
| 164.312(e)(2)(ii) | Encryption at rest | SQLite AES-GCM for private keys (`PrivateKeyPEM` AES-GCM in `jwt_keys.go`) | SHIPPED |

### SOC 2 Type II — CC6 (Logical and Physical Access)

| SOC 2 CC Control | Requirement | Shark Feature | Status |
|---|---|---|---|
| CC6.1 | Logical access security — registered users only | User accounts + SSO + JIT provisioning | SHIPPED |
| CC6.1 | Role-based access restrictions | RBAC matrix, `HasPermission()` | SHIPPED |
| CC6.1 | MFA for privileged access | TOTP + Passkeys, per-route enforcement | SHIPPED |
| CC6.2 | New access provisioning | JIT from SAML/OIDC assertions, admin UI | SHIPPED |
| CC6.2 | Role assignment review | Admin org member + role CRUD | SHIPPED |
| CC6.3 | Access removal (termination) | User deactivation, session revocation | SHIPPED |
| CC6.6 | Logical access from outside boundaries | SSO + MFA gate | SHIPPED |
| CC6.7 | Transmission of data restricted | Proxy CEL rules, per-route auth | SHIPPED |
| CC6.8 | Malicious software prevention | Audit log of privileged ops | SHIPPED |
| CC7.2 | System monitoring — security events | Webhook → Slack #soc-alerts | SHIPPED |
| CC7.2 | Audit trail — who/what/when/where | actor/target/IP/timestamp/req_id | SHIPPED |
| CC9.2 | Key management | ES256 key rotation, JWKS dual-window | SHIPPED |

**NOT YET SHIPPED (honest callouts):**
- **SCIM Directory Sync** — WorkOS's killer feature. Shark does JIT provisioning from SSO assertions but has no SCIM 2.0 endpoint for directory push sync. Tag: Phase Next (post-launch).
- **Impersonation UI** — Core audit logic exists; admin dashboard shows `empty_shell` placeholder. Tag: Phase 9.
- **Audit retention config** — `StartCleanup()` exists in `audit.go` but configurable retention via `sharkauth.yaml` needs wiring. Workaround: manual `DeleteBefore()` or keep all (SQLite scales to millions of rows easily).
- **SOC 2 report / pen test** — Shark provides the technical controls. The SaaS vendor still needs a 3rd-party auditor. Honest positioning: Shark gets you to the audit table; it doesn't issue the report.

---

## 5. Demo Script — 10-Minute Live Walkthrough

**Setup (pre-demo, done ahead of time):**
```bash
# Start SharkAuth single binary
./sharkauth serve --config demos/enterprise/sharkauth-demo.yaml

# Seed demo data
./demos/enterprise/seed-data/seed.sh
```

---

### ACT 1 — "St. Mary's IT calls on Monday" (2 min)

**Story beat:** Maya gets a call. St. Mary's IT needs SSO live in 48 hours or the contract stalls.

**Click: Admin → Organizations → New Org**
- Name: `St. Mary's Hospital`
- Domain: `stmarys.edu`
- (Shows: tenant isolation — all data scoped to org_id)

**Click: SSO → New Connection**
- Type: SAML
- IdP: St. Mary's Okta
- Paste Entity ID + SSO URL from their Okta admin
- Download SP metadata XML → paste into Okta

**CLI (shows it's scriptable — no GUI required):**
```bash
curl -X POST http://localhost:8080/api/v1/sso/connections \
  -H "Authorization: Bearer $SHARK_ADMIN_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "St. Marys Okta",
    "type": "saml",
    "domain": "stmarys.edu",
    "saml_idp_url": "https://stmarys.okta.com/app/exkABC123/sso/saml",
    "saml_idp_entity_id": "http://www.okta.com/exkABC123",
    "saml_idp_cert": "..."
  }'
```

**Test auto-routing:**
```bash
curl "http://localhost:8080/api/v1/sso/auto?email=dr.chen@stmarys.edu"
# → {"redirect": "https://stmarys.okta.com/app/.../sso/saml"}
```

**Wow beat:** Domain `@stmarys.edu` auto-routes to their Okta. Zero user config needed. JIT provisions the account on first login.

---

### ACT 2 — "Define the RBAC Matrix" (2 min)

**Click: Access → RBAC → New Role**

Create 3 roles in the St. Mary's org:

| Role | Permissions |
|---|---|
| PI (Principal Investigator) | `phi:read`, `phi:write`, `study:admin` |
| Coordinator | `phi:read`, `data:entry` |
| Monitor | `phi:read` |

**CLI equivalent:**
```bash
# Create PI role
curl -X POST http://localhost:8080/api/v1/roles \
  -H "Authorization: Bearer $SHARK_ADMIN_KEY" \
  -d '{"name":"PI","org_id":"org_stmarys","permissions":[{"action":"phi","resource":"read"},{"action":"phi","resource":"write"},{"action":"study","resource":"admin"}]}'

# Assign to user
curl -X POST http://localhost:8080/api/v1/orgs/org_stmarys/members \
  -d '{"user_id":"usr_drchen","role":"PI"}'
```

---

### ACT 3 — "The PHI Route: One Rule, Full Enforcement" (3 min — THE WOW MOMENT)

**Story beat:** St. Mary's CISO says: "Every access to `/api/phi/*` must require MFA. No exceptions."

**Click: Infrastructure → Proxy → Add Rule**
```
Path pattern:  /api/phi/*
Method:        GET, POST, PUT
Requirements:
  - role = PI
  - mfa_passed = true
Action on deny: block (403) + emit webhook
```

**CLI:**
```bash
curl -X POST http://localhost:8080/api/v1/admin/proxy/rules \
  -H "Authorization: Bearer $SHARK_ADMIN_KEY" \
  -d '{
    "listener_id": "lst_default",
    "path": "/api/phi/*",
    "methods": ["GET","POST","PUT"],
    "requirements": [
      {"kind": "role", "value": "PI"},
      {"kind": "mfa", "value": "true"}
    ],
    "priority": 10
  }'
```

**Simulate: Coordinator tries /api/phi/patient/42 without MFA**
```bash
curl -H "Authorization: Bearer $COORDINATOR_TOKEN" \
     http://localhost:8080/api/phi/patient/42
# → 403 Forbidden: {"error":"mfa_required","redirect":"/auth/mfa"}
```

**Live in admin dashboard: Audit log row appears instantly:**
```
[2026-04-24T14:32:01Z] DENY  actor=usr_coord01  action=proxy.deny
  target=/api/phi/patient/42  ip=203.0.113.45  req_id=req_8xK2m
  reason=mfa_not_passed  org=org_stmarys
```

**Live in Slack #soc-alerts: Webhook fires (HMAC-SHA256 signed):**
```json
{
  "event": "proxy.deny",
  "actor": "coordinator@stmarys.edu",
  "target": "/api/phi/patient/42",
  "reason": "mfa_not_passed",
  "ip": "203.0.113.45",
  "timestamp": "2026-04-24T14:32:01Z",
  "req_id": "req_8xK2m"
}
```

**Coordinator completes MFA → try again:**
```bash
curl -H "Authorization: Bearer $COORDINATOR_MFA_TOKEN" \
     http://localhost:8080/api/phi/patient/42
# → 200 OK — passes through to upstream
```

**New audit row:**
```
[2026-04-24T14:32:44Z] ALLOW  actor=usr_coord01  action=proxy.allow
  target=/api/phi/patient/42  ip=203.0.113.45  mfa=true
```

**THE WOW MOMENT: ONE proxy rule → MFA enforced, audit written, Slack alerted, retry queued if webhook fails. No code in the app. No middleware. No Lambda. Zero.**

---

### ACT 4 — "The Auditor Asks for 90 Days" (1.5 min)

**Story beat:** HIPAA auditor wants export of all PHI-route access for last 90 days.

**Click: Enterprise → Compliance → Audit Log → Export**
- Date range: 2026-01-24 → 2026-04-24
- Filter: action = proxy.allow OR proxy.deny
- Format: CSV

**CLI:**
```bash
curl "http://localhost:8080/api/v1/admin/audit?from=2026-01-24&to=2026-04-24&action=proxy.allow,proxy.deny&format=csv" \
  -H "Authorization: Bearer $SHARK_ADMIN_KEY" \
  -o phi_access_90days.csv
```

**Sample output (first 3 rows):**
```csv
id,actor_id,actor_type,action,target,ip,status,created_at,req_id
aud_9Kz1,usr_drchen,user,proxy.allow,/api/phi/patient/42,10.0.1.5,success,2026-04-24T14:32:44Z,req_8xK2m
aud_9Kz2,usr_coord01,user,proxy.deny,/api/phi/patient/42,203.0.113.45,failure,2026-04-24T14:32:01Z,req_8xK2m
aud_9Kz3,usr_monitor01,user,proxy.allow,/api/phi/study/7,10.0.1.9,success,2026-04-23T09:15:22Z,req_7yJ1n
```

**Drop the hammer:** Auth0 Enterprise keeps 30 days. To get 12 months you build a custom Datadog pipeline ($4K/mo). Shark keeps logs in SQLite forever (configurable retention). Audit export: one `curl`.

---

### BONUS A — Impersonation Flow (1 min)

**Story beat:** Acme support engineer needs to repro a bug in Dr. Chen's session.

**Click: Enterprise → Impersonation → New Request**
- Target user: `dr.chen@stmarys.edu`
- Reason: "Repro ticket #4821 — PHI tab blank"
- Duration: 30 minutes

System records:
```
[2026-04-24T15:00:00Z] impersonation.start
  actor=eng@acmeclinical.com  target=dr.chen@stmarys.edu
  duration=30m  reason="Repro ticket #4821"
  req_id=req_imp_9x1  ip=198.51.100.1
```

Webhook fires to Slack #soc-alerts. Session auto-expires at 15:30. Immutable audit trail.

*Note: Impersonation admin UI is Phase 9 (placeholder in current dashboard). Core audit + session logic is wired.*

---

### BONUS B — Signing Key Rotation, Live (1 min)

**Story beat:** SOC 2 CC9.2 requires evidence of key rotation policy.

```bash
# Rotate keys (dual-window: old key stays valid for in-flight tokens)
shark keys rotate --algorithm ES256

# Verify JWKS shows both keys
curl http://localhost:8080/.well-known/jwks.json | jq '.keys | length'
# → 2  (new active + recently-retired, both verify tokens)

# After TTL: retired key drops from JWKS
```

`internal/storage/jwt_keys.go:RotateSigningKeys()` — transactional, marks current active as `retired` with `rotated_at=now`, inserts new active. `ListJWKSCandidates(activeOnly=false)` publishes both keys during overlap window. **Zero downtime. No token invalidation.**

---

## 6. Implementation Plan

### Files Needed for Demo

```
demos/
  DEMO_03_ENTERPRISE_COMPLIANCE.md      ← this file
  enterprise/
    sharkauth-demo.yaml                 ← demo-specific config (SSO enabled, audit retention=365d)
    idp-mock/
      README.md                         ← how to run samltest.id or a local saml-idp docker image
      docker-compose.yml                ← SimpleSAMLphp or samltest.id mock
      okta-metadata.xml                 ← sample Okta SP metadata for demo
    seed-data/
      seed.sh                           ← creates org, roles, users, proxy rule, webhook
      audit-sample.json                 ← 90 days of realistic audit events
      audit-sample.csv                  ← same, CSV format
      webhook-payload-sample.json       ← example soc-alert webhook body
```

### Demo Config (`sharkauth-demo.yaml` delta from default)
```yaml
sso:
  enabled: true
audit:
  retention: 8760h   # 365 days
proxy:
  enabled: true
  upstream: http://localhost:9000   # demo upstream (simple echo server)
webhooks:
  enabled: true
mfa:
  totp_enabled: true
  passkeys_enabled: true
```

### SAML Mock IdP Options
1. **samltest.id** — public SAML test IdP, zero config, works for live demos
2. **SimpleSAMLphp** — Docker, `docker run -p 8088:8080 kristophjunge/test-saml-idp`
3. **Okta Developer** — free tier, real Okta SAML app, most realistic for enterprise buyers

### Seed Script Logic (`seed.sh`)
```bash
BASE="http://localhost:8080/api/v1"
KEY="$SHARK_ADMIN_KEY"

# 1. Create org
ORG=$(curl -s -X POST $BASE/admin/orgs -H "Authorization: Bearer $KEY" \
  -d '{"name":"St. Marys Hospital","domain":"stmarys.edu"}' | jq -r '.id')

# 2. Create SSO connection
curl -s -X POST $BASE/sso/connections -H "Authorization: Bearer $KEY" \
  -d "{\"name\":\"St Marys Okta\",\"type\":\"saml\",\"org_id\":\"$ORG\",\"domain\":\"stmarys.edu\",...}"

# 3. Create roles: PI, Coordinator, Monitor
for ROLE in PI Coordinator Monitor; do
  curl -s -X POST $BASE/roles -H "Authorization: Bearer $KEY" \
    -d "{\"name\":\"$ROLE\",\"org_id\":\"$ORG\"}"
done

# 4. Create proxy rule
curl -s -X POST $BASE/admin/proxy/rules -H "Authorization: Bearer $KEY" \
  -d '{"path":"/api/phi/*","requirements":[{"kind":"role","value":"PI"},{"kind":"mfa","value":"true"}]}'

# 5. Register Slack webhook
curl -s -X POST $BASE/admin/webhooks -H "Authorization: Bearer $KEY" \
  -d '{"url":"https://hooks.slack.com/...","events":["proxy.deny","impersonation.start","role.changed"],"secret":"demo_secret_xyz"}'

# 6. Seed 90 days of audit events from JSON
curl -s -X POST $BASE/admin/audit/import -H "Authorization: Bearer $KEY" \
  -d @audit-sample.json
```

---

## 7. The "Wow" Moment

**One proxy rule → five compliance controls simultaneously:**

```bash
curl -X POST /api/v1/admin/proxy/rules \
  -d '{"path":"/api/phi/*","requirements":[{"kind":"role","value":"PI"},{"kind":"mfa","value":"true"}]}'
```

This single API call simultaneously activates:
1. **HIPAA 164.312(a)(1)** — access control on PHI routes
2. **HIPAA 164.312(b)** — audit record created on every request (allow or deny)
3. **HIPAA 164.312(d)** — person/entity authentication via MFA gate
4. **SOC 2 CC6.1** — role-based access restriction enforced at proxy layer
5. **SOC 2 CC7.2** — real-time security event webhook to Slack

No code changes in the application. No middleware to deploy. No Datadog pipeline. No Lambda. **The compliance layer is the auth server.**

---

## 8. Sellable Angle

**One-liner:** *"SharkAuth gives clinical SaaS teams enterprise SSO, per-route MFA, and 12-month audit export — in a single binary — for the cost of a $5/month VPS instead of $10K/month Auth0."*

**Three customer types:**

| Customer Type | Their Pain | Shark's Win |
|---|---|---|
| **Clinical trials SaaS** (Acme Clinical) | Hospital procurement demands SAML + MFA + audit. Auth0 Enterprise = $10K+/mo. | SAML SSO + per-route MFA + unlimited audit retention. $0 license. |
| **Small-biz lending platform** | Fintech enterprise customers need SOC 2 CC6 audit log + role segregation. Build it yourself = 3 months. | RBAC + audit + webhook alerts pre-built. Ship in days, not months. |
| **Compliance reporting SaaS** | Sells to regulated industries (insurance, healthcare). Every customer wants SSO. WorkOS = $125/connection × 40 customers = $5K/mo + no per-route MFA. | Per-org SAML/OIDC, per-route MFA, HMAC-signed webhooks. Linear cost = $0. |

**WorkOS honest comparison:**

| Feature | WorkOS | Shark |
|---|---|---|
| SSO (SAML/OIDC) | Yes, $125/connection | Yes, $0 |
| Directory Sync (SCIM) | Yes, $125/connection | NOT YET (Phase Next) |
| Audit Logs | Yes (via Logstream, extra) | Yes, built-in, unlimited retention |
| Per-route MFA enforcement | No (app-layer, not their proxy) | Yes, proxy CEL rules |
| Self-hosted | No (cloud SaaS only) | Yes, single binary |
| BAA signing | Yes (enterprise plan) | Operator signs own BAA (you own the server) |
| Managed infra / SLA | 99.99% (they manage it) | Operator responsibility |

WorkOS is genuinely better for: teams that want managed infra, SCIM directory sync today, and don't want ops burden. Shark wins on: cost at scale, self-hosted data sovereignty, per-route enforcement, unlimited audit retention, and single-binary simplicity.

---

## 9. UAT Checklist

### Compliance Auditor's Perspective

- [ ] **C-01** Create org "St. Mary's", verify it is tenant-isolated (no cross-org data leakage in audit query)
- [ ] **C-02** Configure SAML connection with samltest.id mock IdP; verify SP metadata served at `/api/v1/sso/saml/{id}/metadata`
- [ ] **C-03** Verify domain auto-routing: `GET /api/v1/sso/auto?email=user@stmarys.edu` returns correct SAML redirect
- [ ] **C-04** JIT provision: first SAML login creates user record; second login updates (upsert, not duplicate)
- [ ] **C-05** Create PI, Coordinator, Monitor roles; assign permissions; verify `HasPermission()` returns correct boolean for each
- [ ] **C-06** Add proxy rule: `path=/api/phi/*  require role=PI mfa=true`; verify Coordinator token gets 403
- [ ] **C-07** Coordinator completes MFA challenge; verify mfa_passed=true session; verify same route returns 200
- [ ] **C-08** Audit log: verify every proxy.deny produces row with actor/target/IP/timestamp/req_id
- [ ] **C-09** Audit log: verify every proxy.allow produces row; real-time SSE event emitted
- [ ] **C-10** Export audit log CSV: `GET /api/v1/admin/audit?from=...&format=csv`; verify all columns present
- [ ] **C-11** Export audit log JSON: same endpoint, `format=json`; verify valid JSON array
- [ ] **C-12** Webhook: configure Slack endpoint; verify HMAC-SHA256 signature on `proxy.deny` event
- [ ] **C-13** Webhook retry: simulate 500 from endpoint; verify retry at 1m, 5m, 30m schedule
- [ ] **C-14** Dead-letter: after all retries exhausted, verify event in dead-letter queue; verify manual replay works
- [ ] **C-15** Key rotation: `shark keys rotate`; verify JWKS shows 2 keys; verify old token still validates; verify new token uses new kid
- [ ] **C-16** Impersonation: initiate impersonation session; verify audit row with actor/target/duration/reason; verify session auto-expires
- [ ] **C-17** Session debugger: admin inspects coordinator session; verify MFA state and role claims visible
- [ ] **C-18** Compliance tab: verify consent records queryable; verify GDPR export produces user data package

### Functional Regression

- [ ] **F-01** Non-PHI route (e.g., `/api/studies`) passes through without MFA for Coordinator
- [ ] **F-02** PI with MFA can access `/api/phi/*` (positive path)
- [ ] **F-03** Monitor (read-only) blocked from `phi:write` even with MFA
- [ ] **F-04** Org isolation: user in org_A cannot query audit logs for org_B

---

## Appendix: Key Source Files

```
internal/audit/audit.go              — Logger.Log(), Query(), StartCleanup()
internal/rbac/rbac.go               — RBACManager, HasPermission(), SeedDefaultRoles()
internal/api/sso_handlers.go        — SAMLMetadata(), SAMLACS(), OIDCAuth(), OIDCCallback(), auto-route
internal/api/mfa_handlers.go        — TOTP setup/verify, Passkey, mfa_passed session upgrade
internal/api/webhook_emit.go        — Dispatcher.Emit(), HMAC signing
internal/api/webhook_handlers.go    — retry schedule, dead-letter, replay
internal/proxy/rules.go             — ReqRole, ReqMFA, rule evaluation
internal/proxy/proxy.go             — reverse proxy + rule enforcement + audit.Log(proxy.allow/deny)
internal/storage/jwt_keys.go        — RotateSigningKeys(), ListJWKSCandidates() (dual-window)
admin/src/compliance.tsx            — CompliancePage (consent, GDPR export, deletion)
admin/src/session_debugger.tsx      — SessionDebugger (admin inspect session claims)
admin/src/rbac.tsx                  — RBAC matrix UI
admin/src/sso.tsx                   — SSO connection management UI
cmd/shark/cmd/keys.go               — `shark keys rotate` CLI
```
