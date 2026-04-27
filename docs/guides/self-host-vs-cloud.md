# Self-Hosted vs Cloud

SharkAuth ships as a single Go binary. Everything in this doc works on self-hosted today. Cloud is not yet available — this doc captures the decision matrix so you can plan your deployment path.

**Audience:** Technical evaluators choosing a deployment model.

---

## The core rule

> The binary is the product. Every auth feature ships in the binary. Cloud sells operational convenience, not capabilities.

Self-hosted is not a "lite" version. It runs the same OAuth 2.1 stack, the same RBAC, the same MCP agent auth, and the same audit pipeline as cloud. There is no feature gating.

---

## Feature matrix

| | Self-Hosted | Cloud (planned) |
|---|---|---|
| **Auth features** | All (OAuth 2.1, OIDC, SSO, MFA, passkeys, magic links, RBAC, M2M keys, agent/MCP auth, token exchange) | Identical — same binary |
| **Audit logs** | Full — filterable, exportable, CSV | Full + managed retention |
| **Admin dashboard** | Svelte, embedded in binary | Next.js, hosted |
| **Database** | SQLite (default) or Postgres | Managed Postgres, multi-tenant |
| **Email** | Self-configured (Resend, SMTP, SES, Postmark, Mailgun) or `shark.email` (dev) | Preconfigured per tenant |
| **OAuth configuration** | Dashboard Settings or `shark admin config` CLI | Dashboard UI |
| **Scaling** | Single instance (SQLite) → horizontal (Postgres + Redis, post-v0.1) | Multi-instance behind LB |
| **High availability** | You operate it | Managed (SLA tiers, see below) |
| **Multi-tenancy** | Single tenant | Multi-tenant, isolated |
| **Data residency** | Your infrastructure, your region | Cloud chooses region (custom domain available on paid tiers) |
| **Uptime SLA** | None (you own it) | 99.5% (Pro) / 99.9% (Business) / 99.99% (Enterprise) |
| **Support** | Community (Discord + GitHub) | Email → Priority → Dedicated |
| **Price** | $0 forever | Free tier + paid tiers (see below) |

---

## When to self-host

Self-host when:

- **Data residency is non-negotiable.** Regulated industries (HIPAA, FedRAMP, financial services) often require that auth tokens and user records never leave your infrastructure. Self-hosted gives you full control: you choose the region, the storage path, the network boundary.
- **You already run infrastructure.** A Go binary + SQLite is a `docker run` and a volume mount. If you have a VPS, a k8s cluster, or a Fly.io app, the operational overhead is minimal.
- **You need to audit the auth code.** OSS means you can read, fork, and modify every line. Security-sensitive teams often require this.
- **You want zero per-MAU cost.** Self-hosted is free regardless of user volume.
- **Compliance requirements mandate on-premises.** SOC 2 Type II / ISO 27001 audits that require evidence of data isolation are much simpler when you control the stack.

---

## When to use Cloud (once available)

Use Cloud when:

- **You don't want to operate anything.** Zero infra, zero migrations, zero on-call for auth.
- **You need multi-region redundancy** without building it yourself.
- **You want managed compliance evidence.** Cloud Business and above generate SOC 2 evidence reports from audit logs automatically.
- **You're starting and want to iterate fast.** Free Cloud tier (5K MAU) is the fastest path to a working auth system for early-stage products.
- **Your team can't maintain a stateful service.** If you have no DevOps capacity, Cloud is the correct choice even if self-host is technically simple.

---

## Scaling thresholds

### SQLite (default self-hosted)

SQLite is embedded in the binary — zero config, zero external dependency. It handles single-instance workloads well.

Practical limits:
- Write throughput: ~500 concurrent writes/sec on commodity hardware (WAL mode)
- Read throughput: effectively unlimited for read-heavy auth workloads
- Storage: no practical limit for auth data volumes (tokens, sessions, audit logs)
- **Hard limit:** single writer. Horizontal scale requires migrating to Postgres.

SQLite is appropriate for most self-hosted deployments. If you're running auth for <100K MAU on a single machine, SQLite is fine.

### Postgres (self-hosted, post-v0.1)

Postgres support is on the [Q3 2026 roadmap](../../SCALE.md). Once available:

Configure via `shark admin config` or the dashboard Settings → Database:

```
storage.driver = postgres
storage.dsn    = ${DATABASE_URL}
```

Postgres unlocks:
- Multiple Shark instances behind a load balancer
- Read replicas for audit log queries
- Managed cloud databases (RDS, Cloud SQL, Supabase, Neon)

### Redis / shared cache (post-v0.1)

Five in-memory stores (rate limit buckets, DPoP nonce cache, device flow state, SSO state, proxy circuit breaker) are currently process-local. For horizontal scale these need shared state. The roadmap targets either Postgres tables with time-windowed indexes or an optional Redis driver. Decision not yet finalized.

Until shared cache lands, running multiple Shark instances with a shared Postgres backend is possible but the in-memory stores are not shared — DPoP replay protection and device flow state will be per-instance.

### HA topology (self-hosted Postgres path)

```
              ┌─────────────────┐
              │  Load Balancer  │
              └────────┬────────┘
              ┌────────┴────────┐
         ┌────▼────┐       ┌────▼────┐
         │  shark  │       │  shark  │
         │  :8080  │       │  :8081  │
         └────┬────┘       └────┬────┘
              └────────┬────────┘
              ┌────────▼────────┐
              │    Postgres     │
              │  (primary +     │
              │   read replica) │
              └─────────────────┘
```

`SHARKAUTH_STORAGE__DSN=postgres://...` on each instance. No distributed lock manager needed — the storage interface handles atomic operations.

---

## Cloud tiers (planned)

Cloud is not yet available. Expected tiers based on [`STRATEGY.md`](../../STRATEGY.md):

| | Starter | Pro | Business | Enterprise |
|---|---|---|---|---|
| **Price** | Free | $49/mo | $249/mo | Custom |
| **MAU** | 5,000 | 50,000 | 500,000 | Unlimited |
| **Overage** | — | $0.005/MAU | $0.003/MAU | Negotiated |
| **SSO Connections** | 1 OIDC | 3 (SAML+OIDC) | Unlimited | Unlimited |
| **Organizations** | 3 | 50 | Unlimited | Unlimited |
| **Webhooks** | 1 endpoint | 10 endpoints | Unlimited | Unlimited |
| **Audit Retention** | 7 days | 90 days | 1 year | Custom |
| **Agent/MCP Auth** | 5 agents | 100 agents | Unlimited | Unlimited |
| **Edge Bundles** | — | 1 region | Multi-region | Global |
| **Support** | Community | Email (48h) | Priority (4h) + Slack | Dedicated + SLA |
| **Dashboard Seats** | 1 | 5 | 20 | Unlimited |
| **Compliance Reports** | — | — | SOC2 evidence | Full compliance |
| **Uptime SLA** | — | 99.5% | 99.9% | 99.99% |
| **Impersonation** | — | — | Yes | Yes |
| **Custom Domain** | — | Yes | Yes | Yes |

Note: Self-hosted has no limits on any of the above — organizations, agents, webhooks, audit retention are all configuration, not gating.

---

## Data residency detail

### Self-hosted

You control everything:
- User PII stays on your infrastructure
- Auth tokens are issued and stored in your database
- Audit logs are on your disk or your managed DB
- Email traffic goes through your configured provider
- No data leaves your network boundary except for the email provider you configure

Useful for GDPR (data residency in EU), HIPAA (PHI stays on-premises), FedRAMP (US Gov cloud only), and financial services requiring on-premises key storage.

### Cloud

Cloud is a multi-tenant SaaS. Until custom domain + data residency features ship:
- Data is in the region Shark Cloud operates in
- Tenant isolation is enforced at the application layer (schema-level `tenant_id` on all tables)
- Compliance reports (SOC 2 evidence) available on Business+

If you have strict data residency requirements and Cloud does not yet support your required region, self-host.

---

## Migration path

You can start on self-hosted and migrate to Cloud later (or vice versa). The data model is the same binary. Migration tooling is planned but not yet available.

Self-hosted → Cloud migration will require:
1. Export users (the Auth0 export format is already supported for import)
2. Migrate OAuth client registrations
3. Migrate audit logs if retention is required
4. Point DNS at Cloud and update `base_url`

---

## Quick decision

| Situation | Deploy model |
|-----------|-------------|
| Regulated industry, data residency required | Self-hosted |
| Need to audit the auth code | Self-hosted |
| Already running k8s / Docker | Self-hosted |
| No infra team, need to ship fast | Cloud (Starter) |
| >100K MAU, need HA today | Cloud (Business) |
| Need SOC 2 compliance reports | Cloud (Business) or self-host with your own evidence pipeline |
| Want zero per-MAU cost at scale | Self-hosted (Postgres) |
| Evaluating before committing | Cloud (Starter, free) or `shark serve --dev` |

---

## Self-hosted quick start

```bash
# 1. Install
go install github.com/sharkauth/sharkauth/cmd/shark@latest
# or download a prebuilt binary from GitHub Releases

# 2. Dev mode (no config needed — ephemeral SQLite, built-in email inbox)
shark serve --dev

# 3. Production — run once, complete setup via dashboard at /admin
shark serve
# First-boot wizard at http://localhost:8080/admin sets base_url, email provider, etc.
# Config stored in DB — manage via dashboard Settings or: shark admin config
```

Production checklist before exposing to users:
- Set `server.base_url` to your HTTPS URL
- Switch `email.provider` from `shark` (rate-limited dev tier) to `resend`, `smtp`, `ses`, `postmark`, or `mailgun`
- Set `server.cors_origins` to your frontend origins
- Mount a persistent volume at the SQLite path (default: `shark.db` in the working directory)
- Put a TLS terminator (Caddy, nginx, Cloudflare) in front

See [README](../../README.md) for the full config reference.
