# Wave 4 — Launch Artifact + Distribution

**Budget:** 2h CC + your hands · **Outcome:** Monday launch shipped, named integrators emerging

## Sequence (Sunday night)

1. Pull main, run full smoke (`./smoke.sh`) — verify GREEN before recording
2. Record 30-second screencast (script below)
3. Capture 3 dashboard screenshots
4. Update README + open PR if not already merged
5. Schedule HN submission for Monday 9:00 ET (window: 8-10am ET for max front-page chance)
6. Pre-write all post drafts (HN, Twitter, Discord, LinkedIn DM)
7. Sleep ~6 hours. Wake up Monday with a calm head.
8. Monday 9:00 ET: submit HN, Twitter thread, Discord posts, DM to Hang Huang.

## Screencast script (30 seconds)

Record on a clean install. No personal handles or paths visible.

```
0:00-0:03   Terminal: `shark serve` running, dashboard URL printed
0:03-0:10   Browser: dashboard home → "Agent Security" attention card visible
0:10-0:18   Click an agent → detail drawer opens → "Security" tab → DPoP keypair + jkt visible
0:18-0:25   Switch tabs to "Delegation Policies" → may_act rules visible
0:25-0:30   Terminal: `shark demo delegation-with-trace` runs, browser opens to report
```

Tools: OBS Studio (mac/linux/windows free), or QuickTime Player (mac), or built-in screen recorder. Output: 1080p, 30fps, mp4 ≤ 30MB for Twitter inline embed.

Save to `playbook/launch-screencast.mp4`. Also upload to YouTube as unlisted (for HN embed) and to a CDN (for Twitter inline).

## Screenshots to capture

1. **Agent detail · Security tab** — DPoP keypair, jkt thumbprint, rotation history visible
2. **Audit page** — delegation breadcrumb visible: `[user] → [agent-A] → [agent-B]`
3. **Demo report HTML** — full page screenshot of the chain graph + token cards + audit table

Save to `playbook/screenshots/`. Use these in posts.

## HN post

### Title — pick one (test mentally; commit Sunday night)

| Option | Title | Strength |
|---|---|---|
| A (safe) | Show HN: Agent-native OAuth in a 30MB binary (DPoP, delegation chains, MCP) | Concrete artifact + numbers. Hardest to torpedo. |
| **B (recommended)** | Show HN: I built RFC-correct agent auth solo in a month – DPoP, delegation, MCP | Founder-story angle. HN rewards craft + speed. |
| C (high-risk/high-reward) | Show HN: SharkAuth – auth for AI agents that Auth0 can't ship without rewriting their token model | Strongest claim. Polarizing — top votes OR top "overclaim" downvotes. |

**Pick B.** Lead with "solo in a month." Let the demo prove the moat. Don't make the moat claim in the title — make it in the post body where you can substantiate.

### Title (≤ 80 chars, recommended)
> Show HN: I built RFC-correct agent auth solo in a month – DPoP, delegation, MCP

**Body:**

```
Hi HN — I'm Raúl, 18, solo, building from Monterrey, Mexico.

I shipped SharkAuth: auth for products that give customers their own agents.
Single binary, ~30MB, sqlite-backed, runs in 30 seconds from `shark serve`.

The customer category: Replit Agents, OpenAI Custom GPTs, Cursor, Lovable,
Bolt, v0, Zapier AI, Decagon, Vapi — products where each customer has agents
that act on their behalf, holding their credentials, calling their resources.
Today these teams hand-roll: per-tenant scoping, agent-token rotation, vault
for the customer's third-party credentials, revocation when something goes
wrong. That stack is usually missing layers.

What ships:
- Five-layer revocation depth-of-defense:
  1. Per-token revoke (RFC 7009 with family cascade) — for leaked tokens
  2. Per-agent revoke-all-tokens — for compromised single agent
  3. Per-customer cascade — when a customer goes rogue, every agent they
     spawned dies in one transaction
  4. Per-agent-type bulk by pattern — when v3.2 of an agent template ships
     a bug, kill all instances without touching customers (W+1)
  5. Per-vault-credential cascade — when a customer's external OAuth is
     compromised, agents holding scope on it auto-revoke (W+1)
- DPoP RFC 9449 — every token cryptographically bound to the agent's keypair.
  Token theft alone is useless without the private key.
- RFC 8693 token exchange with `act` claim chain — delegated sub-tokens audit
  the full actor lineage from human login to third-hop agent.
- Encrypted vault for OAuth provider tokens (Gmail, Slack, GitHub, Notion)
  with per-agent retrieval bound to the delegation chain.
- Full human auth: SSO, magic-link, organizations, RBAC. Because the trust
  chain starts with a human, splitting it across two servers breaks the audit.

Honest scoping: layers 1-3 ship today. Layers 4-5 ship within 2 weeks of
launch. CLI/SDK story is sharp; dashboard onboarding polish lands W18.
Proxy + Auth Flow Builder land in W18+ after regression fixes (hidden in
launch build). TypeScript SDK lands W18-W19. Hosted/cloud tier later.

Boundary: SharkAuth doesn't replace agent-orchestration internals (LangChain,
CrewAI, Claude Code's subagent dispatch). It handles agent-to-external-resource
auth, where the resource is owned by a specific customer of YOUR platform.

What ships:
- DPoP RFC 9449 — every token cryptographically bound to the agent's keypair.
  Token theft alone is useless without the private key.
- RFC 8693 token exchange with `act` claim chain — delegated sub-tokens audit
  the full actor lineage, not just the final caller.
- Encrypted vault for OAuth provider tokens (Gmail, Slack, GitHub, ...) with
  per-agent retrieval bound to the delegation chain.
- Cascade revoke — `POST /api/v1/users/{id}/revoke-agents` kills every agent
  that user spawned plus all their tokens in one transaction.
- Full human auth: SSO, magic-link, organizations, RBAC. Because the trust
  chain starts with a human, splitting it across two servers breaks the audit.

Demo (30s): <screencast URL>
Live trace: run `shark demo delegation-with-trace` — produces an HTML report
showing 3 agents in a delegation chain with cryptographic proofs visible.

Repo: <repo URL>
Docs: <docs URL>

This is launch day. Looking for: MCP server builders or agent framework folks
willing to be the first to integrate. If that's you, please open an issue or
DM. I'll respond same day.

Honest scoping: CLI/SDK story is sharp; dashboard onboarding polish lands W18.

Happy to take pointed questions.
```

## Twitter thread (5 tweets)

**Tweet 1 (with screencast inline):**
> I shipped SharkAuth: open-source agent-native auth.
>
> 30MB single binary. MCP-native OAuth 2.1, DPoP-bound tokens, delegation chains. The auth layer Auth0 can't ship without rewriting their token model.
>
> 30s demo ↓

**Tweet 2 (with Security-tab screenshot):**
> Tokens are cryptographically bound to the agent's keypair (RFC 9449 DPoP).
> Theft alone is useless — the attacker also needs the private key.
>
> Visible in the dashboard, not buried in JWT claims.

**Tweet 3 (with delegation breadcrumb screenshot):**
> Agent delegation = chain of `act` claims (RFC 8693).
>
> SharkAuth shows the full actor chain inline:
> [user] → [triage-agent] → [knowledge-agent]
>
> Each hop has its own DPoP binding. Each hop is audited.

**Tweet 4 (with demo HTML screenshot):**
> One command — `shark demo delegation-with-trace` — spins up 3 agents, runs a delegation chain, generates a self-contained HTML report with the cryptographic proofs visible.
>
> 60 seconds from binary to "wait, this is actually new."

**Tweet 5 (call to action):**
> 18, solo, Monterrey. Built this in <X> weeks.
>
> Looking for the first MCP server builder or agent-framework maintainer to integrate. DM open.
>
> HN: <link>
> Repo: <link>

## MCP Discord post

**Channel:** #showcase or #server-builders (check pinned)

```
Hi all — shipped SharkAuth today, an OAuth 2.1 server built around agents.

For MCP server builders specifically:
- MCP-native authorization spec implemented
- DPoP-bound tokens RFC 9449
- Delegation chains via RFC 8693 token exchange
- Single-binary, 30MB, sqlite-backed, runs locally in 30s

If anyone here wants to be the first MCP server with agent-native auth wired in,
I'll personally help with the integration. DM me here or open an issue.

Demo: <screencast URL>
Repo: <repo URL>
```

## Cloudflare Agents Discord post

Same template, swap "MCP server builders" for "Agents builders." Mention specifically: "if you're building agent infrastructure on Workers and want a self-hostable auth backend, here it is."

## Reddit posts — Monday afternoon (NOT before HN)

**Order rule:** post Reddit AFTER HN front page is achieved. Reddit before HN poisons HN engagement ("seen on r/X yesterday" comments). Monday afternoon (12-2pm ET) is the window — HN ranking is set, Reddit compounds momentum.

**Pick 3 subs maximum. Skip the rest.**

### r/golang

**Title:** `Built RFC-correct agent OAuth in Go — DPoP, delegation chains, single 30MB binary`

**Body angle:** technical implementation deep-dive. Gophers respect RFC adherence and crypto correctness. Lead with the Go-specific decisions.

```
Spent ~1 month solo building SharkAuth — agent-native OAuth server in Go.

Implementation choices that matter for this sub:
- Single binary via `embed` for SQLite migrations + admin UI assets
- DPoP RFC 9449 implementation in pure stdlib crypto (ECDSA P-256, EdDSA, RS/PS)
- Token exchange RFC 8693 with `act` claim chain — uses `jwt-go` minimally,
  most validation is custom because of agent-specific scope downscoping
- Audit log uses sqlite WAL + JSONL replication (no Postgres dependency for self-host)
- ~30MB final binary, cross-compiles to darwin/linux/windows arm64+amd64

Repo: <link>  ·  Demo: <link>

Critique welcome on the `internal/oauth/dpop.go` and `internal/oauth/exchange.go`
specifically — those are the parts where I'd most value a pair of senior eyes.
```

### r/selfhosted

**Title:** `SharkAuth — open-source single-binary auth server (humans + AI agents, full SSO/RBAC/MCP)`

**Body angle:** screenshot-heavy. r/selfhosted runs Docker and evaluates auth tools. Lead with deploy ergonomics.

```
Released today. Single-binary, sqlite-backed, no external deps. `shark serve` and
you have an auth server in 30 seconds.

What it covers (without running 5 services):
- Human auth: SSO (OIDC/SAML), magic-link, password+MFA, organizations, RBAC
- Agent auth: MCP-native OAuth 2.1, DPoP-bound tokens, delegation chains
- Audit log + admin dashboard included

Why one binary: I got tired of running Auth0 plus a delegation library plus an
audit pipeline. SharkAuth is the consolidation.

Self-hosted free, fully open source. Cloud tier coming later.

Screenshots: [dashboard], [agent detail Security tab], [delegation breadcrumb]
Demo: <link>  ·  Repo: <link>  ·  Docker: <docker compose snippet>
```

Include 3-4 screenshots inline. r/selfhosted scrolls fast — visuals win.

### r/LocalLLaMA

**Title:** `Open-source auth for local agents — DPoP-bound tokens, delegation chains (MCP-native)`

**Body angle:** for people running local models / building agents. Frame as the missing piece.

```
If you're building agents that call out to APIs (or your local agent calls another
agent), you've probably hand-rolled some auth. Here's a single binary that handles it.

- DPoP-bound tokens — token theft alone is useless without the agent's private key
- Delegation chains — agent A delegates to agent B, every hop signed and audited
- MCP-native OAuth flows so MCP servers Just Work

Self-hosted, sqlite, runs in 30 seconds. Open source.

Demo (30s): <link>
Repo: <link>

Curious if anyone here is wiring auth for their local agent stacks — would love
to learn what people are using today and where the pain is.
```

The last paragraph is intentional — invites people to comment with their pain, which seeds engagement.

### Subs to AVOID (will downvote or remove)

- **r/programming** — strict mods, first-time-poster self-promo often removed. Risk > reward.
- **r/startups, r/Entrepreneur** — anti-self-promo, near-guaranteed removal.
- **r/SideProject** — low-quality engagement. Drives stars but rarely integrators.
- **r/MachineLearning** — research-focused, will downvote anything commercial-looking.
- **r/aws, r/devops** — wrong audience.

### Reddit posting hygiene

- Each sub has rules in the sidebar — read them before posting. Mods enforce. Removed posts hurt.
- If you've never commented in a sub before, post is more likely to be flagged. Mitigation: Sunday afternoon, comment substantively in 2-3 recent posts in each sub. Build minimal account history.
- Don't cross-post the same body to all 3. Each sub gets its own adapted copy (above).
- Reply to every comment in the first 2 hours, same as HN.
- Don't lead with "Show HN:" prefix on Reddit — looks like cross-posting and gets downvoted.

### Anti-pattern: posting to 10 subs

Mod teams talk. Aggressive cross-posting can ban your account across multiple subs at once. Three well-targeted posts > ten carpet-bombs.

## LinkedIn DM to Hang Huang (InsForge CEO)

```
Hey Hang — launching today as promised. Repo is now public:

<repo URL>
Demo (30s): <screencast URL>

Quick recap of why I think this is interesting for the InsForge community:
agents become active internet users, DPoP + delegation are the right primitives,
incumbents are 12-18 months from retrofitting it.

If you do engage with the launch as you offered, the assets that work best:
- 30s screencast (Twitter inline)
- HN thread: <HN URL>
- Repo stars

Also — your offer about InsForge backend stands. If we ever spin up a hosted
SharkAuth, InsForge is on the shortlist for the data layer. Let me know what
makes sense.

Thanks again for the YC application links — using them this week.

— Raúl
```

## Outreach to candidate-user list (from Wave 0 / 05-assignment.md)

Each of the 10 names from `.planning/launch-targets.md` gets a personalized DM Sunday night, before the public launch. Template:

```
Hey <name> —

Saw <specific thing they shipped/posted>. I'm launching SharkAuth tomorrow,
open-source auth for agents (MCP-native OAuth, DPoP, delegation chains).

Specifically thought of you because <one-line reason rooted in their work>.

If <their project> needed agent auth in the next 30 days, would this be useful?
Happy to help wire it up — would mean a lot to have you as the first integrator
in the launch story.

Demo (30s): <link>
Repo: <link>

No pressure. If now's not the moment, totally get it.

— Raúl
```

Send these BEFORE the public posts. If even one converts, the launch story changes from "I built X" to "Sarah at Acme is using SharkAuth — here's why."

## Definition of done for Wave 4

- Screencast recorded and uploaded
- Screenshots saved to `playbook/screenshots/`
- README updated with screencast link + screenshots
- HN draft saved (don't submit yet)
- Twitter thread drafts saved
- Discord posts drafts saved
- Hang Huang DM saved
- 10 personalized outreach DMs sent Sunday night
- Monday 9:00 ET: HN submitted, Twitter thread posted, Discord posts dropped, Hang Huang DM sent

## HN day tactics — first 3 hours determine ranking

**Pre-launch checklist (Sunday night):**

- Test `shark serve` on 3 fresh machines (or VMs / Docker containers). HN top comment "broken on Apple Silicon" kills the post. Test darwin/arm64, darwin/x64, linux/x64 minimum.
- Make sure the demo screencast is uploaded and the URL works (test in a private/incognito window).
- Pre-write the first comment you'll post on your own thread.
- Schedule HN submission for Monday 9:00 ET (window: 8-10am ET maximizes US-East-Coast morning rush).

**T+0 to T+5 minutes after submitting:**

- Post your own first comment on the thread, top-level. Three short paragraphs:
  1. "Author here. 18yo solo founder from Monterrey, Mexico. Built this in ~1 month with Claude Code."
  2. "Why this exists: <one sentence on the agent-auth gap>. AMA about the implementation, RFCs, or the launch process."
  3. Demo link + repo link.
- DM 5-8 friends/community-members the HN URL. Ask them to upvote IF they actually find the post interesting. Do not coordinate vote-bombing — HN catches it.

**T+5 to T+30 minutes:**

- Watch the post. Reply to every comment within 5-10 minutes.
- If a top comment is "this is just Auth0 with a different theme," respond with a technical breakdown: which RFCs, which specific token-model differences, link to the exact file:line in the repo.
- If a top comment asks for screenshots, paste the dashboard Security-tab screenshot inline.
- If someone reports a bug, thank them, link to a GitHub issue you create on the spot. Visible bug-handling builds trust.

**T+30 to T+2 hours:**

- Stay on the thread. Front-page rank is decided in this window.
- If you hit front page (top 30), post the thread URL on Twitter, Discord, LinkedIn — drives more eyes back to HN, which feeds the ranking algorithm.
- Don't sleep. Don't take a meeting. The thread is the launch.

**T+2 to T+6 hours:**

- Reply rate can drop to once per 30 min. But still reply to every substantive comment.
- Capture screenshots of HN ranking every 30 min — these go into `playbook/launch-day-log.md` and the YC application.

**Anti-patterns (avoid):**

- Asking friends to upvote without reading. HN penalizes coordinated voting and your account/post can get flagged.
- Engaging hostile comments emotionally. Calibrated technical response wins. Snark loses.
- Disappearing for hours during the first 2 hours. The post dies without you.
- Editing the title or body after submission. HN penalizes edits in some cases.

## Tracking after launch

Create `playbook/launch-day-log.md` Monday morning. Track:
- HN rank trajectory (every 30 min, save screenshot)
- Star count delta on the repo
- DMs received + handle of each
- Which of the 10 outreach contacts responded
- First named integrator (if any) — capture handle, project, the line they posted

This file becomes evidence for the YC application and MX funding application.
