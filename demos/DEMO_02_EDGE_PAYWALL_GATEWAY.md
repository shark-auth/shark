# Demo 02 — Edge Paywall Gateway for Vertical SaaS Tier Monetization

> **Tagline:** Ship Stripe-level tier paywalls at the edge in 90 seconds — zero backend changes, one binary.

---

## 1. Story

**Persona:** Marcus Webb, solo founder, "VibeCode" — an AI-powered IDE SaaS targeting indie developers and small dev teams.

**The Pain:**
Marcus has 8 microservices: `api-completion`, `api-refactor`, `api-debug`, `api-collab`, `api-deploy`, `api-docs`, `api-billing`, `api-dashboard`. Every one of them has a copy of this pattern:

```go
// In every single service — repeated 8 times
tier, err := db.GetUserTier(ctx, userID)
if tier != "pro" {
    http.Error(w, "upgrade required", 403)
    return
}
```

He added Stripe Customer Portal for billing. Then LaunchDarkly for feature flags. Then a custom `paywallMiddleware` package he copy-pastes into each service. Each time he ships a new pro feature he:

1. Adds a flag in LaunchDarkly ($20/seat/mo)
2. Wires Stripe webhooks to sync tier to his DB (3–4 hrs/feature)
3. Adds middleware to the new service (copy-paste debt)
4. Deploys all 8 services to propagate the change

**The stack cost at 10K MAU:**
- Auth0 (identity): ~$230/mo (10K MAU beyond free tier at $23/1000)
- Stripe Connect (billing engine): 0.25% + $0.25/transaction
- LaunchDarkly (feature flags): ~$160/mo (8 seats × $20)
- Schematic / custom paywall middleware: $149/mo or 2 weeks of eng
- Cloudflare Workers + custom authorizer glue: ~$35/mo + eng hours
- **Total: $500–700/mo + 1 eng-week per new gated feature**

**With Shark:**
- Single binary (`shark serve`), self-hosted, $0 SaaS fees at any MAU
- One CLI command or Admin API call gates any path across all upstreams
- Tier claim is JWT-baked — proxy evaluates at the edge, zero DB calls
- Branded upgrade page auto-rendered from `shark init` design tokens
- Audit + webhook fanout built-in

---

## 2. Architecture Diagram

```
                         VIBECODE USERS
                              │
                    ┌─────────▼──────────┐
                    │  Browser / IDE CLI  │
                    └─────────┬──────────┘
                              │  HTTPS :443
                    ┌─────────▼──────────────────────────────┐
                    │         SharkAuth (single binary)       │
                    │                                         │
                    │  ┌──────────────────────────────────┐   │
                    │  │  Reverse Proxy (port 8080)        │   │
                    │  │                                   │   │
                    │  │  1. Strip X-Shark-* spoofed hdrs  │   │
                    │  │  2. Resolve Identity from JWT     │   │
                    │  │     (tier claim: free | pro)      │   │
                    │  │  3. Engine.Evaluate() ──────────► │   │
                    │  │     atomic.Pointer[RoutingTable]  │   │
                    │  │     CEL rule: require tier:pro    │   │
                    │  │                                   │   │
                    │  │  DecisionAllow ──────────────────►│──►│ upstream :3001
                    │  │    inject X-Shark-User-ID         │   │ /api/ai-completion
                    │  │    inject X-Shark-Scopes          │   │
                    │  │    inject X-Shark-Trace-ID        │   │
                    │  │                                   │   │
                    │  │  DecisionPaywallRedirect ────────►│──►│ Hosted Upgrade Page
                    │  │    (tier:free hits tier:pro rule) │   │ (design tokens from
                    │  │    audit: proxy.paywall event     │   │  shark init palette)
                    │  │    webhook → Slack #alerts        │   │
                    │  └──────────────────────────────────┘   │
                    │                                         │
                    │  ┌──────────────────────────────────┐   │
                    │  │  Admin API  :9090                 │   │
                    │  │  GET/POST/DELETE /admin/proxy/rules│  │
                    │  │  POST /admin/proxy/reload         │   │
                    │  │  GET  /admin/proxy/status/stream  │   │
                    │  └──────────────────────────────────┘   │
                    │                                         │
                    │  SQLite (rules.db)  ◄── atomic hot-swap │
                    └─────────────────────────────────────────┘

    VibeCode Upstreams (unchanged — zero backend modifications):
    :3001 api-completion  :3002 api-refactor  :3003 api-debug
    :3004 api-collab      :3005 api-deploy    :3006 api-docs
```

---

## 3. Shark Feature Map

| Demo Step | Feature | File(s) |
|-----------|---------|---------|
| Start proxy with rules | `shark serve` embedded proxy, `atomic.Pointer[RoutingTable]` | `internal/proxy/engine.go`, `internal/proxy/rules.go` |
| Add tier:pro rule via API | `POST /api/v1/admin/proxy/rules` handler | `internal/api/proxy_admin_v15_handlers.go` |
| Add tier:pro rule via CLI | `shark proxy-admin rules add --json` | `cmd/shark/cmd/proxy_admin.go` |
| Free user hits gated route | `Engine.Evaluate()` → `DecisionPaywallRedirect` (ReqTier mismatch) | `internal/proxy/rules.go:Engine.Evaluate`, `internal/proxy/engine.go` |
| Branded upgrade page renders | Hosted page with injected design tokens from sharkauth.yaml branding | `internal/api/hosted_handlers.go`, `internal/admin/dist/hosted.html` |
| Strip spoofed headers | `StripIncoming` in `ServeHTTP` | `internal/proxy/proxy.go:ServeHTTP`, `internal/proxy/headers.go` |
| Pro user passes through | `DecisionAllow` → header injection | `internal/proxy/headers.go`, `internal/proxy/proxy.go` |
| X-Shark-* headers injected | `InjectIdentityHeaders()` | `internal/proxy/headers.go` |
| DPoP proof verified | `dpopCache.Validate()` at proxy layer | `internal/proxy/proxy.go:SetDPoPCache` |
| Live rule edit (no restart) | `Engine.SetRules()` → `atomic.Pointer.Store()` | `internal/proxy/engine.go:SetRules` |
| Audit event fired | `audit.Log(action="proxy.paywall")` | `internal/audit/`, `internal/proxy/proxy.go` |
| Webhook to Slack | `webhook.Dispatcher.Emit("proxy.paywall", payload)` | `internal/webhook/dispatcher.go` |
| Admin dashboard rule table | Admin SPA proxy rules panel | `internal/admin/dist/` (Settings/Proxy panel) |
| Circuit breaker per upstream | Per-instance state machine, lifecycle: Closed→Open→HalfOpen | `internal/proxy/` (circuit breaker) |

---

## 4. Demo Script (10 Minutes)

### T+0:00 — Setup (pre-loaded, show terminal)

```bash
# VibeCode mock upstreams already running (Node.js)
# :3001 /api/ai-completion  — returns {"completion": "..."}
# :3002 /api/refactor       — returns {"refactored": "..."}

# SharkAuth already initialized
cat sharkauth.yaml
# branding:
#   logo_url: "https://vibecode.io/logo.png"
#   primary_color: "#6C47FF"
#   company_name: "VibeCode"

shark serve
# SharkAuth v0.9 listening :8443 (auth) :8080 (proxy) :9090 (admin)
```

### T+0:45 — The Problem (live context)

```bash
# Show the duplicated middleware across all 8 services
cat demos/edge_paywall/upstream-mock/middleware-debt.txt
# → 8 files, each with the same tier check — 240 lines of copy-paste
```

### T+1:30 — Add the paywall rule in ONE command

```bash
# CLI path (sales-friendly, copy-pasteable)
shark proxy-admin rules add --json '{
  "name":    "ai-completion-pro-gate",
  "path":    "/api/ai-completion",
  "methods": ["GET","POST"],
  "require": "tier:pro",
  "allow":   "tier:pro",
  "paywall_app": "vibecode-upgrade"
}'

# Confirm it's live
shark proxy-admin rules list
# ID  NAME                      PATH                  REQUIRE    ACTION
# r1  ai-completion-pro-gate    /api/ai-completion    tier:pro   paywall
```

### T+2:30 — Free user hits the gated route

```bash
# Mint a free-tier JWT (tier claim = "free")
FREE_TOKEN=$(curl -s -X POST http://localhost:8443/oauth/token \
  -d "grant_type=password&username=alice@vibecode.io&password=demo" \
  | jq -r .access_token)

# Hit the AI completion endpoint
curl -i -H "Authorization: Bearer $FREE_TOKEN" \
  http://localhost:8080/api/ai-completion

# HTTP/1.1 302 Found
# Location: http://localhost:8443/paywall/vibecode-upgrade
# X-Shark-Deny-Reason: tier predicate failed: have "free", rule requires "pro"
```

### T+3:00 — The "wow" moment: branded upgrade page renders

```bash
# Open browser to the redirect URL
open http://localhost:8443/paywall/vibecode-upgrade
```

*Browser shows:*
- VibeCode logo (pulled from sharkauth.yaml branding.logo_url)
- Violet #6C47FF accent color (primary_color from shark init)
- "Upgrade to VibeCode Pro" headline
- Feature list: AI Completion, Unlimited Refactor, Deploy Assist
- "Upgrade Now" CTA button → Stripe billing link
- **Zero backend code written. Zero. The HTML was rendered by Shark at the edge.**

### T+4:00 — Pro user passes through cleanly

```bash
# Mint a pro-tier JWT (tier claim = "pro")
PRO_TOKEN=$(curl -s -X POST http://localhost:8443/oauth/token \
  -d "grant_type=password&username=bob@vibecode.io&password=demo" \
  | jq -r .access_token)

curl -i -H "Authorization: Bearer $PRO_TOKEN" \
  http://localhost:8080/api/ai-completion \
  -d '{"prompt": "refactor this function"}'

# HTTP/1.1 200 OK
# (upstream response — VibeCode API replied normally)
```

```bash
# Show what the upstream actually received (logged by mock server)
cat demos/edge_paywall/upstream-mock/last-request.log
# X-Shark-User-ID: usr_b9f3c2a1
# X-Shark-Scopes: completion:read completion:write
# X-Shark-Trace-ID: 7f3a-9b2c-...
# X-Shark-User: (stripped — spoofing attempt would have been blocked)
```

### T+5:30 — Live rule edit: no restart

```bash
# Add a second gated feature — live, no downtime
shark proxy-admin rules add --json '{
  "name":    "deploy-pro-gate",
  "path":    "/api/deploy",
  "methods": ["POST"],
  "require": "tier:pro",
  "paywall_app": "vibecode-upgrade"
}'

# Immediately test — atomic.Pointer hot-swap, zero reload
curl -i -H "Authorization: Bearer $FREE_TOKEN" \
  http://localhost:8080/api/deploy

# HTTP/1.1 302 Found — rule live in <1ms
```

### T+6:30 — Admin dashboard: visual rule editor

*Switch to browser → Admin Dashboard → Proxy Rules panel*

1. Show the rules table: ai-completion-pro-gate, deploy-pro-gate
2. Click edit on `ai-completion-pro-gate`
3. Change `require` from `tier:pro` to `tier:enterprise`
4. Save — rule propagates instantly (atomic swap)
5. Re-run free user curl → still blocked (now for enterprise tier)
6. Re-run pro user curl → now blocked too (pro < enterprise)

### T+7:30 — Audit log + webhook to Slack

```bash
# Tail the audit log — every proxy.paywall event captured
shark audit tail --filter action=proxy.paywall --limit 10

# timestamp            action          user                path
# 2026-04-24T10:03:11  proxy.paywall   alice@vibecode.io   /api/ai-completion
# 2026-04-24T10:03:14  proxy.paywall   alice@vibecode.io   /api/deploy
# 2026-04-24T10:07:22  proxy.paywall   carol@vibecode.io   /api/ai-completion
```

*Show Slack channel #vibecode-monetization — webhook delivered:*

```json
{
  "event": "proxy.paywall",
  "user_id": "usr_a4e1c8b2",
  "email": "alice@vibecode.io",
  "path": "/api/ai-completion",
  "required_tier": "pro",
  "user_tier": "free",
  "timestamp": "2026-04-24T10:03:11Z"
}
```

*"Every time a free user hits a pro wall, you get a Slack ping. That's a conversion signal — you can trigger an in-app nurture email the same day."*

### T+8:30 — Circuit breaker demo (bonus if time allows)

```bash
# Kill the upstream mock
kill $(lsof -t -i:3001)

# Shark circuit breaker opens after threshold
curl -i -H "Authorization: Bearer $PRO_TOKEN" \
  http://localhost:8080/api/ai-completion
# HTTP/1.1 503 Service Unavailable
# X-Shark-Deny-Reason: circuit open: ai-completion upstream

# Restart upstream
node demos/edge_paywall/upstream-mock/server.js &

# Half-open probe succeeds → circuit closes
curl -i -H "Authorization: Bearer $PRO_TOKEN" \
  http://localhost:8080/api/ai-completion
# HTTP/1.1 200 OK  (circuit closed, traffic restored)
```

### T+9:30 — Cost comparison close

> "Marcus was paying $500–700/month for Auth0 + LaunchDarkly + Schematic to replicate what Shark does in one binary. At 100K MAU, Auth0 alone is $2,100/month. Shark is a Go binary on a $12 Hetzner VPS. The paywall page is served from memory — no Vercel edge function bill, no third-party session lookup."

---

## 5. Implementation Plan

### Files to Create

```
demos/
└── edge_paywall/
    ├── upstream-mock/
    │   ├── server.js              # Node.js Express mock for all 8 /api/* routes
    │   │                          # Logs received headers to last-request.log
    │   ├── middleware-debt.txt    # Illustrative copy-paste tier check × 8 services
    │   └── package.json
    ├── seed/
    │   ├── proxy-rules.json       # Pre-seeded rules for demo startup
    │   │                          # ai-completion-pro-gate, refactor-pro-gate
    │   ├── users.sql              # alice@vibecode.io (free), bob@vibecode.io (pro)
    │   └── webhook-subscription.json  # Slack webhook target
    ├── sharkauth-vibecode.yaml    # Demo sharkauth.yaml with VibeCode branding
    └── run-demo.sh                # One-shot setup: starts mock + shark + seeds rules
```

### Key Implementation Notes

**upstream-mock/server.js:**
- Express server, all routes respond 200 with JSON
- Log all incoming headers to `last-request.log` for the demo reveal
- Routes: GET/POST `/api/ai-completion`, `/api/refactor`, `/api/debug`, `/api/deploy`, `/api/collab`, `/api/docs`

**seed/proxy-rules.json:**
```json
[
  {
    "name": "ai-completion-pro-gate",
    "path": "/api/ai-completion",
    "methods": ["GET", "POST"],
    "require": "tier:pro",
    "allow": "tier:pro"
  },
  {
    "name": "refactor-free",
    "path": "/api/refactor",
    "methods": ["GET", "POST"],
    "require": "authenticated",
    "allow": "authenticated"
  }
]
```

**sharkauth-vibecode.yaml:**
```yaml
branding:
  logo_url: "https://vibecode.io/logo.png"
  primary_color: "#6C47FF"
  company_name: "VibeCode"
  support_email: "hello@vibecode.io"
```

**run-demo.sh:**
```bash
#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"

echo "Starting VibeCode mock upstreams..."
node upstream-mock/server.js &
MOCK_PID=$!

echo "Seeding proxy rules..."
curl -s -X POST http://localhost:9090/api/v1/admin/proxy/rules/import \
  -H "Authorization: Bearer $SHARK_ADMIN_KEY" \
  -H "Content-Type: application/json" \
  -d @seed/proxy-rules.json

echo "Registering Slack webhook..."
curl -s -X POST http://localhost:9090/api/v1/admin/webhooks \
  -H "Authorization: Bearer $SHARK_ADMIN_KEY" \
  -H "Content-Type: application/json" \
  -d @seed/webhook-subscription.json

echo "Demo ready. SharkAuth proxy at :8080, admin at :9090"
echo "Mock PID: $MOCK_PID (kill with: kill $MOCK_PID)"
```

---

## 6. The "Wow" Moment

**Primary wow:** At T+3:00, the SE opens a browser to the redirect URL and a fully-branded VibeCode upgrade page renders — correct logo, correct violet color palette, correct feature list — without a single line of paywall code written in any of the 8 backend services. The audience sees that the upgrade page came out of `sharkauth.yaml` branding config set during `shark init`. Zero React components. Zero Vercel deployments. Zero Stripe Customer Portal custom theming.

**Secondary wow:** At T+5:30, the SE adds a second pro-gated route (`/api/deploy`) and immediately curls it. The rule is live in under 1 millisecond — no `shark serve` restart, no `nginx -s reload`, no Kubernetes rollout. The `atomic.Pointer[RoutingTable]` swap is lock-free; the data plane never stalls.

**Tertiary wow (conversion signal):** At T+7:30, the Slack message lands. Marcus realizes: every time a free user hits a paywall, he gets a real-time signal he can wire to HubSpot, Customer.io, or Intercom for conversion nurture. He didn't have to build that either — it's a webhook subscription against `proxy.paywall` events, wired in 30 seconds via the admin API.

---

## 7. Sellable Angle

**One-liner:** "Replace Auth0 + LaunchDarkly + Schematic + custom paywall middleware with one self-hosted binary — and gate any API route by tier in 90 seconds."

**Three Customer Types:**

| Customer | Pain | Shark Pitch |
|----------|------|-------------|
| **Vertical SaaS founder (Marcus archetype)** — legal-tech, dental-SaaS, vibe-coding IDE, 8+ microservices | Tier check duplicated in every service; $500+/mo SaaS stack; 1 eng-week per new gated feature | One CLI command gates any route; branded paywall from config; $0 SaaS bill |
| **B2B platform (multi-tenant)** — team plans, seat-based billing, per-org entitlements | Need org-scoped tier enforcement across 10s of services; custom middleware per service team | CEL rules + JWT-baked org tier claim; header injection carries org context; audit log per tenant |
| **API-first startup scaling from 10K → 100K MAU** | Auth0 bill jumps from ~$0 to $2,100/mo at 100K MAU; Cloudflare Workers + custom authorizers add $300+/mo | Shark is self-hosted; no per-MAU pricing; same binary from 10 to 10M users |

---

## 8. Competitive Comparison

| Capability | Auth0 + LaunchDarkly + Schematic | Cloudflare Access + Workers | AWS API GW + Authorizer | **SharkAuth Proxy V1.5** |
|---|---|---|---|---|
| Tier-aware paywall at edge | No (LD flags + custom code) | No (no tier concept) | No (custom authorizer code) | **Yes — ReqTier predicate, DecisionPaywallRedirect** |
| Branded upgrade page | Stripe Customer Portal only | No | No | **Yes — design tokens from sharkauth.yaml** |
| Zero-restart rule propagation | LaunchDarkly SDK poll (30s) | Config deploy (minutes) | No hot-swap | **Yes — atomic.Pointer, <1ms** |
| Audit log per paywall hit | Custom implementation | Cloudflare Logpush ($) | CloudWatch ($) | **Built-in audit.Log** |
| Webhook on paywall event | Zapier/custom | No | No | **Built-in Dispatcher** |
| DPoP proof at proxy layer | No | No | No | **Yes — SetDPoPCache** |
| Price at 10K MAU | ~$500–700/mo | ~$50–100/mo + eng | ~$35/mo + eng | **$0 SaaS (self-hosted)** |
| Price at 100K MAU | ~$2,400–3,500/mo | ~$200–400/mo + eng | ~$350/mo + eng | **$0 SaaS (self-hosted)** |

---

## 9. UAT Checklist

- [ ] `shark serve` starts with proxy on :8080, admin on :9090, auth on :8443
- [ ] `run-demo.sh` seeds proxy rules and webhook subscription in <5s
- [ ] `POST /api/v1/admin/proxy/rules` with tier:pro rule returns 201 and rule appears in GET list
- [ ] `shark proxy-admin rules list` CLI renders table with correct columns
- [ ] Free-tier JWT (`tier: "free"`) hitting `/api/ai-completion` returns 302 to `/paywall/vibecode-upgrade`
- [ ] Response includes `X-Shark-Deny-Reason: tier predicate failed: have "free", rule requires "pro"`
- [ ] Hosted upgrade page renders VibeCode branding (logo, #6C47FF color, company name) from sharkauth.yaml
- [ ] Pro-tier JWT (`tier: "pro"`) hitting `/api/ai-completion` returns 200 from upstream mock
- [ ] Upstream mock `last-request.log` shows `X-Shark-User-ID`, `X-Shark-Scopes`, `X-Shark-Trace-ID`
- [ ] Upstream mock `last-request.log` shows NO `X-Shark-*` headers from the client (spoofing stripped)
- [ ] Adding a second rule via CLI propagates in <1ms (verify with immediate curl, no sleep)
- [ ] Editing a rule via admin dashboard (tier:pro → tier:enterprise) takes effect without `shark serve` restart
- [ ] `shark audit tail --filter action=proxy.paywall` shows every paywall hit with user + path
- [ ] Slack webhook receives JSON payload within 2s of paywall event firing
- [ ] Killing upstream mock → circuit breaker opens → 503 with `X-Shark-Deny-Reason: circuit open`
- [ ] Restarting upstream mock → circuit half-open probe → traffic restored automatically
- [ ] `DELETE /api/v1/admin/proxy/rules/{id}` removes rule; subsequent request passes through (no rule = default deny logged)
- [ ] DPoP-bound access token passing through pro-gated route: proof validated, request forwarded; invalid proof returns 401

---

## 10. Real Shark File References

| Concept | File | Key Symbol |
|---------|------|-----------|
| Atomic routing table | `internal/proxy/engine.go` | `Engine.rules atomic.Pointer[RoutingTable]`, `Engine.SetRules()` |
| ReqTier + PaywallRedirect | `internal/proxy/rules.go` | `RequirementKind.ReqTier`, `DecisionPaywallRedirect`, `Engine.Evaluate()` |
| Header strip + injection | `internal/proxy/headers.go` | `InjectIdentityHeaders()`, strip loop in `ServeHTTP` |
| ServeHTTP dispatch | `internal/proxy/proxy.go` | `ReverseProxy.ServeHTTP`, `SetDPoPCache`, `HeaderDenyReason` |
| Admin rules API | `internal/api/proxy_admin_v15_handlers.go` | `CreateProxyRule`, import handler |
| Admin rules list | `internal/api/proxy_handlers.go` | `handleProxyRules` → `GET /api/v1/admin/proxy/rules` |
| Webhook fanout | `internal/webhook/dispatcher.go` | `Dispatcher.Emit()`, HMAC-SHA256, retry schedule |
| CLI proxy-admin | `cmd/shark/cmd/proxy_admin.go` | `proxyAdminCmd`, `rules add --json` |
| Hosted upgrade page | `internal/api/hosted_handlers.go` | Branding injection into hosted.html |
| Audit log | `internal/audit/` | `Logger.Log(action="proxy.paywall", ...)` |
