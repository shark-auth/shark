# Launch Week → YC Interview Playbook

> Companion to `07-yc-application-strategy.md`. That file = strategy. This file = day-by-day execution.
> Written 2026-04-27. Launch target 2026-05-05 (Monday). YC W26 application after launch traction lands.
> Goal: convert "strong narrative" 5-7% interview rate → "named adopters + HN top + InsForge integration" 15-25% rate.

---

## North Star Metric

**One number that decides interview odds: # of NAMED humans publicly running `shark serve` by application submission.**

Not GitHub stars. Not waitlist signups. Not HN points. Those are vanity. Named adopters = demand evidence. YC reads "0 users" as terminal. YC reads "@kettler at Smithery is using shark for X" as categorical jump.

**Target by application date:** ≥3 named adopters with public quote/handle.

---

## 8-Day Countdown

### Day -8 (Mon 04-28) — Lock the Story

**Morning (3h):**
- Finalize landing page hero per `shark_idea.md` § 2 + § 17 (unified-platform lead)
- Record 30-second concierge screencast. First 5 seconds = chain graph + DPoP proof reveal. NO talking head. Just the demo.
- Draft HN title: *"Show HN: SharkAuth – open source auth for humans and the agents they ship (Go, single 30MB binary)"*

**Afternoon (3h):**
- Walk 11-item pre-launch checklist (`PRE_LAUNCH_CHECKLIST.md`)
- `shark doctor` exits 0 on prod-like instance
- Test rollback to `051f6e5`

**Evening (1h):**
- DM InsForge founder. **Script (copy-paste, edit name):**

> Hey [name], you said shark looked awesome a few weeks back. I'm launching Monday and applying to YC W26 right after. Two asks:
>
> 1. Would you actually try `shark serve` on InsForge this week? If it works for 1 use case, would you say so publicly (tweet/LinkedIn)?
> 2. If it breaks, tell me what broke. I'll fix it in 24h.
>
> Either outcome helps. Honest no is fine. 30-min call if useful.

**STOP. Wait for reply before continuing.** This is the highest-leverage move of the week.

---

### Day -7 (Tue 04-29) — Cold DMs to MCP Authors

**Goal:** 20 personalized DMs sent. Target 3 yesses by Day -3.

**Targets (research handles first):**
- Smithery contributors (top 5 by GitHub commits)
- Cloudflare Agents team eng (Twitter/LinkedIn)
- Vercel AI SDK contributors
- LangChain MCP integration maintainers
- LlamaIndex MCP authors
- Anthropic MCP Discord active builders
- Top 10 servers on https://github.com/modelcontextprotocol/servers contributors

**DM template (Twitter/X DM, 200 chars):**

> Built shark — open source auth for human + agent in one Go binary. Saw your work on [SPECIFIC PROJECT]. Would 5 min of your time + a free deploy help save you from rolling DPoP? Demo: [screencast link]

**LinkedIn template (longer, 400 chars):**

> Hey [name], I saw your work on [project]. I'm launching SharkAuth Monday — auth for humans + agents in one self-hosted Go binary. RFC 8693 delegation chains, DPoP, cascade revoke. Built it because every MCP team I talked to was hand-rolling this badly.
>
> Would a free deploy + 30 min on Zoom be useful? If shark saves you 2 weeks, I'd love a public quote.

**Track in spreadsheet:** name, handle, project, sent date, replied, status.

---

### Day -6 (Wed 04-30) — Convert Replies → Adoption

- For every reply: get them on Zoom within 24h. Screen-share `shark serve`. Watch them deploy.
- **The bug-fix loop:** if they hit a bug, fix it live. Push commit. Re-deploy. Show velocity. This is itself a YC signal.
- After successful deploy: ask for public quote. Don't leave it implicit. **Script:**

> If this saved you time, would you tweet/post about it Monday when I launch? I'll send draft text — feel free to rewrite or skip if it's not useful for you.

**Quote template (provide as starting point):**

> Just deployed @sharkauth on [project]. RFC 8693 delegation + DPoP working in 60 seconds. Was about to spend a sprint rolling this myself.

---

### Day -5 (Thu 05-01) — Build Press Kit

**Press kit folder: `press-kit/`**

- 30-second screencast (.mp4, 1080p)
- 60-second walkthrough video (script below)
- 5 hero screenshots (admin dashboard, delegation chain canvas, audit log render, doctor output, demo HTML report)
- 3 code snippets (DPoP token, token exchange, cascade revoke) as syntax-highlighted PNGs for Twitter
- Logo (existing sharky-full.png + dark/light variants)
- One-liner pitch in 5 lengths: 10 / 30 / 55 / 100 / 280 chars
- Founder photo + bio (40 words)

**60-second video script:**

> [0-5s] Chain graph appears. Three boxes: Maria → Concierge → Flight Booker → Payment Processor. DPoP proofs flash on each edge.
>
> [5-15s] Voiceover: "Your customers' agents are already booking flights, syncing calendars, charging Stripe. They're just not doing it safely."
>
> [15-30s] Terminal: `curl ... | sh` → `shark serve` → admin URL printed → first agent token in <60s.
>
> [30-45s] Code snippet: 10-line DPoP + token exchange. "RFC 9449 + RFC 8693 in 10 lines. Auth0 cannot do this."
>
> [45-55s] Cascade revoke: `client.users.revoke_agents("usr_maria")` → all 4 child agent tokens go red in audit log.
>
> [55-60s] One frame: "Auth for humans and the agents they ship. 30MB binary. Self-hosted free. sharkauth.com"

---

### Day -4 (Fri 05-02) — Buffer + Quote Collection

- Catch any DMs that bounced
- Collect public quotes from anyone who deployed
- Pre-write 5 tweets for launch day (drip schedule)
- Pre-write the HN post body (lead comment pinned by you)
- **Reach out to 3 YC W25/W26 founders beyond InsForge.** Use the Slack/Discord backchannels. Ask: "Launching Monday. Would a retweet help if you find it useful?"

---

### Day -3 (Sat 05-03) — Rehearse + Stress Test

- Full dry-run: deploy shark from scratch on a clean VM. Time it. Record any friction. Fix.
- Stress test concierge demo: run 50 times. Any flake = fix.
- Final landing page review: open in 3 browsers + mobile. Hero loads <1s.
- Practice the "what is shark" 30-second pitch out loud. Record yourself. Listen back.

---

### Day -2 (Sun 05-04) — Final Lock

- No code changes after 6pm. Freeze.
- Schedule HN post for Monday 9am ET (peak HN traffic window)
- Schedule LinkedIn + Twitter posts
- DM all interested adopters: "launching tomorrow, if you can post your quote any time Mon-Tue that'd help"
- Sleep 8h. Critical.

---

### Day 0 (Mon 05-05) — LAUNCH

**6am local:** Final smoke run. Verify shark.exe builds clean. Push final commit.

**9am ET (HN peak):**
- Submit Show HN
- Pin lead comment with concierge demo gif + 3 code snippets
- Tweet thread (10 tweets) with screencast as #1
- LinkedIn post (longer-form, founder voice, transport-hack origin story)
- DM every adopter: "launched, here's the HN link, would love your quote in the thread"

**Hour 1-3 (HN front page window):**
- Refresh comments every 5 min. Reply to EVERYTHING. Even hostile.
- Reply pattern: thank them, address the technical point, link to specific code/doc, end with question that invites continued engagement.
- Don't ask for stars. Ask for feedback.

**Hour 4-12:**
- Cross-post Reddit: r/golang, r/programming, r/selfhosted, r/MachineLearning
- Each subreddit gets a custom post (NOT same copy). Lead with what that audience cares about.
- Reach out to anyone who tweeted with "this looks cool" — convert to deployment.

---

### Day +1 (Tue 05-06) — Compound

- Tally: stars, named adopters, HN ranking, replies
- DM anyone who engaged but didn't deploy: offer 1:1 setup help
- Write Day +1 follow-up post with "what we learned, who's deploying, what's next"

---

### Day +2-3 (Wed-Thu 05-07-08) — Application Prep

Application skeleton below. Fill with launch numbers.

---

## YC W26 Application Draft (300 words target)

**Question 1: What is your company going to make? (300 chars)**

> SharkAuth is auth for products that ship agents to customers. Open source. Single 30MB Go binary. Real OAuth identity per agent, DPoP-bound tokens, RFC 8693 delegation chains, cascade revoke when a customer churns. Self-hosted free, cloud paid.

**Question 2: Why this? (1500 chars)**

> Every team shipping agents — Lovable, Cursor-style products, MCP servers — is hand-rolling DPoP + RFC 8693 + audit chains. Most ship it broken. Auth0 doesn't cover agent identity. WorkOS doesn't. The category is empty because OAuth experts haven't met agent builders.
>
> I built shark in [N] days. ~[100k] LOC. RFC-correct: DPoP (9449), token exchange (8693), DCR (7591), introspection (7662), revocation (7009), AS metadata (8414), resource indicators (8707), JWKS (7517). 60+ endpoints. Embedded SQLite. Embedded React admin. `shark serve` and you're running.
>
> Launch [date]. By submission: [N] GitHub stars, [N] HN points, named adopters: [@handle1 at project1], [@handle2 at project2], [@handle3 at project3]. [InsForge YC W26 founder] said publicly it's [quote].
>
> Window: 12-18 months before Auth0 retrofits. Wedge: drop-in agent auth for MCP servers.

**Question 3: Why you? (1500 chars)**

> 18yo, 2nd-semester CS at [school], Monterrey, Mexico. Solo. Built shark in [N] days while taking 5 classes.
>
> Origin: [transport hack story — concrete, what you cracked, what you found, why it taught you about identity systems].
>
> Why solo: tried to recruit a co-founder; pitch didn't land before I had artifact. Built the technical bet alone to prove it. Resuming co-founder search post-launch.
>
> Background: [add 2 specific shipped projects beyond shark, even small ones]. Pattern is consistent — pick a hard system, ship the working thing in weeks not months.

**Question 4: How will you make money?**

> OSS self-hosted free forever. Cloud-hosted SaaS for teams who don't want to run it ($X/MAU + $Y/agent). Enterprise: SSO into shark itself, audit export to Datadog/Splunk, SLA, on-prem support. Sentry/Plausible playbook — OSS for distribution, SaaS for revenue.

**Question 5: What's hard about this?**

> Auth space is a graveyard. Auth0 won the human-auth war. Every clone died. We're not cloning Auth0. We're building the agent-auth primitive Auth0 hasn't shipped — and bundling human auth as a freebie because the same data model serves both. The risk is Auth0 ships RFC 8693 + DPoP themselves. Our bet: 12-18 month window, OSS distribution moats them out before they react.

---

## Application Video (60s, recorded Day +2)

Same script as press-kit 60s video, with one frame change at end:

> [55-60s replaced]: Founder face on camera. "I'm Raul, 18, Monterrey, Mexico. I built shark in [N] days while taking 5 classes. [Quote from real adopter] is using it in prod. We have 12 months before Auth0 wakes up. I want to spend them at YC."

---

## Risk Register

| Risk | Probability | Mitigation |
|---|---|---|
| HN flop (front page miss) | Medium | Cross-post Reddit Day 0. Have 3 backup launch beats — Day +3 follow-up with metrics, Day +7 "first integrators" post, Day +14 "lessons" post. |
| Zero adopters by Day -3 | Medium-high | Lower the bar: any friend's project + 1 InsForge use case + you running shark on a personal MCP = 3 named adopters. Adopter doesn't have to be a stranger. |
| Critical bug in launch demo | Medium | Day -3 stress test 50x. Prefer broken-but-honest "known issue, fix shipping" over hidden landmine. |
| InsForge founder ghosts | Medium | Other YC founders accessible via Slack backchannels. Have 3 backups. |
| YC says "come back when you have revenue" | High | Acceptable outcome. Apply Fall (W27) batch with revenue. Launch week traction is not wasted — it's the foundation. |

---

## What NOT to do this week

- Don't add features. Freeze scope per `08-launch-scope-cuts.md`.
- Don't fix the proxy bugs. They're W+1.
- Don't write blog posts. Screencast > blog post 100x for launch traction.
- Don't apply to YC before launch numbers land. Empty application = wasted shot.
- Don't argue with HN trolls. Reply to substantive comments only.
- Don't compare to Auth0 in marketing copy directly. Show, don't tell.

---

## Daily Standup Format (with yourself)

End of each day, write 3 lines in `playbook/launch-log.md`:

```
## Day -N (date)
- Shipped: [concrete thing]
- Blocked: [thing or "nothing"]
- Adopter count: [N named, M waitlist]
```

This becomes application ammo. The log itself is a YC signal — execution velocity visible.

---

## Founder-State Hygiene

- Sleep 7+ hours every night this week. The launch is decided by Day 0 execution, not by exhaustion.
- One social-media check window per day, max 30 min. Not all day.
- Have one person you can text who is NOT a founder. The week is psychologically heavy. Don't isolate.
- The launch is not the YC interview. Even a "soft" launch (300 stars, 1 named adopter) is a strong YC application if the narrative is tight.

---

## Done Definition

By 2026-05-08 (application submission day):
- [ ] HN top 30 minimum (top 10 = strong)
- [ ] ≥3 named adopters with public quotes
- [ ] InsForge founder publicly engaged (post / tweet / LinkedIn)
- [ ] ≥500 GitHub stars (vanity but applicable signal)
- [ ] Application video recorded + reviewed by 1 trusted reader
- [ ] 300-word application written + reviewed by 1 trusted reader
- [ ] Launch log filled for all 8 days

If 5/7 boxes checked → submit immediately.
If <5/7 → delay one batch, build for 8 more weeks, apply Fall with revenue.

---

*This is the operational sibling to `07-yc-application-strategy.md`. Strategy = why. This = how. Update daily.*
