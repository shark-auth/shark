# SharkAuth — 10-Day Launch Playbook

Today is 2026-04-24 (Day 0). YC P26 application deadline: 2026-05-04 (Day 10).
Companion to: design doc + YC application draft in same directory.

Operating principle: every day ends with a shipped artifact. Ties go to shipping. If a day's primary goal slips past EOD, it does not "roll over" — the rest of the calendar compresses around it.

Two parallel tracks:
- **T1 — Product & Release:** get v1.0 shippable, published, documented.
- **T2 — Demand & Distribution:** cold DMs, partner calls, launch prep, YC app.

The only non-negotiable deliverables: 3 named design partners, 1 demo video, 1 binary release, 1 YC application submitted by Day 10.

---

## Day 0 (today, 2026-04-24) — Foundation

**T1 goals:**
- Confirm LICENSE (MIT or Apache 2.0) at repo root. Commit if missing.
- Freeze feature scope. No new features until Day 10. Bug fixes and polish only.
- Create `release` branch from `main`. All shipping work happens here; `main` stays stable.
- Audit README.md against actual capabilities — remove any claim the audit matrix flags as MISSING or STUB.

**T2 goals:**
- Write cold-DM template (see appendix). Send 10 cold DMs to MCP server authors on GitHub. Priority targets: authors of MCP servers with >100 GitHub stars, anyone who has tweeted about MCP auth pain, anyone Anthropic has linked from their MCP docs.
- Create tracking sheet: partner name, DM sent date, reply status, call scheduled, installation status, quote.
- Register domain if not already. Buy sharkauth.com (or similar) today.
- Post on YC co-founder matching board. One paragraph: who you are, what you're building, what you need in a co-founder (cloud infra + GTM).

**Exit criteria:** 10 cold DMs sent. LICENSE committed. Release branch exists. Domain bought. Co-founder post live.

---

## Day 1 (2026-04-25) — Binary Release Pipeline

**T1 goals:**
- Write goreleaser config. Build darwin-amd64, darwin-arm64, linux-amd64, linux-arm64, windows-amd64.
- GitHub Actions workflow `.github/workflows/release.yml`: on `v*` tag, run goreleaser, publish Release with binaries + SBOM + checksums.
- GHCR container publish workflow: on `v*` tag, build multi-arch image, push to `ghcr.io/<org>/sharkauth:<tag>`.
- PyPI publish workflow for `sdk/python` (mirror of existing npm workflow).
- Tag `v0.9.0-rc.1` as dry run. Verify all artifacts land correctly.

**T2 goals:**
- Follow up on Day 0 DMs (the 30% that replied) — schedule 20-min calls in Day 2-3 window.
- Send 10 more cold DMs (different targets — Discord server owners, SaaS founders whose public roadmap mentions auth).
- Draft landing page copy. Keep it short: hero (one sentence), feature matrix (from audit), install command, "why we built this" paragraph, Discord + GitHub links.

**Exit criteria:** Binaries on GitHub Releases (pre-release tag OK). GHCR image pushed. 20 DMs out total. Landing copy drafted.

---

## Day 2 (2026-04-26) — Docs + OpenAPI

**T1 goals:**
- Generate or hand-write OpenAPI 3.1 spec covering all public endpoints (admin API + auth API + OAuth endpoints). Commit `docs/openapi.yaml`.
- Wire `/api/docs` route serving Scalar or Stoplight UI from the spec.
- Write 3 new docs pages: (1) "Drop SharkAuth in front of your MCP server in 5 minutes" — happy-path quickstart. (2) "Agent delegation with Token Exchange" — the feature no competitor has. (3) "Self-host vs. Cloud" — positioning doc, explicit about what ships now vs. later.
- Polish existing `docs/hello-agent.md` — remove typos, add sequence diagram.

**T2 goals:**
- Do 2-3 partner calls if scheduled. Script: 5-min product intro, 10-min watching them try to install, 5-min asking what broke. No pitch. Listen.
- After each call, send a "thank you + install guide" email with the quickstart link. Offer to pair on the install if they hit friction.
- Draft 1-minute Loom for YC application. Script: problem (10s), demo (40s), ask (10s).

**Exit criteria:** OpenAPI spec live at `/api/docs`. 3 new doc pages shipped. 2+ partner calls completed. YC video script drafted.

---

## Day 3 (2026-04-27) — Partner Polish + Demo Recording

**T1 goals:**
- Fix every bug and rough edge surfaced in Day 2 partner calls. Commit fixes to `release` branch.
- Record the 5-minute main demo: `shark serve` first-boot → MCP client registers via DCR → agent obtains token → agent delegates to second agent via RFC 8693 → proxy enforces scope → audit log shows the delegation chain. One take, no cuts if possible.
- Record the 1-minute YC application video: tighter, punchier, problem-then-demo-then-ask.

**T2 goals:**
- Partner installation push: if a partner from Day 0-2 calls has agreed to try it, pair with them live today. Goal: 1 real installation before EOD.
- Send 10 more cold DMs. At this point, prior DMs' non-responders can get a polite follow-up ("saw you work on X — 5-min look?").
- Draft HN launch post. Title: "Show HN: SharkAuth — open-source OAuth 2.1 with agent delegation (single binary)". Body: what it does in one paragraph, install command, what's different from Auth0/Clerk/Stytch, honest about what's missing (SAML IdP, Postgres).

**Exit criteria:** Both videos recorded. 1+ live partner installation. HN post drafted. 30 DMs sent total.

---

## Day 4 (2026-04-28) — Partner #2 + Landing Page Live

**T1 goals:**
- Ship the landing page. Static Next.js or Astro on Vercel/Cloudflare Pages. Must have: hero, 3-card feature matrix, install command (one line), demo video embed, Discord/GitHub buttons, cloud waitlist signup form (email only).
- Wire up cloud waitlist storage (simple — Postgres on Neon, or an Airtable form).
- Set up Discord server with channels: `#announcements`, `#install-help`, `#feedback`, `#showcase`.
- SEO basics: og tags, sitemap.xml, robots.txt, Google Search Console verification.

**T2 goals:**
- Land partner #2 via a live install session. At this point one of the 30+ DMs should have converted.
- Write Product Hunt launch draft (different voice from HN — more polished, less technical).
- Write LinkedIn launch post. Short, founder voice, the Insforge comment style — pain, product, ask.
- Begin drafting YC written application (use `raul-main-yc-p26-application-20260424-134836.md` as starting point).

**Exit criteria:** sharkauth.com live with waitlist. Discord server public. Partner #2 confirmed. YC app draft in progress.

---

## Day 5 (2026-04-29) — Midpoint Review + Pivot Gate

**Checkpoint: if by EOD we have fewer than 2 confirmed partner installations, pivot from Approach A to Approach C (OSS-first launch narrative). Reallocate Day 6-9 toward star count and install count instead of partner hunting.**

**T1 goals:**
- Burn down remaining bug list. Release-branch must be `v1.0.0-rc.1` quality by EOD.
- Tag `v1.0.0-rc.1`. Re-run release pipeline. Verify binaries and images land.
- Write install-verification script — curl command that downloads binary, starts it, hits `/healthz`, prints "SharkAuth ready" or specific error. This is the "does it actually work on a fresh machine" canary.

**T2 goals:**
- Partner #3 push. Any warm lead gets a live-install slot today.
- Finalize YC application written sections. Send to 1-2 trusted readers (mentor, peer founder) for brutal feedback.
- Schedule the HN launch window. HN front-page dynamics favor Tuesday-Thursday 7-10am PT. Day 7 (Wednesday 2026-05-01) is the target.
- Write the Insforge-founder LinkedIn comment — reuse the drafted comment + updated traction numbers.

**Exit criteria:** `v1.0.0-rc.1` tagged. 3 partners confirmed OR Approach C pivot declared. YC app feedback received.

---

## Day 6 (2026-04-30) — Launch Eve

**T1 goals:**
- Final polish day. No new features. Every change is a bug fix or doc improvement.
- Verify install-verification script on fresh VMs: Ubuntu 22.04, Debian 12, macOS 14, Windows Server 2022. Catch environmental bugs before HN users hit them.
- Pre-stage all launch assets: README badges, landing-page copy, Discord welcome message, HN post body, Product Hunt copy, Twitter/X thread, LinkedIn post.
- Tag `v1.0.0`. Run final release pipeline. This is the version that ships tomorrow.

**T2 goals:**
- Schedule all launch posts for Day 7 morning (7am PT). Use Buffer or manual calendar reminders.
- Brief the partners: "We're launching tomorrow morning. A nice comment on HN or a retweet would mean a lot." No pressure, but every friend in the launch window matters.
- Final YC app read-through. Fill in traction numbers (partner count, star count from pre-launch crawl, waitlist count).
- Prepare launch day ops kit: Discord moderation macros, template HN replies, a running FAQ doc for repeated questions.

**Exit criteria:** v1.0.0 tagged. All launch posts scheduled. All partners briefed. YC app 95% done.

---

## Day 7 (2026-05-01) — LAUNCH

**This is the 16-hour workday. Plan meals. Block the calendar.**

**Morning (7-10 AM PT):**
- HN submission. Do NOT ask for upvotes in DMs (HN auto-detects). Post organically; let partners find it and comment.
- Product Hunt post goes live.
- LinkedIn post (including the Insforge-founder comment on their post).
- Twitter/X thread.
- Announcement in Discord.

**All-day ops:**
- Respond to every HN comment within 30 min for the first 6 hours. Take every technical question seriously. Admit gaps plainly ("we don't ship SAML IdP yet — Q3 roadmap").
- Monitor GitHub Issues, Discord, Twitter mentions. Triage: bug → `release` branch fix, question → docs update, partnership lead → follow-up later.
- Update waitlist and star counters in a pinned tweet every few hours. Momentum is a signal.

**Evening (6-10 PM PT):**
- Write launch day retrospective doc. Star count, install pings (via telemetry), waitlist signups, HN position peak, notable comments, conversion bottlenecks.
- Update YC application traction section with real numbers from the day.

**Exit criteria:** Launch executed. Retro written. Real numbers in YC app.

---

## Day 8 (2026-05-02) — Post-Launch Momentum + YC Polish

**T1 goals:**
- Ship hotfix release (`v1.0.1`) for any issues the launch surfaced. Every Day-7 bug report gets triaged by EOD.
- Update README with launch metrics badge (stars + install count + Discord members).
- Write "Week in review" blog post (Monday 2026-05-05 publish). Establishes update cadence early.

**T2 goals:**
- Reply to every email, DM, and comment from the launch — personal replies, not templates.
- Identify the 3-5 most engaged new partners and schedule 30-min calls in the next week.
- YC application final polish. Fill every field. Upload videos. Attach any press mentions.

**Exit criteria:** v1.0.1 out. All launch-day DMs replied to. YC app ready to submit.

---

## Day 9 (2026-05-03) — YC Submission Eve

**T1 goals:**
- No code changes. Stop touching the codebase. Stabilize.
- Verify YC application once more. Read every answer out loud. Fix any sentence that makes you wince.
- Have one mentor + one peer founder do a final pass. Timebox each review to 45 minutes.

**T2 goals:**
- Reach out to any YC alum in your network (Insforge founder, anyone from the original comment thread) for a potential referral. Warm intro > cold submission.
- Prepare interview packet: 5-min demo video, full feature matrix, honest gap list, customer quotes, roadmap.
- Rest. Sleep 8 hours. Brain needs to be sharp for Day 10.

**Exit criteria:** YC app 100% done. Mentor reviews received. Referral reached.

---

## Day 10 (2026-05-04) — Submit + Celebrate

**Morning:**
- Submit YC application before noon PT. Do not submit at 11:59 PM on deadline day — submit early so the partners reviewing can see polish.
- Confirm submission email received.
- Post to Discord: "YC app is in. Here's what's next."
- Post to Twitter: short, humble, link to GitHub.

**Afternoon:**
- Write Week-2 plan. Whatever happens with YC, the product has users now and they need onboarding, support, and the next feature.
- Take the evening off.

**Exit criteria:** Submitted. Screenshot captured. Week-2 plan drafted.

---

## Fallback Calendars (if something slips)

**If Day 5 midpoint shows <2 partners:**
- Scrap Day 6 polish day; it becomes Day 2 of launch prep.
- HN launch moves to Day 6. Accept lower polish for earlier traction signal.
- YC app framing pivots from "design partners + traction" to "OSS flagship + stars + installs."

**If binaries break on a platform (Day 1-6 discovery):**
- Drop that platform from the Day 7 launch. Darwin and Linux are non-negotiable; Windows can slip.
- Note the gap honestly in the YC app.

**If a partner blocks on a missing feature (SAML IdP, SMS OTP, etc):**
- Do NOT build it in the 10-day window. Commit to a Q3/Q4 delivery in writing. Lose that partner if they insist; ship the roadmap commitment in the YC app instead.

**If HN front page doesn't land on Day 7:**
- Launch on Product Hunt on Day 8 as a recovery move.
- r/selfhosted, r/golang, r/selfhostedai on Day 8-9.
- YC app traction section leans on Discord growth + waitlist + design partners, not star count.

---

## Appendix: Cold DM Template

Subject: **MCP server auth in 5 min — would love 20 min of your time**

Body (personalize the first two sentences per recipient):

> Hey [Name], saw your work on [their MCP server / project]. I'm building SharkAuth — single-binary open-source OAuth 2.1 for MCP servers with native agent delegation (RFC 8693).
>
> Got 20 min this week? I'd love to watch you try to install it and find out what breaks. Zero pitch. You get a free install of agent-native auth; I get brutal feedback before I launch publicly next week.
>
> GitHub: <link>. Happy to hop on whenever works.
>
> Raul

Rules for the template:
- Always personalize the first two sentences. Generic DMs get ignored.
- No calendar link in cold DMs — it reads as presumptuous. Let them propose a time.
- Offer something concrete (a free install, early access to cloud, a Discord role).
- Keep it under 100 words total.

---

## Appendix: HN Launch Post (draft)

**Title:** Show HN: SharkAuth – open-source OAuth 2.1 with agent delegation (single binary)

**Body:**

> Hi HN — I've been building SharkAuth over the last 8 months to solve a specific pain: every MCP server ships with zero identity layer, and every available auth option is wrong for the agent era. Auth0 prices agent auth as a 50%-uplift add-on. Clerk and Stytch are SaaS-locked. Okta is enterprise-only. Keycloak is a Java-sized operational burden.
>
> SharkAuth is a single 20MB Go binary. It's a full OAuth 2.1 authorization server with RFC 7591 Dynamic Client Registration, RFC 9449 DPoP, RFC 8693 Token Exchange with native agent delegation chains (agent-A delegates to agent-B delegates to agent-C, with a full `act` claim audit trail), plus the identity primitives you actually need: passkeys, TOTP MFA, magic links, social login, multi-tenant orgs, RBAC, audit log, webhooks. Reverse-proxy mode drops auth in front of any existing HTTP service with zero code changes.
>
> Ships on SQLite. Postgres and hosted SharkAuth Cloud are Q3 2026. TypeScript, Python, and React SDKs. OpenAPI spec. MIT license.
>
> Install: `curl -sSL sharkauth.com/install | sh` (or `docker run ghcr.io/.../sharkauth`).
>
> Honest about gaps: SAML IdP mode, SMS OTP, and RFC 9728 Protected Resource Metadata are on the roadmap, not shipped yet. Postgres support is required before the hosted tier opens.
>
> I'm a solo founder, 18, in Monterrey. Happy to answer any technical or strategic question in the thread.
>
> Repo: <github>. Docs: <sharkauth.com/docs>. Discord: <link>.

---

## Appendix: Daily Scorecard Template

Record EOD every day Day 1-10:

```
Day X (date):
  T1 shipped: [bullets]
  T2 shipped: [bullets]
  Partners touched today: N  |  Partners confirmed (cumulative): N
  DMs sent today: N  |  DMs sent (cumulative): N
  GitHub stars: N
  Waitlist: N
  Discord members: N
  Blockers: [bullets]
  Tomorrow's priority: [one sentence]
```
