# YC Application Strategy — Honest Chances + Tactics

This file answers the question you asked at the end of the office hours session: what are my chances if my demo is strong + I lead with the obsession narrative + the public transport hack story?

The answer below is honest, not flattering. Treat it as a planning document, not a verdict.

## Honest base rates

YC accepts roughly 1.5-2% of applications overall. The distribution is not uniform. The relevant variables for your application:

| Variable | Your value | Effect on rate |
|---|---|---|
| Solo founder | Yes | -30 to -50% relative (YC prefers teams of 2-3, but funds solo regularly) |
| First-time founder | Yes | Slight negative — but neutralized for technical builders with shipped artifacts |
| Pre-launch | Yes | Significant negative — YC asks "how many users?" and zero is the worst answer |
| Strong technical artifact | Yes | Strong positive — RFC-correct DPoP + RFC 8693 + agent audit is uncommon |
| Hot space (AI agents) | Yes | Mixed — relevance is high, but saturated category means competition is fierce |
| Origin story (transport hack) | Yes | Strong positive — YC's archetype |
| Compressed shipping (100k+ LOC in a month solo) | Yes | Strong positive — "Relentlessly resourceful" archetype |
| Geographic | Monterrey, Mexico | Slight positive — YC actively recruits LATAM talent |
| Age (18) | Yes | Mixed — YC funds young founders regularly, but very-early-undergrad founders are rare |

## Realistic chance ranges

These are best-guess ranges. Your individual application could be anywhere in or beyond these bands.

| Scenario | Estimated interview rate |
|---|---|
| Pre-launch, solo, zero users, application weakly written | 1-2% |
| Pre-launch, solo, zero users, strong demo + strong narrative + transport hack story leads | 3-7% |
| Same as above + 1 named integrator publicly using SharkAuth by submission | 5-12% |
| Same as above + 5+ users posting publicly | 10-20% |
| Same as above + early revenue or LOI from a known org | 15-25% |

**Read this carefully:** the highest-leverage move between now and submission is **acquiring one named integrator**, not polishing the application. Every percentage-point gain after "strong narrative" comes from traction.

## What helps your application most

In rough order of leverage:

1. **One named user by submission.** Even a friend's MCP server. Even a personal project of someone in this list. It changes the answer to "how many users?" from "zero" to "@<handle> at <project>." That is a categorical jump.
2. **Demo video lead.** YC reviewers watch the application video. Lead with the 30-second screencast in the first 5 seconds. They decide whether to keep watching in the first 10. Make those 10 seconds the chain-graph + DPoP proof reveal, not your face.
3. **Transport hack as origin.** This is your YC archetype story. It signals: unauthorized curiosity + technical depth + obsession over a system. Tell it concretely (what you cracked, what you found) without bragging.
4. **Compressed shipping evidence.** "100k+ LOC, solo, Claude Code, ~30 days, RFC-correct" is a sentence. Use it. Show commit history if asked.
5. **Specific market thesis.** "12-18 month window before Auth0 retrofits agent semantics" — tighter than "AI agents are big."

## What hurts (and how to neutralize)

| Hurt | Neutralization |
|---|---|
| Solo founder | Frame: "tried to recruit, pitch didn't land, built the technical bet alone to prove it. Will resume cofounder search post-launch." Honest + agency. |
| Zero users | Acquire one named integrator before submission (see Wave 0). Application can then say: "first integrator @<handle> at <project> live as of <date>." |
| Pre-launch | Launch IS this Monday. By the time YC reads, you'll have launch traction + GitHub stars + at minimum HN data. Frame application around the launch, not pre-launch. |
| First-time founder | Lean on transport hack + 100k LOC compressed shipping. Both demonstrate "this person finishes things." |
| Age 18 / second-semester CS | Lean in. Don't apologize. YC has funded younger. The transport hack at age <X> is a story; suppress academic framing. |

## Application video — TWO formats, sequenced

Two videos compete for attention. They serve different audiences and ship at different times.

| Format | Audience | Ships when | Effort |
|---|---|---|---|
| **A: Raw 30s demo screencast** | HN, Reddit, MCP Discord, README top | Monday launch (mandatory) | 30 min |
| **B: Polished 60-90s startup-launch video** | YC application, LinkedIn, Twitter pinned, landing page | Wednesday-Thursday post-launch | 5-8h |

**Sequence:** Sunday: ship product + record A. Monday: launch with A. Tue-Wed: convert DMs. Wed night: record B with launch data in hand. Thursday: post B everywhere + embed in YC application.

**Why Wednesday-Thursday for B is the better timing:**
- Real launch data to reference ("X stars, Y integrators, Z DMs converted")
- You'll be calmer than Sunday (Sunday adrenaline = visible on camera, looks anxious)
- B becomes a "1-week retrospective" video — separate news cycle from Monday's launch
- YC application closes Saturday-ish — Thursday recording leaves 48-72h to polish

### Video A — 30s raw demo (Monday launch)

```
0:00-0:05   Terminal: `shark serve` running, dashboard URL printed
0:05-0:12   Browser: dashboard home → "Agent Security" attention card visible
0:12-0:20   Click an agent → detail drawer opens → "Security" tab → DPoP keypair + jkt visible
0:20-0:25   Switch tabs to "Delegation Policies" → may_act rules visible
0:25-0:30   Terminal: `shark demo delegation-with-trace` runs, browser opens to report
```

No voiceover. No music. Pure visual. HN engineers prefer raw artifacts.

### Video B — 90s polished startup-launch video (Wed-Thu, used for YC application)

```
0:00-0:10  YOU on camera, plain background, no title card.
           "Three years ago I cracked the cryptography of my city's transit
            system out of curiosity. That's how I learned auth. Last month,
            agents started becoming real users on the internet. So I built
            the auth they need."

0:10-0:25  CUT TO: shark serve in terminal → first-boot magic-link login → dashboard home
           Voiceover: "SharkAuth. Single binary, 30 seconds to running. Full
            human auth — magic-link, SSO, organizations. Plus the thing
            Auth0 cannot ship: agent identities."

0:25-0:50  CUT TO: register agent → delegation chain demo → vault retrieval (third hop fetches Gmail)
           Voiceover: "Every agent inherits its human's privileges. Three
            agents in a delegation chain. Each token cryptographically bound
            to the agent's keypair. The third-hop agent fetches an encrypted
            Gmail credential from the vault. Watch the audit log update."

0:50-1:10  CUT TO: cascade revoke demo — admin runs revoke-agents on the user
           Voiceover: "When the human is revoked, every agent they spawned
            dies in the same transaction. Rogue insider attribution in one
            query. This is the architectural bet."

1:10-1:30  BACK TO YOU on camera
           "I'm Raúl. 18, solo, Monterrey, Mexico. 100,000 lines of RFC-correct
            code in a month with Claude Code. <N> integrators in launch week.
            <if you have a quote from a real user, drop it here>.
            SharkAuth: <repo URL>. The agent era needs its own auth."
```

**Production notes:**
- Voice steady. Confidence > enthusiasm. Don't smile excessively.
- 5 takes minimum on the on-camera bookends. Pick the one where your voice doesn't shake on the transport-system line.
- No background music for first cut. Add music ONLY if it doesn't distract from voice.
- 1080p minimum, 4K if your phone supports. Vertical for Twitter/LinkedIn auto-crop, horizontal master for YouTube embed.
- Plain background — wall, bookshelf, or neutral. Not your bedroom. Not Mexican-flag context unless it's deliberate framing.
- Lighting: natural daylight from window in front of you, NOT behind you.

**Where Video B goes after Thursday:**
- YC application video field (mandatory, this IS the video)
- Twitter pinned tweet
- LinkedIn launch post (with caption: "1 week into SharkAuth — here's what shipped and what's next")
- Landing page hero section (when you have one)
- Founder DM attachments to high-value targets

### Will Video B raise YC chances?

Yes, marginally. Polished video signals execution capability and taste — both YC factors. Same-direction, same-magnitude bump as +1 named integrator. Don't trade Video B for the named user. Do both. Sequence: Wave 0 → Waves 1, 1.5, 2, 3 → Wave 4 launch with A → name-user conversion → Video B Thursday → YC submit Saturday.

Notes:
- Voice is steady, not breathy or excited. Confidence > enthusiasm.
- Speak in declaratives: "I built X." "Auth0 is 18 months away." "Launched Monday."
- Don't say "I think." Don't say "I hope." Don't apologize for being solo.
- Record at least 5 takes. Pick the one where your voice doesn't shake on the transport-hack line.

## Application long-form — section-by-section

YC's questions vary slightly per batch but core fields are stable.

### "What does your company do? (~55 words) — LOCKED VERSION (depth-of-defense framing)"

```
SharkAuth is auth for products that give customers their own agents. Every
token traces to the customer who authorized the agent that issued it. When
something goes wrong, you have precise responses at different blast radii:
revoke a leaked token, kill one agent, cascade-revoke a customer's whole
fleet, kill all instances of a buggy agent-type, or disconnect a compromised
vault credential. Five layers, one mental model. Open source. Self-hosted free.
```

**Why this version (third refinement, post code-audit of revocation primitives):**
- Earlier framings ("agent-native OAuth" / "agent security platform" / "cascade revoke") were each too narrow. The real moat is FIRST-CLASS LINEAGE between customer and agent — which enables multiple precise responses at different blast radii, not one blunt cascade.
- "Depth of defense" is enterprise-security industry language. Buyers respond to it.
- Five layers named explicitly: per-token, per-agent, per-customer cascade, per-agent-type pattern, per-vault-credential. Concrete and audit-mappable.
- Honest about what ships when (3 layers Monday, 2 layers ship W+1 to W+2 in Wave 1.5+ + Wave 1.6).

### Five-layer security response model (canonical)

| Layer | Threat | Response | Status |
|---|---|---|---|
| Token | Customer's agent token leaks via prompt injection | RFC 7009 revoke + family cascade | ✅ ships |
| Agent | One specific agent compromised across sessions | `POST /agents/{id}/tokens/revoke-all` | ✅ ships |
| Customer fleet | Rogue customer abuses platform via their agents | `POST /users/{id}/revoke-agents` cascade | Wave 1.5 (~6h) |
| Agent-type pattern | Buggy agent-type v3.2 across all customers | Bulk-revoke by `client_id` pattern | Wave 1.6 (~3h) |
| Vault credential | Customer's external OAuth compromised | Vault-disconnect cascades to agents | Wave 1.6 (~4h) |

### Customer answer for YC interview (updated)

> "Replit's eng team shipping Replit Agents to customers. OpenAI's Custom GPTs team. Cursor's security lead. Lovable, Bolt, v0. Every product that gives each customer their own agents. Today they hand-roll: per-tenant scoping, agent-token rotation, vault for the customer's third-party credentials, revocation when a customer is banned. Their handrolled stack typically has gaps — usually missing some of: token-family cascade, customer-fleet revoke, agent-type bulk revoke, vault-cascade. SharkAuth ships all five layers as a single binary with the cryptographic primitives done correctly."

### Customer category appendix (use this for YC application long-form + interview)

Concrete platforms that fit the wedge today, by vertical. Use these names freely in interviews — they ground the pitch in real targets.

**1. AI coding assistants:** Cursor, Replit Agents, Lovable, Bolt, v0, Devin, GitHub Copilot Workspace, JetBrains AI Assistant, Cline (hosted), Aider hosted, Codeium agents, Continue.dev hosted, Sourcegraph Cody enterprise.

**2. Customer support AI:** Intercom Fin, Decagon, Crescendo, Sierra, Maven AGI, Forethought, Ada, Lorikeet, Espressive, Cresta.

**3. Sales / outbound AI:** Clay, Apollo AI, Outreach AI, Salesloft AI, 11x, Artisan, Regie, Lavender AI, Smartlead AI, Instantly AI.

**4. Voice / phone AI:** Bland, Vapi, Retell, Synthflow, Air AI, ElevenLabs Conversational, Hume EVI hosted, Soul Machines, Phonely.

**5. Workflow automation AI:** Zapier AI, Zapier Central, Make AI, n8n hosted, Pipedream AI, Lindy AI, Tray AI, Workato AI.

**6. Custom-agent platforms:** OpenAI Custom GPTs (Actions OAuth — direct fit), ChatGPT Apps SDK, Anthropic Custom Agents (when shipped), Google Gemini Gems, GLAM AI, Poe Apps.

**7. Personal / lifestyle AI:** Rabbit R1, Friend.com, Personal AI, Saga, Mem AI, Tab, Tana AI, Avi.

**8. Vertical industry AI (highest $ per platform):**
- Legal: Harvey, Spellbook, Eve, Robin AI, Hebbia
- Medical: Hippocratic, Glass, OpenEvidence, Abridge, Ambience, Suki
- Finance: Hebbia, Crux, DataHerald, Salient, Eigenlayer
- HR: Mercor, Paradox, HireVue AI, Eightfold AI agents

**9. Browser-based agents:** Browser-use, Multion, Browserbase, Anchor Browser, Tinybird AI, Lavi, Camelot.

**10. Code review / DevSecOps AI:** Greptile, Coderabbit, Codium, Snyk DeepCode AI, Sourcegraph Cody enterprise, Aikido, Semgrep AI, Continue.dev enterprise.

**Second-order categories (not direct platform sales but real volume):**
- Self-hosted in regulated industries (banks, healthcare, legal, defense) — open-source self-hosted is the only viable path for some compliance regimes
- Government / public sector (GSA, DoD, FedRAMP-track agentic tools)
- Security industry recursion (Wiz AI, Snyk AI, Dragos AI, CrowdStrike Charlotte AI — security tools shipping agents need agent security)

### TAM math (for YC application "how big could this be?")

**Direct platform plays:** ~50-100 named platforms today fit the wedge. ARR potential per platform: $30K-$500K depending on customer count. Conservative 50 × $100K = $5M ARR ceiling at scale; aggressive 200 × $200K = $40M ARR over 3-5 years.

**Self-hosted / regulated:** 10-100x volume on direct platforms, larger contracts ($50K-$500K+ each). Where Auth0 and Stytch made most of their revenue.

**Cloud tier (when shipped):** per-end-customer usage billing scales linearly with each platform's customer base. A platform with 100K end-customers at $0.001/agent-action/month easily produces $1M+ ARR from one customer relationship.

**Comparable comps:**
- Auth0 — acquired 2021 by Okta for $6.5B (identity for humans)
- Stytch — Series B 2022 at $1B (passwordless for humans)
- WorkOS — Series B 2023 at ~$1B (enterprise SSO)
- Clerk — Series B 2024 at $400M (auth for SaaS)
- Permit.io — Series A 2024 at ~$50M (authorization-as-service)

The auth-platform comp set is billion-dollar territory. "Auth platform for agents" plausibly maps to the same exit potential, with the bet that compliance maturity in the agentic-AI economy hits within 3-5 years.

### 5-year arc for YC application "long-term vision"

- **2026 (launch):** AI-coding-assistant + voice-AI + customer-support-AI early integrators (categories 1, 4, 2)
- **2027:** workflow-automation + custom-GPT-platform expansion (categories 5, 6) — pattern recognition kicks in across the AI economy
- **2028:** vertical industry AI hits SharkAuth as compliance demands rise (category 8 — legal/medical/finance/HR contracts at $200K+/year)
- **2029:** enterprise self-hosted becomes major revenue line (regulated industries, gov)
- **2030:** cloud tier dominant; SharkAuth is the de facto agent-auth standard, $100M+ ARR plausible if AI economy grows as projected

This arc has fund-returner shape. Plausible $100M ARR year 4, $500M+ ARR year 6, billion-dollar exit territory year 7-9. The wedge is concrete enough that even the haircut version (50% of these categories adopt, half the ARR projection) still returns a typical YC check 100x+.

### YC interview pushback to expect: "Doesn't Claude Code already do this?"

Same answer as before — Claude Code does internal subagent dispatch (orchestration, not auth). SharkAuth handles agent-to-external-resource auth scoped to a specific customer's identity, with five-layer revocation. Different problem.

**Why this version (refined post-Claude-Code-boundary observation):**
- "Products that give customers their own agents" names the architecture concretely. Each customer has agents that are tied to THAT customer specifically — not shared, not pooled. SharkAuth's model maps exactly.
- "Platform" yellow flag word avoided.
- Concrete examples that fit: Replit Agents, OpenAI Custom GPTs, Cursor, Lovable, Bolt, v0, Zapier AI, CrewAI Cloud, Decagon, Bland, Vapi, Intercom Fin.
- Three concrete behaviors. Zero adjectives. The fear ("customer's agent abuses your system") is named in the second sentence.

### Customer answer for YC interview

When asked "who is your customer?":

> "Replit's eng team shipping Replit Agents to their customers. OpenAI's Custom GPTs team. The auth lead at Lovable, Bolt, or v0. Cursor's security team. Each of these products gives every customer their own agents. Today they hand-roll: per-tenant scoping, agent-token rotation, vault for the customer's GitHub/Slack/Gmail credentials, revocation when a customer is banned. That stack is fragile and usually has gaps. SharkAuth ships it as a single binary with the cryptographic primitives done correctly."

Don't say "developers." Don't say "AI startups." Name companies. Name roles. Name fears.

### YC interview pushback to expect: "Doesn't Claude Code already do this?"

Almost certainly comes up. Anticipated answer:

> "Claude Code handles delegation INSIDE its harness — its code-review-agent calls its test-runner-agent as a function call. That's orchestration, not auth. SharkAuth fits a different problem: when one of those agents needs to push to the customer's GitHub, send email through the customer's Gmail, or call the customer's Stripe API. That's agent-to-external-resource auth, scoped to ONE specific customer's identity, with cryptographic proof that it was that customer's authorized agent and not someone else's. Claude Code doesn't solve that — it doesn't have to. The platform shipping a Claude Code-style product to enterprise customers does. That platform is our customer."

This reframes the question. Claude Code (and OpenClaw, Cursor agent mode, etc.) are not competitors. They are the AGENT TECHNOLOGY their customers ship inside the platforms that buy SharkAuth.

Variant of the same answer, shorter (use if interview is rapid):

> "Claude Code does delegation between its own subagents. We do delegation between an agent and the customer's external resources. Different problem. Claude Code's customer is the platform that ships it to enterprises — that platform is ours."

**Secondary answer — only use if interviewer pushes a second time** (logged 2026-04-26, see `09-post-launch-harness-skill-eureka.md`):

> "Claude Code is also our distribution channel. Post-launch we ship a Claude Code skill plus an MCP server wrapper. Subagents inside any harness — Claude Code, Cursor, Cline, OpenClaw — self-register against shark via the existing DCR endpoint with zero harness-team buy-in. Harnesses become our user-acquisition surface, not our competition. Every harness install is a foothold inside a platform team's existing developer flow."

Do NOT lead with this. The five-layer revocation answer above is the load-bearing one. The skill / MCP answer is a sales objection-neutralizer used inside deals, not a primary pitch.

### "Why did you pick this idea?"

Lead with transport hack. One paragraph. Then bridge to agent auth.

```
At <age> I cracked the cryptography of <city> public transport, reverse-engineered
the API, and could read/write transit data. I never used it. I just had to know
how it worked. That obsession — taking apart auth systems to understand them —
became the spine of how I think.

When agent protocols (MCP, OAuth 2.1, DPoP, RFC 8693) started landing in 2025,
I noticed the same pattern: the standards exist, but every team is hand-rolling
them on top of human-session infrastructure. I built SharkAuth because I wanted
the auth layer that I would have used at every previous project, instead of
hand-rolling DPoP signers myself.
```

### "Why are you the right team?"

Don't apologize for solo. Lean into compressed shipping.

```
I shipped 100,000+ lines of working, RFC-correct code in approximately one month,
solo, with Claude Code as my collaborator. The hardest cryptographic primitives
(DPoP proof validation, RFC 8693 token exchange with `act` claim chains, agent-aware
audit) are implemented and tested. I tried to recruit a cofounder. The pitch did
not land with the people I asked. Rather than wait, I built the technical bet
alone. I will resume the cofounder search post-launch with traction in hand.
```

### "What's new about what you make?"

```
Auth0, Clerk, WorkOS, Stytch all treat agents as machine-to-machine OAuth clients.
SharkAuth treats agents as a first-class entity type with delegation primitives
that human-session auth cannot replicate without rewriting their token model.

Specifically:
- DPoP RFC 9449 — every token cryptographically bound to the agent's keypair.
  Token theft alone is useless without the private key.
- RFC 8693 token exchange — delegated sub-tokens record an `act` claim chain.
  Audit log shows the full actor lineage, not just the final caller.
- Agent-aware audit + `may_act` policy enforcement — which agents can delegate
  to which, with cryptographic proof at each hop.

SharkAuth is also a complete auth server — SSO, magic-link, organizations, RBAC,
full human auth. This is deliberate: an agent acts on behalf of a human, so
splitting auth across two servers (Auth0 for humans, something else for agents)
breaks the audit chain. SharkAuth is one server, one delegation-aware trust
chain, from human login to the agent's third-hop sub-token.

The 12-18 month window is real: incumbents will retrofit agent semantics, but
they have to rewrite their token model AND their identity model to do it.
We're agent-native from line one, with humans included.
```

### Anticipated YC interview pushback: "Why not just use Auth0 + a delegation library?"

Have this answer rehearsed:

> "Because the user logging into your app and the agent acting on their behalf are the same trust chain. Splitting them across two servers means you can't audit the chain end-to-end — you have separate logs for the human session and the agent's tokens, with no cryptographic link between them. Shark is one server, one audit log, one delegation-aware token model from human login to agent's third-hop delegated sub-token. That's why human auth + agent auth in the same binary isn't feature soup, it's the architecture."

### "Where do you live now, and where would the company be based after YC?"

> Currently Monterrey, Nuevo León, Mexico. Will relocate to SF Bay Area for the 3-month YC batch (in-person, mandatory). Plan to operate the company from Monterrey post-batch. Reasons: cost structure (3-5x lower burn), talent pipeline (Tec Monterrey + UANL), US Central time zone alignment, MXN-denominated costs against USD revenue. Will be in SF regularly for fundraising and partner meetings.

(Don't be vague. YC asks because it's a yes/no constraint for the batch itself. They DO NOT require post-batch SF residency. LATAM YC companies that thrived from home base: Rappi, Kavak, Bitso, Cornershop, Nubank-adjacents, Truora, Habi. Be honest about deliberate base choice — YC respects deliberate thinking.)

### Anticipated follow-up: "Why not stay in SF post-batch?"

If asked in interview, have this answer rehearsed:

> "Solo-founder burn rate is the metric I optimize for in year one. Monterrey gives me 3-5x more runway per dollar raised. Senior identity-systems engineers exist remote-first and through Tec Monterrey alumni at half SF cost. I will be in SF for every fundraise and major partner meeting — probably 6-8 trips a year. The company HQ being in MTY is a deliberate cost structure choice, not a lifestyle one. I want this company to outlast its first round."

This signals: you've thought about it, you know the tradeoffs, you're not running away from SF for personal reasons. It's a strategic call.

### "How long have you been working on this?"

> Approximately one month of full-time solo execution. Roughly 100,000 lines of code, RFC-correct primitives, working binary, smoke suite passing 375 tests. Pre-launch as of <date>; public launch <Monday date>.

### "How far along are you? Do you have a beta? Are you launched? Do you have users?"

This is the brutal one if you have zero users. Acquire one before submitting.

```
Public launch <Monday date>. As of <submission date>: <N> GitHub stars, <N>
integrators publicly using SharkAuth (first: @<handle> at <project>), <N>
issues from real users, HN front-page reach <X>. Smoke suite: 375 tests passing.
Open source — repo at <URL>.
```

### "If accepted, will both founders work on this full time? (Co-founders only)"

> Solo. Full-time on SharkAuth. No other employer or commitment.

## Don't-do list

- Don't lie or inflate numbers. YC verifies. Worst-case discovery during interview = instant disqualification.
- Don't pad with credentials you don't have. The work itself is the credential.
- Don't say "we" when you mean "I." Solo is fine. Faking team is not.
- Don't lead with the transport hack on the WRITTEN application. Lead with what SharkAuth does. The hack belongs in "why this idea" or in the video.
- Don't quote Garry, PG, or YC partners back at YC. They've heard it.

## After submission

- Continue shipping. YC sometimes flips a no to yes if traction explodes between submission and decision.
- Engage with YC partners on Twitter without being thirsty. Reply substantively to their tweets, not "great point!"
- If you get an interview: practice the 10-min pitch with three different people. Time it. Cut anything not in the question being asked.
- If you don't get an interview this batch: apply again next batch with the next 3 months of traction. Brian Chesky was rejected multiple times. Tom Blomfield was rejected before Monzo got in. The application is a snapshot, not a verdict.

## My honest read

Strong narrative + strong demo + zero users = ~3-7% interview chance.

If between now and submission you can convert ONE name from `.planning/launch-targets.md` into a publicly-stated integrator, that climbs to 5-12%. That single conversion is worth more than another week of code.

If two or three convert, this becomes a real shot.

The bottleneck is not the application. The bottleneck is named users by submission day. Spend your post-launch hours on that, not on rewriting the application.

## GitHub stars and HN ranking — what they actually buy

YC partners (Garry Tan most explicitly) have stated repeatedly that stars alone are noise. Stars + named users + HN traction read as a complete signal. Stars without anything else read as "good launch execution, unclear product-market fit."

### Star-count rough upgrade math (with security-moat pitch + transport hack)

Updated post-pitch-refinement (security-moat framing locked in 50-word answer):

| Launch outcome at submission | Pre-security-pitch | With security-moat pitch |
|---|---|---|
| 200 stars, 0 users | ~4-7% | ~6-9% |
| 500 stars, 0 users | ~5-9% | ~7-11% |
| 1k stars, 0 users | ~7-12% | ~9-14% |
| 1k stars + 1 named integrator | ~10-18% | ~13-20% |
| 1k stars + 3+ named integrators | ~15-25% | ~18-28% |
| HN #1-5 + 1k stars + 1 integrator | ~15-22% | ~18-25% |
| HN #1-5 + 1k stars + 1 platform conversation* | ~17-24% | ~21-28% |

*platform conversation = "the security/auth lead at <Cursor/Replit/Lovable> replied to the launch and we're scheduled to talk." Even an unscheduled "interesting, follow up in 2 weeks" reply registers as evidence for the architectural pitch.

**The +2-3 percentage point bump comes from:**
- Differentiated wedge ("agent security" > "agent OAuth" in YC reviewer eyes)
- Fear-based pitch ("rogue insider") resonates with enterprise reviewers more than feature-based pitch
- More shareable framing → easier named-user conversion in launch week

**Important nuance:** the architectural pitch RAISES the bar on user evidence. "Agent OAuth" needed one MCP server builder. "Agent security platform" implies enterprise/platform buyers. YC will want at least ONE conversation with a real platform team's security/auth lead — even unconverted, the conversation is evidence.

**Add to launch outreach:** the security/auth lead at one or two AI-coding-assistant companies (Cursor, Replit, Lovable, Bolt, Vercel v0). Send the cascade-revoke demo. A reply lands the architectural-pitch validation YC wants.

These bands are noisy. They are not promises.

### What 1k stars actually buys you

1. **HN visibility loop.** Top of /show drives more stars which drives more visibility. Self-reinforcing for ~36 hours.
2. **Inbound DM traffic.** 1k stars typically yields 20-50 unsolicited DMs in launch week from developers who saw it. Your job is converting 3-5 of those into named integrators by Saturday.
3. **YC partner direct attention.** Garry Tan, Diana Hu, Harj Taggar all check HN and Twitter daily. HN #1-5 with a memorable demo gets noticed without applying. Sometimes leads to a direct partner DM. Be ready: respond fast and specific.
4. **Permission to say "took off on launch"** in the application. "1k stars in 7 days" is concrete and verifiable.

### What 1k stars does NOT buy you

- Demand evidence. YC reviewers read "1k stars, 0 users" as "good marketing, unclear if anyone needs it." Stars are distribution signal, not product-market signal.
- A waiver from the "how many users?" question. Stars do not substitute. Still need named users.
- A direct partner intro. Even if Garry sees the HN post, he expects you to apply through the form.

### HN ranking matters more than raw star count

HN #1-3 puts you in front of every YC partner that morning. 500 stars + HN #2 beats 1k stars + HN #15 for application impact.

Tactical: time your post for Monday 8-10am ET. Get 4-8 upvotes from friends/communities in the first 30 minutes. Stay online to reply to every comment in the first 2 hours. HN ranking depends on early engagement velocity.

### The conversion ratio that actually moves YC

`stars → DMs → named integrators`

- 1k stars + 0 DMs = bad (no real interest, just gawking)
- 1k stars + 30 DMs + 0 named integrators = neutral (you're not closing)
- 1k stars + 30 DMs + 5 named integrators = jackpot — this is the ~15-25% band

Your job between Tuesday and Friday is not to chase more stars. It is to close every promising DM into a named, public integration. One Twitter post from a developer saying "I integrated SharkAuth into <my project>, here is the diff: <link>" is worth more than 500 additional stars.

### Submission-day math

Two scenarios:

**Scenario A: Submit Saturday with 1k stars + 0 DMs converted.**
YC reviewer reads: "good marketing, unclear if anyone needs it." Interview chance ~7-10%.

**Scenario B: Submit Saturday with 600 stars + 2 named integrators publicly using SharkAuth + HN #3 screenshot in the application.**
YC reviewer reads: "the product is real, distribution works, founder closes." Interview chance ~12-18%.

Scenario B beats Scenario A even with fewer stars.

### Action priority Tuesday-Friday post-launch

1. Reply to every DM within 4 hours, ideally 1 hour. Speed signals seriousness.
2. Offer to integrate SharkAuth into their project personally. "I'll do the integration PR myself, you review it." Most will say yes if asked.
3. After successful integration, ask them to post about it publicly. Provide the post draft so they only have to copy-paste-edit.
4. Track named integrators in `playbook/launch-day-log.md`. Update the YC application as each one lands.
