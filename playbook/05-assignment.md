# Wave 0 — The Assignment (DO THIS FIRST)

**Budget:** 90 minutes · **Output:** `.planning/launch-targets.md` with 10 named candidate integrators · **Run before any code**

## CRITICAL UPDATE — target shift post-pitch-refinement

The pitch sharpened to "auth for products that give customers their own agents." This means the target list shifts:

- **Higher priority now:** platform-team contacts at products in this category — Replit eng/security, OpenAI Custom GPTs team, Cursor security/auth lead, Lovable founder/eng, Bolt eng, Vercel v0 team, Zapier AI eng, Decagon eng, Bland eng, Vapi eng, Intercom Fin team, CrewAI Cloud eng. **These are customers, not integrators.** Different DM template (below).
- **Lower priority now:** individual MCP server builders. They're real but they're builders, not buyers. They drive adoption signal for HN/launch but don't pay rent.

Target the list with at least 5 platform-team contacts and 5 MCP/agent-builder contacts. Mix.

## Why this is Wave 0

The demo is engineering. The list is the company.

Without 10 specific named people to send the demo to Sunday night, Monday's launch lands in a void. Stars without integration is fool's gold. With this list, you DM 10 humans Sunday night with a personalized message + the demo. Even one converting changes the launch story from "I built X" to "X is integrated by Y."

That single conversion is the headline of your YC application and your MX funding writeup. It is worth more than another shipped feature.

## Constraint

90 minutes. Don't go past it. The list does not need to be perfect — it needs to exist.

## Where to look (in order, time-boxed) — REVISED

### 0. Platform-team contacts (30 min) — NEW HIGHEST PRIORITY

For each product in the customer category, find the security/auth/platform engineering lead:

- **Replit** — search LinkedIn "Replit security" / "Replit platform engineering"; check Twitter for handles. Recent Replit Agents launch posts mention specific team leads.
- **OpenAI Custom GPTs / Actions** — harder to penetrate but try: people who tweet about Custom GPTs Actions OAuth specifically. The OAuth-implementation team is small.
- **Cursor** — eng team is small and public. Founders + early eng are on Twitter. DM the eng founder directly with one line on rogue-customer attribution.
- **Lovable, Bolt, v0** — small teams, founders take DMs. Vercel v0's eng leads are known on Twitter.
- **Zapier** — has a Zapier AI / Central team. Zapier eng blog identifies leads.
- **Decagon, Crescendo, Sierra, Maven AGI** — customer-support-AI products. Each has eng leads on Twitter who post about agent reliability.
- **Bland, Vapi, Retell** — voice-AI products. Similar pattern.
- **Intercom Fin** — Intercom AI team.
- **CrewAI Cloud, LangChain Cloud, LlamaIndex hosted** — when used as hosted multi-tenant products, they're customers too.

**DM template for platform-team contacts (different from builder template):**

```
Hey <name> — saw you work on <product>. Quick question:

When a customer of <product> uses their AI agent to abuse the system (rogue
prompts, exfiltrating data, etc.), how do you currently trace the agent action
back to the specific customer who authorized it, and revoke all their agents
in one transaction?

I just shipped SharkAuth, an open-source auth server built around exactly this
problem. 30s demo: <link>. Would love your read on whether this maps to a real
pain at <product>, or if you've already solved it differently.

Repo: <link>
No pressure — happy with a one-line "we use X for that" reply too.

— Raúl
```

This DM does three things: names a specific concrete fear (rogue customer attribution + cascade revoke), frames as a question not a pitch, and asks them to validate or refute. Either response is intel.

### 1. Anthropic MCP Discord (15 min)

- Join: https://www.anthropic.com/discord (or the public MCP Discord invite link from anthropic.com)
- Channels to mine: `#showcase`, `#server-builders`, `#general`, anywhere people post their MCP servers
- For each MCP server you find, check: who built it? Are they active in the last 30 days?
- Look for: server builders shipping in production, not toy demos. Bonus: anyone openly complaining about auth.

**Capture per person:**
- Discord handle
- Project name
- Last meaningful post (link)
- Their pain (your guess, in one line)

### 2. Cloudflare Agents Discord (10 min)

- Cloudflare Agents framework is hot, growing fast. Builders are ambitious.
- Channels: their Discord's #builders / #showcase / #help
- Look for: people building agent platforms on Workers. They will hit auth pain soon.

### 3. Smithery (10 min)

- https://smithery.ai/ — MCP server registry
- Browse the most-installed servers + most-recent. Click through to repos.
- Maintainers who have shipped + maintained MCP servers = the people most likely to feel auth pain.

### 4. Vercel AI SDK contributors (10 min)

- https://github.com/vercel/ai → contributors tab
- Recent commits. People shipping AI infra at Vercel scale.
- Focus on contributors with their own side projects related to agents.

### 5. LangChain / LlamaIndex maintainers + power users (10 min)

- Recent contributors who ship agent applications, not framework features
- People who blog about agent auth pain (search "OAuth agents site:medium.com" or similar)

### 6. Twitter / X (10 min)

- Search: "MCP server" auth, "agent auth" oauth, "DPoP", "RFC 8693"
- People who tweet specifically about agent auth pain in the last 60 days.
- Indie hackers building agent products.

### 7. HN comments (10 min)

- Search HN: `agent oauth`, `agent authentication`, `MCP authorization`
- Look at last 30 days. Read comments. People debating auth approaches = people who will read your launch.

### 8. GitHub search (10 min)

- `org:cloudflare agents auth` — Cloudflare Agents team
- Search code for `// TODO: real auth` or `// HACK: hardcoded token` in agent repos. Those repos have a real problem.

## The list format

Save to `.planning/launch-targets.md`:

```markdown
# Launch Targets

Generated: 2026-04-26
Goal: 10 named MCP/agent builders for personalized outreach Sunday night.

## #1 — @handle (project)

- Where: <Discord/Twitter/GitHub link>
- Last activity: <date> · <one-line description>
- Pain (guess): <one line>
- Why them specifically: <one line>
- DM template variant:
  Hey <name> — saw <thing>. I'm launching SharkAuth tomorrow...
  
  <see playbook/04-wave4-launch.md outreach template — fill in their specifics>

## #2 — @handle (project)

[same format]

...

## #10 — @handle (project)

[same format]

## Backlog (extras you found, save for launch+1)

- @handle — <one line>
- @handle — <one line>
```

## Quality bar

A name on this list must answer YES to all four:

1. **Real human, not a category.** "Smithery contributors" is wrong. "@<specific handle> who has shipped 2 MCP servers in the last 30 days" is right.
2. **Currently active.** Posted/committed something in the last 30 days. Dead handles waste DMs.
3. **Plausibly feels the pain.** They're building agents that touch external APIs, OR they've explicitly mentioned auth pain.
4. **You can write a single specific line about their work.** Not "you build cool stuff" — "I saw your post about DPoP integration in <project>" or "your <X> server has the prefetch caching pattern I've been meaning to copy."

If you cannot write the specific line, they don't make the list. Move on.

## Anti-patterns (DO NOT do these)

- Don't pad with friends or classmates who won't actually integrate. The list is for cold/warm outreach.
- Don't pick "VPs of engineering at <big co>" — they don't integrate side projects on launch day.
- Don't pick anonymous handles with no public work — there's no specific line to write.
- Don't pick people you've never heard of based on followers alone. Activity > follower count.

## What you do with the list

Sunday night, after Wave 3 is done:

1. Use the DM template from `playbook/04-wave4-launch.md`
2. Personalize for each of the 10 — fill in `<thing they shipped>`, `<one-line reason>`
3. Send all 10 between 8pm-11pm local Sunday
4. Track responses in `playbook/launch-day-log.md`
5. If any responds with interest, drop everything Monday morning to help them integrate. That is the launch.

## If you finish in <90 minutes

Don't keep going. Move to Wave 1. The marginal 11th name is worth less than 30 more minutes of UI work.

## If you can't find 10

Lower the bar to 7 named + 3 communities to post in (e.g., MCP Discord general channel + Cloudflare Agents Discord + r/LocalLLaMA). The list is not about hitting 10 — it's about having a deliberate distribution plan for Monday morning.
