# Launch Scope Cuts (What's NOT in the Monday launch)

This file is the explicit list of features intentionally absent from the Monday launch positioning. Each cut has a reason and a re-introduction plan.

If something is in the codebase but NOT mentioned in launch posts, it's because of one of three reasons:

1. **Has known bugs** — shipping launch traffic into it = bad top HN comments
2. **Not finished enough** — UX cliff that drops users
3. **Not in the pitch focus** — adds noise, dilutes the customer category

## CUT 1 — Proxy + Auth Flow Builder (coming-soon placeholder, real ship W18+)

**Status as of 2026-04-26:** Phase 6 shipped 2026-04-19 with 244 smoke PASS. Subsequent integration revealed bugs — currently 2 P0 + 4 P1 + 6 P2 documented in `INVESTIGATE_REPORT.md`.

**Why cut from launch:**
- Launch traffic into a P0-bug surface = top HN comment "tried it, proxy crashes" within 2 hours
- Proxy is not in the locked customer pitch ("auth for products that give customers their own agents") — adjacent capability, not the moat

**Action for Monday (UPDATED per user direction):**
- KEEP proxy nav entry visible in sidebar — honest scoping in UI
- Route to `<ComingSoon />` placeholder describing what ships in v0.2
- Real proxy component preserved in tree (renamed `proxy_manage_real.tsx`), accessible at `/proxy-dev` only when `VITE_FEATURE_PROXY=true`
- README "Roadmap" section names proxy
- HN post body: "Proxy + Auth Flow Builder land in W18 after regression fixes (coming-soon placeholder in launch build)"

**Why coming-soon over hidden:**
- Users see what's planned — builds anticipation
- Doesn't break sidebar layout
- Doesn't surprise users who'd been told "proxy ships in v0.2"
- Reads as honest scoping discipline, not as missing feature

**Implementation:** see `01d-wave1-7-ui-cleanup-and-impeccable-rebuilds.md` Edit 2 (shared `<ComingSoon />` component + route swap + import-path rename).

**Re-introduction plan:**
- W18 (week after launch): triage P0 bugs, ship fixes
- W19: regression-test proxy against smoke suite, ship to ≥375 PASS baseline
- W20: swap ComingSoon back to real component on `/proxy`, post "proxy + auth flow builder shipping in v0.2" on HN follow-up + Twitter
- Separate news cycle 2-3 weeks post-launch

## CUT 2 — Wave 4.5 Layer 4+5 (bulk-pattern revoke, vault cascade) — soft-deferred

**Status:** designed in `01c-wave1-6-bulk-pattern-and-vault-cascade.md`, not implemented.

**Why soft-cut:**
- Layers 1-3 ship Monday and the pitch can credibly claim "5 layers" with W+1 commitment for 4+5
- Wave 1.6 budget (~7h) lands Wed-Thu post-launch alongside Video B recording
- If Wave 1.6 slips past Thursday, YC application uses "4 of 5" honestly

**Re-introduction plan:** Wed-Thu post-launch per Wave 1.6 file. Public roadmap commits to W+1 even if recording slips to W+2.

## CUT 3 — Setup-flow polish (deferred to W18)

**Status:** dashboard cliff fixes (extend setup token TTL, SMTP health check, inline policy editor) were originally Wave C in the design doc. Cut explicitly when Wave 1 (UI moat exposure) was prioritized.

**Why cut:**
- Wave 1 + Wave 1.5 changes ALREADY address the worst cliff (get_started.tsx going from 1/10 to 7/10 with agent track, plus the bug-fix coverage)
- Remaining setup-token TTL bug + silent SMTP fail are documented but lower-frequency than the dashboard moat-visibility issues
- Launch post explicitly says "dashboard onboarding polish lands W18"

**Re-introduction plan:** W18 (week after launch). Document in launch retrospective Friday post.

## CUT 4 — TypeScript SDK (deferred to W18 follow-up)

**Status:** Python SDK ships at launch with 5 killer methods (Wave 2). TypeScript SDK has minimal React-auth-focused surface, NOT agent-native methods.

**Why cut:**
- Python ships fastest from current foundation per Wave 2
- TypeScript follow-up is its own news cycle ("now in TS too")
- MCP server builders (the customer wedge) are split TS/Python — Python gets ~half, TS follow-up gets the other half W+1 to W+2
- Better to ship one polished SDK than two half-baked

**Re-introduction plan:** W18-W19 ship TS SDK with same 5 methods. Mirror the Python release as a "TypeScript now ships too" post on HN, Twitter, Discord. Two separate news cycles from one engineering investment.

## CUT 5 — Hosted/cloud tier

**Status:** mentioned in pitch as future revenue. NOT shipping at launch.

**Why cut:**
- Open-source self-hosted is the wedge for distribution and trust
- Hosted tier requires: billing, multi-tenant infra, ops on-call, support — none of which a solo founder ships in 24-48h
- Pitch is honest: "Open source. Self-hosted free. Cloud paid (later)."

**Re-introduction plan:** post-YC-batch, post-funding. Hosted tier is the path to revenue and is part of the YC application narrative ("self-hosted free is the wedge; hosted/cloud paid is the revenue plan"). Don't promise a date.

## CUT 6 — Enterprise features (audit export, SAML SSO, custom RBAC)

**Status:** SAML SSO and basic RBAC ship. Custom RBAC, audit export to S3/Datadog, SOC2 attestation are NOT included.

**Why cut:**
- Enterprise features are sales-driven — built when a real enterprise customer asks
- Pre-launch, asking "should we ship audit export to S3?" without a customer is YAGNI
- The launch is to validate the wedge, not to win enterprise deals

**Re-introduction plan:** when a paid pilot from a Replit/Cursor/Lovable-tier team requests it. Build to demand, not to spec.

## CUT 7 — Harness self-register skill + MCP server (eureka 2026-04-26, scoped to W19+)

**Status:** Eureka logged 2026-04-26. Backend primitive (DCR `POST /oauth/register`, RFC 7591) already ships at `internal/api/router.go:755`. Wrapper distribution (Claude Code skill + `shark serve --mcp` subcommand) is post-launch only.

**Why cut from Monday launch:**
- Monday launch leads on **five-layer revocation** ("auth for products that give customers their own agents"). Adding "we also ship a Claude Code skill" to launch posts dilutes the security-moat framing YC reviewers respond to.
- Skill / MCP wrapper is a **sales objection-neutralizer** used inside deals, not a primary pitch. Mixing the two messages in one launch dilutes both.
- Wrappers are 4-5 days CC build (W19a-d). Monday launch budget is exhausted by Wave 1.5 / 1.6 / 1.7 / 1.8 + Wave 2 + Wave 3.
- Separate news cycle 2-3 weeks post-launch buys a second wave of HN attention.

**Re-introduction plan:**
- W19a: shark CC skill (1 day CC) — `skills/shark-cc/SKILL.md` + bash setup + project scaffolding
- W19b: `shark serve --mcp` subcommand (2-3 days CC) — Go, MCP stdio + SSE, 5 initial tools
- W19c: Skill marketplace + MCP registry submission packets (0.5 day human)
- W19d: Demo screencast — skill install → agent self-registers → first DPoP token in 30s (0.5 day)
- W20: Single news-cycle post: HN + Twitter + Discord + LinkedIn ("now in your Claude Desktop")

**Full design sketch:** `09-post-launch-harness-skill-eureka.md`.

**What launch communications can and cannot say:**
- Monday HN post / Twitter thread: silent on skill + MCP wrapper. Five-layer revocation only.
- Monday README "Roadmap" section: may name "Claude Code skill + MCP server (W19+)" as a one-line item alongside proxy / TS-SDK / hosted tier.
- YC application long-form: pushback answer in `07-yc-application-strategy.md` may include the secondary skill-as-distribution paragraph, but ONLY as a follow-up after the orchestration-vs-auth primary answer.

## What this list means for launch communications

The HN post body (in `04-wave4-launch.md`) explicitly says:

```
Honest scoping: layers 1-3 ship today. Layers 4-5 ship within 2 weeks of
launch. CLI/SDK story is sharp; dashboard onboarding polish lands W18.
Proxy + Auth Flow Builder land in W18+ after regression fixes. TypeScript
SDK lands W18-W19. Hosted/cloud tier later.
```

Honest scoping wins more credibility than hidden bugs.

## Pre-launch verification (Sunday night)

Before going live Monday:

- [ ] Proxy nav entry visible BUT route shows ComingSoon placeholder
- [ ] `/proxy-dev` route works only when `VITE_FEATURE_PROXY=true`
- [ ] Real proxy code renamed to `proxy_manage_real.tsx` (preserved in tree)
- [ ] Compliance nav entry renamed to "Exporting logs" → ComingSoon placeholder
- [ ] Branding nav entry → ComingSoon placeholder
- [ ] No README link goes directly to proxy/compliance/branding live demos
- [ ] HN post body matches the scope-cuts list
- [ ] Dev-email banner bug is FIXED (verified by switching mode and refreshing)
- [ ] Identity Hub stripped of duplicates (no Sessions & Tokens duplicate, no OAuth Server duplicate)
- [ ] Settings tab restructured to absorb moved sections coherently
- [ ] Get-started flow rebuilt via `/impeccable` craft (or fallback patch if Wave 1.7 slipped)
- [ ] CLI output polished per Wave 1.8 (or fallback to current state if 1.8 slipped)
