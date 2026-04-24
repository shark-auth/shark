# SharkAuth — Locked Decisions + Compressed Calendar + Cold-DM Kit

Generated 2026-04-24 after founder made taste decisions TD1-TD5.
External constraint: other accelerator deadline forces launch on 2026-04-29 (compresses calendar by 2 days).
Supersedes launch-playbook calendar; cross-reference design doc + YC app for non-calendar content.

---

## Locked Decisions (source of truth)

| ID | Decision | Rationale |
|----|---|---|
| TD1 | **Hybrid wedge.** YC title + HN post + demo video lead with RFC 8693 agent delegation. README + landing page keep broader value prop (SSO, passkeys, MFA, proxy, orgs). | Sharpest differentiation in the surfaces that matter for partner/user conversion; broader surface still helps mid-market discovery. |
| TD2 | **Launch Tuesday 2026-04-29 (Day 5).** Hard external constraint — other accelerator deadline. | Bonus: 5-day post-launch window before YC submit 5/4 instead of 3-day. |
| TD3 | **Defer pricing.** YC app says "OSS free forever self-host; hosted pricing finalized during batch." | Avoids price-taker trap. Keeps optionality for flat-rate vs per-MAU framing based on partner conversations. |
| TD4 | **Track both partner goals.** Personal commitment = 3 partners. YC app reports real numbers (whatever landed) on Day 10. | Aspirational target drives cold-DM velocity; honest framing in app. |
| TD5 | **v0.9.0 Day 6. v1.0.0 only after 2 weeks clean API.** | Admin API reshaped once in 6 months. SDK types hand-written. SemVer freeze premature. |

---

## Compressed Calendar (Day 0 → Day 10)

### Day 0 — today, 2026-04-24 (Thursday)

**Hard fixes (must-ship EOD):**
1. `LICENSE` file at repo root (MIT). README references it; file doesn't exist.
2. Repo-root cleanup. Move `.log`, `.db`, `.jsonl`, `spoofme.md`, `cj*.txt`, `mfacj.txt`, `rotate.log`, `del*.log`, `show.log`, `list.log`, `rot.log`, `w15_server.log`, `server.log`, `app.log` to `.gitignore` + delete tracked. Move 12 loose PNGs to `docs/images/`. Move `HANDOFF.md`, `SMOKE_TEST.md`, `LANE_D_SCOPE.md`, `PROXYV2.md`, `PHASE3.md`, `CLOSE_GAPS_PLAN.md`, `DASHBOARD_DX_EXECUTION_PLAN.md`, `PLAN_UPDATABLE_SETTINGS_AND_AI_GAPS.md`, `MTESTS.md`, `newtests.md`, `FRONTEND_WIRING_STATUS.json`, `FRONTEND_WIRING_GAPS.md`, `BACKEND_WIRING_GAP_REPORT.md`, `DEVEX_REVIEW_2026-04-20.md`, `CHANGELOG.internal.md`, `SECURITY_AUDIT.md`, `CLAUDE_SEC_ANALYSIS.md`, `SECRETS.md`, `runes-*` to `docs/internal/`. Keep root: README, LICENSE, CHANGELOG, CONTRIBUTING, SECURITY, DESIGN, PROJECT, STRATEGY (at most).
3. Atomic refresh-token rotation. `internal/oauth/store.go:258-265` — replace with single `UPDATE oauth_tokens SET revoked_at=? WHERE request_id=? AND token_type='refresh' AND revoked_at IS NULL RETURNING id` check `RowsAffected()`.
4. `SCALE.md` at repo root. Document 4 in-memory stores (DPoP replay cache at `dpop.go:48-78`, device rate map at `device.go:33-55`, per-IP limiter at `middleware/ratelimit.go:46-84`, per-key limiter at `apikey.go:121-162`). State Q3 2026 Postgres + shared cache roadmap. Strike "stateless-safe" from design doc's claims in `raul-main-launch-yc-design-*.md`.
5. Buy domain if not already bought.

**Demand work:**
- Send 10 cold DMs using template in appendix below.
- Post on YC co-founder matching board.
- Start partner tracking sheet.

**Exit criteria:** LICENSE committed, repo root clean, atomic rotation shipped, SCALE.md live, 10 DMs out, domain bought.

---

### Day 1 — 2026-04-25 (Friday) — Binary Release Pipeline

**T1:**
- `.goreleaser.yml`: darwin-amd64, darwin-arm64, linux-amd64, linux-arm64, windows-amd64. SBOM + cosign signing.
- `.github/workflows/release.yml`: on `v*` tag → goreleaser + upload.
- `.github/workflows/ghcr.yml`: on `v*` tag → buildx multi-arch → push `ghcr.io/<org>/sharkauth:<tag>`.
- `.github/workflows/publish-python-sdk.yml`: on `py-sdk-v*` tag → twine upload to PyPI.
- Tag `v0.9.0-rc.0` as dry run. Verify all three artifacts land.

**T2:**
- Follow up on Day 0 DMs. Schedule partner calls for Day 2.
- Send 10 more cold DMs.
- Draft landing page copy.

**Exit criteria:** Binaries publish on tag. GHCR image publishes on tag. PyPI pipeline exists. 20 DMs out.

---

### Day 2 — 2026-04-26 (Saturday) — Docs + Partner Calls

**T1:**
- OpenAPI 3.1 spec — **scope cut per autoplan finding: 30 endpoints covering hello-agent path + admin proxy-rules CRUD. Not all 200.** `docs/openapi.yaml` + `/api/docs` Scalar UI.
- 3 new doc pages:
  - `docs/mcp-5-min.md` — "Drop SharkAuth in front of your MCP server in 5 minutes" (delegation-led).
  - `docs/agent-delegation.md` — the RFC 8693 chain demo (the wedge).
  - `docs/self-host-vs-cloud.md` — explicit positioning.
- Ship `shark doctor` CLI command. Checks config, DB writability, JWKS keys, port bind, base_url reachability, admin key presence. `cmd/shark/cmd/doctor.go`.

**T2:**
- 2-3 partner calls if scheduled. Script: 5-min intro, 10-min watch install, 5-min broken-stuff feedback. No pitch.
- Unify SDK naming: pick `@sharkauth/*` scope. Rename `sdk/typescript/` exports, update `packages/shark-auth-react/package.json`, republish under new name on Day 4.

**Exit criteria:** OpenAPI at `/api/docs`. 3 new doc pages. `shark doctor` command ships. SDK naming unified. 2+ partner calls done.

---

### Day 3 — 2026-04-27 (Sunday) — Videos + Partner Installs

**T1:**
- Fix every bug from Day 2 calls.
- Record main demo (5 min): `shark serve --dev` → MCP client DCR → agent-A gets token → agent-A delegates to agent-B via token-exchange → proxy enforces scope → audit log shows `act` chain.
- Record YC app video (1 min): tight — problem 10s, delegation demo 40s, ask 10s.

**T2:**
- Live-install with partner #1 if they agreed.
- Send 10 more follow-up DMs (non-responders get "saw you work on X" nudge).
- Draft HN post + PH post + LinkedIn post + Twitter thread. All lead with delegation (TD1).

**Exit criteria:** Both videos recorded. 1+ confirmed partner install. Launch posts drafted. 30 DMs sent total.

---

### Day 4 — 2026-04-28 (Monday) — Launch Eve

**T1:**
- Ship landing page on sharkauth.com. Static Next.js/Astro on Vercel. Hero line: "OAuth 2.1 + RFC 8693 agent delegation. One binary. Zero vendor lock-in." Below fold: broader feature matrix (TD1 hybrid). Install one-liner. Demo video. Waitlist form. Discord + GitHub buttons.
- Discord server live. Channels: announcements, install-help, feedback, showcase.
- SEO basics: og tags, sitemap, robots.txt.
- Tag `v0.9.0-rc.1`. Verify all release artifacts.

**T2:**
- Finalize YC app draft. Send to 1-2 trusted readers for brutal feedback.
- Brief partners on Day-5 launch: "A comment on HN tomorrow would mean the world."
- Pre-schedule all launch posts for Tuesday 7am PT.

**Exit criteria:** Landing page live. Discord public. `v0.9.0-rc.1` tagged. YC app 95% done.

---

### Day 5 — 2026-04-29 (Tuesday) — LAUNCH DAY

**Morning 7-10am PT:**
- Tag `v0.9.0`. Release pipeline runs. Binaries + container + SDKs publish.
- HN submission. **Title: "Show HN: SharkAuth – open-source OAuth 2.1 + RFC 8693 agent delegation (single binary)".** (Delegation-led per TD1.)
- Product Hunt live.
- LinkedIn post + Insforge-founder comment.
- Twitter thread.
- Discord announce.

**All-day ops:**
- Respond to every HN comment within 30 min for first 6 hours.
- Admit gaps plainly. Point to SCALE.md and roadmap when asked about horizontal scale.
- Triage: bug → hotfix, question → docs update, partnership lead → Day 6 follow-up.
- Update pinned waitlist + star count every 2 hours.

**Evening:**
- Launch retrospective doc. Real numbers (stars, pings, waitlist, HN peak, notable comments).

**Exit criteria:** `v0.9.0` tagged + launched. Retro written.

---

### Day 6 — 2026-04-30 (Wednesday) — Post-Launch + YC Patch

**T1:**
- Ship `v0.9.1` hotfix for launch-day bug reports.
- README badges: stars + install count + Discord members.

**T2:**
- Reply to every email + DM + comment from launch. Personal replies.
- Identify 3-5 most engaged new leads → schedule 30-min calls for Day 7-8.
- Update YC app "How many users?" section with real numbers from launch day. Past tense. Specificity over aspiration (per CEO-lens feedback).
- Promote audit matrix to linked appendix in YC app (per autoplan finding).
- Rewrite YC app pricing paragraph per TD3: "OSS free forever self-host; hosted pricing finalized during batch based on partner conversations."
- Rewrite YC app version reference: `v0.9.0` shipped; v1.0.0 gated on API stability.

**Exit criteria:** Hotfix out. All DMs replied. YC app updated with real launch data + appendix + pricing + version language.

---

### Day 7 — 2026-05-01 (Thursday) — YC App Polish

**T1:**
- Partner call #1 (schedule from Day-6 leads).
- Documentation touch-ups based on Day-5/6 confusion reports.

**T2:**
- YC app read-through. Fix anything that makes you wince.
- Write "Week in review" blog post for Monday publish.
- Begin YC interview prep: anticipate 10 hardest questions, draft answers.

---

### Day 8 — 2026-05-02 (Friday) — YC App Final

**T1:**
- Partner call #2.
- Stop touching code. Stabilize.

**T2:**
- Two YC-app readers: one mentor, one peer founder. 45-min timeboxed reviews each.
- Any YC-alum warm intro attempt — reach out to Insforge founder + anyone from the comment thread.

---

### Day 9 — 2026-05-03 (Saturday) — Eve

**T1:**
- Zero code.
- Verify YC app once more. Read out loud. Fix every wince.

**T2:**
- Prepare interview packet: 5-min demo video, feature matrix, gap list, customer quotes, roadmap.
- Rest. Sleep 8 hours.

---

### Day 10 — 2026-05-04 (Sunday) — SUBMIT

**Morning:**
- Submit YC P26 before noon PT.
- Discord post.
- Twitter humble note.

**Afternoon:**
- Week-2 plan drafted.
- Evening off.

---

## Revised YC App Sections (drop-in replacements)

Three sections from `raul-main-yc-p26-application-20260424-134836.md` need updates based on locked decisions.

### REPLACE §"Describe what your company does in 50 characters or less"

```
Open-source agent delegation for MCP servers.
```

(Reason: TD1 hybrid — YC title leads with delegation. Broader value prop comes in "What is your company going to make?" section.)

### REPLACE §"Do you have revenue?"

```
No. SharkAuth is open-source and free forever for self-hosted deployments. Hosted SharkAuth Cloud is blocked today on PostgreSQL support (Q3 2026) — pricing will be finalized during the batch based on partner conversations, but the shape is "free OSS self-host, paid hosted tier for teams that don't want to operate it themselves." The Supabase / Postgres.js model is the target: own the developer narrative first, monetize the hosted tier second.
```

(Reason: TD3 defer pricing.)

### REPLACE §"How far along are you?"

Keep the existing paragraph about 38/54 capabilities and file-level evidence. CHANGE the last sentence from:

> "Full feature matrix with file-level evidence available on request."

TO:

> "Full feature matrix with file-level evidence attached as Appendix A to this application. We shipped v0.9.0 publicly on 2026-04-29; v1.0.0 is gated on two weeks of partner usage without breaking changes — we'd rather keep SemVer honest than marketing-first."

(Reason: TD5 v0.9.0 + autoplan finding "promote audit matrix to appendix.")

### ADD §"Appendix A: Feature Matrix" (new section, end of YC app)

Paste the audit matrix tables from the design doc (lines 66-160 of `raul-main-launch-yc-design-20260424-134836.md`). 54 rows. File-level citations. Status labels.

### UPDATE §"How many users do you have?" on Day 6

Rewrite in past tense with Day-5-launch real numbers. Template:

> "As of 2026-04-30, we shipped v0.9.0 publicly on Tuesday. In the first 24 hours: [N] GitHub stars, [N] telemetry-confirmed `shark serve` installs, [N] waitlist signups for hosted cloud, [N] HN peak position. [N] design partners are running SharkAuth in non-demo environments: [list with quotes if attributable]."

---

## Revised Design Doc Wedge Section

From `raul-main-launch-yc-design-20260424-134836.md`, REPLACE §"Target User & Narrowest Wedge" sub-bullet "Narrowest wedge (first 30 days):" with:

> **Narrowest wedge (launch-day messaging, per TD1):** "Drop SharkAuth in front of your MCP server — get RFC 8693 agent delegation chains in 5 minutes. OAuth 2.1 + DCR + DPoP + SSO + passkeys + MFA + orgs + RBAC + audit are all included." Lead every customer touchpoint (YC title, HN post, demo video) with the delegation sentence. README and landing page keep the broader feature list but in supporting position.

---

## Cold-DM Kit (ready to send TODAY)

### Target list — start here

Find these on GitHub + Twitter + Discord. 10 targets for today, 10 more tomorrow.

1. MCP server authors with >100 GitHub stars. Search `site:github.com mcp server go OR typescript OR python`.
2. Authors of MCP adapters for popular services (Linear, Notion, GitHub, Slack MCP wrappers).
3. Anyone who's tweeted "MCP auth" or "agent auth" pain in the last 60 days.
4. Maintainers of agent frameworks (LangChain, CrewAI, Smolagents, OpenHands) — not agent users, the maintainers.
5. Founders of AI agent products you admire who are likely hand-rolling auth.
6. Authors linked from Anthropic's official MCP docs examples.

### Template — personalize first 2 sentences

**Subject:** MCP auth in 5 min — 20 min of your time?

**Body:**

> Hey [Name], saw your work on [specific project / specific tweet / specific commit]. I'm building SharkAuth — open-source OAuth 2.1 for MCP servers with native RFC 8693 agent delegation chains (single Go binary, drops in front of any service, zero code changes).
>
> Got 20 min this week? I'd love to watch you try to install it and find out what breaks. Zero pitch. You get free agent-native auth; I get brutal feedback before I launch publicly on Tuesday.
>
> GitHub: https://github.com/<org>/shark (pasted after LICENSE + repo cleanup ships today).
>
> — Raul, 18, Monterrey

### Rules

- Personalize first two sentences. Generic DMs die.
- No calendar link. Reads presumptuous. Let them propose a time.
- Offer something concrete (free install, early Discord role).
- Under 100 words.
- Send via whatever channel they're active on: Twitter DM > GitHub discussion > email > LinkedIn.

### Tracking (copy into spreadsheet or Notion)

```
| Name | Project | Channel | DM sent | Reply | Call date | Installed | Quote-able? | Notes |
|------|---------|---------|---------|-------|-----------|-----------|-------------|-------|
```

### Follow-up cadence

- Day 0: initial DM.
- Day 2: no reply → short nudge ("saw you work on X, 5 min?").
- Day 4: no reply → final attempt with landing page link.
- After Day 4: drop. Focus time on responders.

### Success signal

- 30% open rate = normal.
- 5% reply rate = normal.
- 1-2% convert to a 20-min call = normal.
- 30 DMs × 1.5% = 0.45 calls = realistic 1 partner. 60 DMs = 1 call. To hit 3 partners you need ~90 DMs over 5 days, OR 1 warm intro that converts.

**This is why the YC-app partner goal is flexible (TD4) — cold-DM math is what it is.** Keep the personal commitment at 3; let the app report honestly.

---

## What Changed vs. Original Playbook

| Original | Revised | Why |
|---|---|---|
| HN launch Day 7 (Wed 5/1) | HN launch Day 5 (Tue 4/29) | External accelerator deadline |
| v1.0.0 on Day 6 | v0.9.0 on Day 5, v1.0.0 post-launch | TD5 (eng risk) |
| OpenAPI all 200 endpoints Day 2 | OpenAPI 30 endpoints Day 2 | Autoplan eng finding (realistic scope) |
| "Stateless-safe" claim in pitch | SCALE.md + honest framing | Autoplan eng finding (in-memory stores) |
| Pricing "$29/mo" in YC app | "OSS free + hosted pricing during batch" | TD3 |
| YC title unspecified | "Open-source agent delegation for MCP servers" | TD1 (delegation-led) |
| Day 0 soft work | Day 0 = LICENSE + repo cleanup + atomic rotation + SCALE.md + 10 DMs | Autoplan DX + eng hard-fixes |
| Partner goal = 3 hard | Partner goal = 3 personal / actual in YC app | TD4 |

---

## Immediate Next Action (right now)

1. Open terminal in repo.
2. `touch LICENSE` — paste MIT text. Commit.
3. `git rm --cached *.log *.db *.db-* *.jsonl resp.json` — stop tracking.
4. Write `SCALE.md` per spec above. Commit.
5. Fix `internal/oauth/store.go:258-265` atomic UPDATE. Test. Commit.
6. Send first 10 cold DMs before end of day.

Everything else on Day 0 flows from those 6 steps. Ship in that order.
