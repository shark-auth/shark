# shark.email — Zero-Config Email for Self-Hosted Shark

**Date:** 2026-04-14
**Status:** Planned

---

## Problem

Email is the #1 adoption killer for self-hosted auth. Developer runs `shark serve`, everything works, then hits "send magic link" and needs to:
1. Pick an email provider (Resend? SES? Mailgun?)
2. Sign up, verify a domain, configure DNS (SPF, DKIM, DMARC)
3. Get API keys or SMTP credentials
4. Configure Shark's YAML
5. Debug deliverability issues

This takes 30-60 minutes and kills the "running in 10 seconds" promise. Most developers abandon or defer email-dependent features (magic links, verification, password reset).

---

## Solution

### 1. Dev Inbox (Zero Config, Local Only)

`shark serve --dev` captures all outbound email and displays it in the dashboard under a "Dev Inbox" tab.

- No SMTP config needed during development
- Magic links, verification emails, password resets — all visible in dashboard
- Links are clickable directly from Dev Inbox
- Emails also printed to stdout as clickable URLs
- Like Mailhog/Mailtrap but built into the binary, zero setup

This ships with v0.1.0. Non-negotiable for DX.

### 2. shark.email Relay (Free Tier, Cloud-Backed)

A free transactional email relay for self-hosted Shark instances.

**Configuration:**
```yaml
email:
  provider: "shark"
  api_key: "${SHARK_EMAIL_KEY}"
```

One line. No domain verification for getting started. Emails sent from `noreply@shark.email` (shared domain) or from a verified custom domain.

**Free tier:** 1,000 emails/month
- Covers ~500 MAU with magic links + verification
- Enough for any side project or early-stage startup
- No credit card required

**Paid tier (bundled with Cloud):**
- Cloud Starter: 5,000 emails/mo
- Cloud Pro: 50,000 emails/mo
- Cloud Business: 500,000 emails/mo
- Overage: $0.001/email

**Custom domain:** Available on Pro+ (bring your own domain, Shark handles SPF/DKIM setup with guided DNS wizard in dashboard)

### 3. Provider Presets (For Teams With Existing Email)

Named provider shortcuts instead of raw SMTP:

```yaml
email:
  provider: "resend"       # Maps to smtp.resend.com:465 automatically
  api_key: "${RESEND_API_KEY}"
```

Supported presets:
- `resend` — Resend HTTP API (already implemented, auto-detected)
- `sendgrid` — SendGrid SMTP/API
- `ses` — Amazon SES
- `postmark` — Postmark
- `mailgun` — Mailgun
- `smtp` — Raw SMTP (current behavior, always available)
- `shark` — shark.email relay

Each preset knows the host, port, auth method, and optimal settings. Developer provides one API key.

---

## Strategic Value

### Funnel Entry
shark.email gives self-hosted users a reason to create a Shark Cloud account without committing to cloud hosting. Free account -> free email relay -> relationship established -> future upsell to Cloud Pro when they grow.

### Adoption Multiplier
Removing email friction means:
- Magic links work out of the box in dev (Dev Inbox)
- Magic links work in production with one config line (shark.email)
- Password reset, email verification, MFA recovery — all "just work"
- Time-to-working-auth drops from 30-60 min to under 2 min

### Stickiness
Once a project sends auth emails through shark.email, migrating away requires reconfiguring email (new provider, DNS, etc.). Low-friction lock-in without being hostile — they can always switch to any SMTP/provider.

### Revenue
At scale, email is cheap to operate (~$0.0001/email via SES) but valuable to charge for. The margin on email relay is extremely high.

---

## Implementation

### Dev Inbox (v0.1.0)
- New email sender: `internal/email/dev.go`
- Implements same `Sender` interface as SMTP/Resend
- Stores emails in memory (ring buffer, last 100)
- Dashboard tab: `/admin/dev-inbox` — list + detail view
- Stdout logging: prints magic link / verification URLs directly
- Auto-enabled when `--dev` flag is set or no email provider configured

### shark.email Relay (v0.3.0)
- New email sender: `internal/email/sharkemail.go`
- Simple HTTP API: `POST https://email.sharkauth.com/v1/send`
- Auth: API key (issued when user creates free Shark Cloud account)
- Backend: SES or Postmark (whichever is cheapest at volume)
- Rate limiting: per-key, enforced server-side
- Bounce/complaint handling: auto-disable keys with high bounce rates

### Provider Presets (v0.2.0)
- Extend config parser to accept `provider` field
- Map provider name to connection settings
- Keep raw SMTP as fallback
- Validate API key format per provider where possible

---

## Pricing

| | Free | Pro | Business |
|---|---|---|---|
| Monthly emails | 1,000 | 50,000 | 500,000 |
| Sending domain | shark.email (shared) | Custom domain | Custom domain |
| Overage | Blocked | $0.001/email | $0.001/email |
| Deliverability monitoring | -- | Basic | Full |

Bundled with Cloud tiers — not a separate product. Self-hosted users who only want email get the free 1,000/mo.

---

*shark.email is the trojan horse: solve the biggest OSS pain point for free, create a Cloud relationship, upsell later.*
