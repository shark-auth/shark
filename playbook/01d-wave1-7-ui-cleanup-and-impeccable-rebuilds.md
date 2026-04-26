# Wave 1.7 — UI Cleanup, Coming-Soon Placeholders, Get-Started Impeccable Rebuild

**Budget:** 10-12h CC · **Run after Wave 1.5, before Wave 2** · **Mandatory for honest launch — cuts surface area to defend the moat**

## Why this wave exists

User's call: "cutting lot of features to deliver real moat. this hurts."

Right call. The launch is for the moat. Every additional half-baked surface = surface for HN comments to attack. Better to show "coming soon" honestly than to ship broken features. Better to remove duplicate/incongruent UI than to confuse the YC reviewer who installs shark to evaluate.

## What this wave covers

1. Dev-email banner bug fix (~30 min) — confirmed in code audit, not actually fixed in W18 recovery despite memory note
2. Coming-soon placeholders for proxy + compliance/exporting-logs + branding (~1h)
3. Identity tab vs Settings tab cleanup — remove duplicates, fix incongruencies (~2.5h)
4. Get-started full `/impeccable` rebuild focused on Agent integrations (~6-8h)
5. Discipline directive: every NEW component or screen uses `/impeccable` workflow (zero direct cost — process change)

## Edit 1 — Dev-email banner bug fix (~30 min)

**File:** `admin/src/components/dev_email.tsx:390-406`

**Status from code audit:** W18 recovery (2026-04-26) memory entry claims "fixed dev-email tab" but the actual code at lines 390-406 is unchanged. Bug still ships. Memory note misleading — fix this in Wave 1.7 for real.

**Bug:** banner guard condition checks only `error.includes('404')`. When user changes provider to `dev` in Settings, server returns 200 on the next call but stale error state on the component shows the warning anyway until refetch fires. Source-of-truth is `config.email.provider`, but the component never reads it.

**Fix (smallest diff):**

```typescript
// Add at top of DevEmail component
const { data: config } = useAPI('/admin/config');

// Replace the line ~390 guard
if (!loading && error && (error.includes('404') || error.includes('Not Found'))) {
  // Only show "not dev" warning if the actual config says provider is not dev
  if (config?.email?.provider !== 'dev') {
    return (
      <Banner intent="warning">
        Email provider is not dev. Outbound mail is being sent through your live provider.
        To capture mail here instead, switch the provider to dev in
        Settings → Email Delivery (or run <code>shark mode dev</code> from the CLI).
      </Banner>
    );
  }
  // If config says dev but error persists, show a softer "syncing" state
  return <Banner intent="info">Refreshing dev inbox… give it a moment.</Banner>;
}
```

Plus: trigger `refresh()` when the tab mounts, not just on the 3s poll. Hooks into the existing `useEffect` at line 348-351.

**Acceptance:**
- Switch provider to dev in Settings → click Dev Email tab → no warning banner
- Switch back to live provider → click Dev Email tab → warning shows correctly
- Refresh page in dev mode → no warning flicker

## Edit 2 — Coming-soon placeholders (~1h total)

**Pattern:** keep nav entries visible, swap the route component to a shared `<ComingSoon />` placeholder, preserve real component code in tree at a different import path or behind a feature flag.

**Why coming-soon over hidden:** users see what's planned. Builds anticipation. Honest scoping in the UI itself. Doesn't break sidebar layout. Doesn't surprise people who'd been told "proxy ships in v0.2."

### Edit 2a — Shared `<ComingSoon />` component (~20 min)

**File:** `admin/src/components/coming_soon.tsx` (new)

```tsx
import { Link } from 'react-router-dom';

interface ComingSoonProps {
  feature: string;
  shipDate: string;        // "W18+", "v0.2", "Q3 2026"
  description?: string;
  trackingIssue?: string;  // GitHub issue URL for users to follow
}

export function ComingSoon({ feature, shipDate, description, trackingIssue }: ComingSoonProps) {
  return (
    <div className="flex flex-col items-start gap-4 p-8 max-w-2xl">
      <h1 className="text-2xl font-mono">{feature}</h1>
      <span className="text-sm uppercase tracking-wider text-fg-muted">
        Coming · {shipDate}
      </span>
      {description && <p className="text-fg-secondary">{description}</p>}
      {trackingIssue && (
        <Link to={trackingIssue} className="underline">
          Track progress on GitHub →
        </Link>
      )}
    </div>
  );
}
```

Style: monochrome, square, matches `.impeccable.md v3` lock. No icons. No animations. Honest visual scoping.

### Edit 2b — Proxy: route to coming-soon (~15 min)

**Files:** `admin/src/router.tsx` (or wherever routes live), nav stays as-is.

```tsx
// preserve real component, just don't route to it from production sidebar
import { ProxyManage } from './components/proxy_manage_real';  // renamed import path
import { ComingSoon } from './components/coming_soon';

// Production route
<Route 
  path="/proxy" 
  element={
    <ComingSoon
      feature="Proxy + Auth Flow Builder"
      shipDate="v0.2 (W18-W19)"
      description="Reverse-proxy with policy-driven auth gating, per-route protection, and visual auth flow builder. Shipping after regression fixes for the 12 known bugs surfaced in launch testing."
      trackingIssue="https://github.com/<repo>/issues/proxy-v0.2"
    />
  } 
/>

// Dev/internal route (env-gated, for your testing)
{import.meta.env.VITE_FEATURE_PROXY === 'true' && (
  <Route path="/proxy-dev" element={<ProxyManage />} />
)}
```

**Rename existing component file:** `proxy_manage.tsx` → `proxy_manage_real.tsx`. This makes the swap obvious in code review and prevents accidental import resurfacing.

**File `INVESTIGATE_REPORT.md`** stays in repo root with the 12 known bugs documented. Coming-soon placeholder links to it (or a public GitHub issue mirror).

### Edit 2c — Compliance tab → "Exporting logs" coming-soon (~10 min)

**Files:** sidebar nav (rename label), router (swap component).

- Rename nav label: `Compliance` → `Exporting logs`
- Route to `<ComingSoon feature="Exporting logs" shipDate="v0.2 (W18-W19)" description="Export audit logs to S3, Datadog, BigQuery. SOC2-ready audit trail with cryptographic chain integrity. Built when first enterprise customer asks." />`
- If a real `compliance.tsx` component exists, rename to `compliance_real.tsx` and gate behind `VITE_FEATURE_COMPLIANCE` flag (same pattern as proxy)

**Why "Exporting logs" name:** more concrete and engineer-friendly than "Compliance" (which reads as marketing-speak). Engineers who need it know exactly what they're getting.

### Edit 2d — Branding page → coming-soon (~10 min)

**Files:** sidebar nav stays (label stays "Branding"), router swaps component.

- Route to `<ComingSoon feature="Branding & White-label" shipDate="v0.3 (Q3 2026)" description="Custom domain, custom logo, custom email templates, custom OAuth consent screen. White-label for platforms that resell SharkAuth to their customers." />`
- Rename existing `branding.tsx` → `branding_real.tsx` if real component code exists

**Rationale for v0.3 not v0.2:** branding is enterprise/reseller-tier, not launch-tier. Don't promise W+1 if it's not real. Better to push the date out and beat it later.

## Edit 3 — Identity vs Settings tab cleanup (~2.5h)

**Per code audit, current state has duplicates and misplaced items.** Cleanup defines the line: Identity = WHO can authenticate. Settings = HOW the system operates.

### The reorganization (file-by-file)

| Surface | Current location | Target location | Action |
|---|---|---|---|
| Sessions & Tokens config | Identity Hub + Settings (DUPLICATE) | Settings only | Remove from Identity Hub |
| Active sessions list (live) | Identity Hub | Settings → Maintenance | Move |
| OAuth Server config (issuer, lifetimes, DPoP) | Identity Hub | Settings → OAuth Providers | Move |
| MFA enforcement | Identity Hub | Settings → Auth Policy | Move |
| SSO connections | Identity Hub | Settings → OAuth Providers | Move (renamed section "OAuth & SSO") |
| Authentication methods (password, magic-link, passkeys, social) | Identity Hub | Identity Hub | Stays — this IS Identity |
| Email Delivery | Settings only | Settings only | Stays. Optionally: read-only mirror in Identity (skip for launch) |
| Server (URL, CORS, port) | Settings | Settings | Stays |
| Auth Policy (password rules, reset TTL) | Settings | Settings | Stays |
| Audit & Data (retention, purge) | Settings | Settings | Stays |

### Final structure after cleanup

**Identity Hub** (WHO can authenticate):
- Authentication methods
  - Password rules summary (read-only, full config in Settings → Auth Policy)
  - Magic link summary
  - Passkeys summary
  - Social providers (Google, GitHub, Apple, Discord) — list + status

**Settings** (HOW the system operates):
- Server (base URL, CORS, port)
- Sessions & Tokens (cookie + JWT lifetimes, signing key rotation)
- Auth Policy (password rules, magic link TTL, password reset)
- OAuth & SSO (OAuth Server config, OAuth Providers, SSO connections)
- Email Delivery (provider, sender, test send)
- Audit & Data (retention, purge, export-coming-soon)
- Maintenance (active sessions, device codes, danger zone)

### Files affected

- `admin/src/components/identity_hub.tsx` — strip moved sections, keep authentication-methods only
- `admin/src/components/settings.tsx` — receive moved sections, ensure SECTIONS array order is logical
- `admin/src/components/sessions_active.tsx` (or wherever live-session list lives) — move from Identity to Settings
- Possibly create `admin/src/components/oauth_sso.tsx` consolidating OAuth Server + OAuth Providers + SSO into one Settings section

### Acceptance

- No duplicate sections between Identity and Settings
- Source of truth for each setting is unambiguous
- A YC reviewer installing shark and clicking around for 5 minutes sees coherent IA, not "wait, I just edited this in two different places"
- Smoke suite stays GREEN

## Edit 4 — Get-Started full `/impeccable` rebuild (~6-8h)

**Why full rebuild, not patch:** per code audit, current `get_started.tsx` has 12 steps, 10 of which are human-only OAuth content (Sign-in button, callback route, useShark hook). Adding an "Agent Onboarding" section beneath that mess (the prior plan from Wave 1 Edit 5) leaves a 1/10 surface at maybe 5/10. Day-1 users post-magic-link land here. Worth the rebuild.

**Process:** use the `/impeccable` skill (frontend-design discipline) for this rebuild. Per the user's directive: "for every new component or screen remember to use that workflow."

`/impeccable` workflow steps:
1. Skill invocation: `/impeccable` (or `frontend-design`)
2. Provide design intent: "Get-started flow for SharkAuth, agent-first product onboarding. Audience: developer who just installed shark, post-magic-link login, evaluating in 5 minutes."
3. Skill produces design proposal — review, iterate
4. Generate component code following `.impeccable.md v3` lock (monochrome, square, editable)
5. Verify against existing dashboard tokens
6. Smoke + visual review

### Target structure for the rebuild (give as input to /impeccable)

**Three tracks, user picks ONE based on what they're building:**

```
Welcome to SharkAuth.

What are you building today?

  [ ] Agent integration  →  Full agent onboarding flow
  [ ] Human auth         →  OAuth/SSO/magic-link setup
  [ ] Both               →  Agent + Human (most platforms)

(Selection is sticky — user can switch tracks but the page remembers.)
```

**Agent integration track (PRIMARY for moat positioning):**
1. Register your first agent — link to /agents/new + CLI snippet `shark agent register --name my-agent`
2. Get DPoP-bound credentials — Python: `prover = DPoPProver.generate()` with one-line explanation of what jkt thumbprint means
3. Define delegation policy — link to agent detail Delegation Policies tab + CLI: `shark agent policy set <agent-id> --may-act <other-agent-id> --scope email:read`
4. Issue your first token — Python: `client.oauth.get_token_with_dpop(...)` (uses Wave 2 SDK)
5. Try delegation chain — Python: `client.oauth.token_exchange(...)` to derive a sub-token
6. Verify the audit trail — link to /audit?delegation=true
7. (Optional) Connect a credential vault — link to Vault tab

**Human auth track (for completeness):**
1. Configure email delivery (Settings → Email Delivery)
2. Configure OAuth providers (Google/GitHub/etc.)
3. Test magic-link login flow
4. Install React/JS SDK (link to docs)
5. Mount auth in your app

**Both track:** linear walk through Agent track (1-7) then Human track (1-5). Agent first because the moat.

### Auto-completion probes (preserve from current implementation)

The existing `get_started.tsx` has a system that polls `/admin/...` endpoints to auto-mark steps complete when the underlying state is done. Preserve this pattern. Each step has a probe; when the probe returns truthy, the step shows as ✓ done.

Example probes for the new agent track:
- "Register your first agent" → `GET /api/v1/agents?limit=1`, complete if `data.length > 0`
- "Get DPoP-bound credentials" → check audit log for `oauth.token.issued` event with DPoP metadata
- "Define delegation policy" → `GET /api/v1/agents/{id}/policies`, complete if any exist
- "Try delegation chain" → check audit log for `oauth.token.exchanged`
- "Verify the audit trail" → mark complete on first /audit page visit

### `/impeccable` discipline directive (process change, zero direct cost)

User stated: "for every new component or screen remember to use that workflow."

**Codify in playbook:** every component or screen created from this point forward (including Wave 1.5's `me_agents.tsx`, Wave 1.7's `coming_soon.tsx`, Wave 3's demo report HTML, any future surface) goes through `/impeccable` review before merge. Skip allowed only for sub-30-line bugfix patches that don't introduce new surfaces.

Rationale: launch is 24-48h away. The dashboard is the moat surface. Quality discipline at the per-component level beats trying to retroactively fix taste later. Anchoring on `/impeccable` ensures no surface ships at <8/10 craft.

## Definition of done for Wave 1.7

- Dev-email banner shows correctly based on actual config state
- Proxy nav entry visible, route shows ComingSoon placeholder, real component preserved at `/proxy-dev` env-gated
- Compliance nav entry renamed to "Exporting logs", route shows ComingSoon
- Branding nav entry route shows ComingSoon
- Identity Hub stripped to auth-methods-only
- Settings tab restructured with no duplicates from Identity
- Get-started flow rebuilt via `/impeccable` craft, agent track is the primary path
- Auto-completion probes preserved + extended to new steps
- Smoke suite GREEN (375 PASS)
- Visual smoke: walk through dashboard as fresh user, score "is this a serious product?" subjectively at ≥7/10

## Acknowledged trade-offs

- **Time:** Wave 1.7 alone is 10-12h. Combined with mandatory Waves 0-4 + 1.5, total mandatory is ~40h. This is at the upper bound of 48h calendar window. Sleep budget: ~6h. Real CC time over 2 days: ~30h. **Slip risk: real.**
- **Feature breadth cut:** proxy, compliance, branding all become coming-soon. Cleaner story; fewer features in screenshots.
- **The `/impeccable` discipline tax:** every new component costs more upfront in design review. The launch is the right time to install this discipline — never cheaper than now.

## Cuts available within Wave 1.7 if budget breaks

If Sunday afternoon hits and Wave 1.7 isn't done, cut in this order:
1. Identity/Settings cleanup (defer to W18 — looks bad but doesn't break)
2. Get-started rebuild → revert to original Wave 1 Edit 5 plan (90-min agent-track patch on top of existing 12 steps)
3. Coming-soon placeholders → ship but with minimal text (no descriptions, just "coming v0.2")

Do NOT cut: dev-email bug fix. That one is a top HN comment risk.
