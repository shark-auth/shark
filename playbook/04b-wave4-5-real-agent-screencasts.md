# Wave 4.5 — Polished Launch Video (B) + Real-Agent Secondary Screencasts

**Budget:** 8-12h CC + your hands · **Outcome:** polished 90s startup-launch video for YC application + 2-3 real-agent screencasts for launch-week-2 campaign · **Run Wed-Fri post-launch (NOT before)**

## Two artifacts, sequenced

### Artifact 1: Polished 90s startup-launch video (Video B)

**Why this exists:** the raw 30s demo (Wave 3 + Wave 4) wins HN. The polished 90s video wins YC + LinkedIn + investor DMs. Different audiences need different formats.

**When to record:** Wednesday night, AFTER you have launch data to reference (stars, named integrators, DMs converted). Not Sunday — Sunday adrenaline shows on camera as anxious.

**Full script + production notes:** in `07-yc-application-strategy.md` under "Application video — TWO formats, sequenced." Don't duplicate here. Single source of truth.

**Distribution:**
- YC application video field (the official video)
- Twitter pinned tweet (Thursday morning)
- LinkedIn launch post Thursday — caption: "1 week into SharkAuth — here's what shipped and what's next"
- Landing page hero (when you have one)
- Founder DM attachments to high-value targets

**Effort:** 5-8h CC for the recording, editing, voiceover sync. Add 2-3h if you do music + color grading.

### Artifact 2: Real-agent secondary screencasts

**Why secondary, not primary:** Real agents (Claude Code, OpenClaw, Cursor, Cline) are non-deterministic, network-dependent, can rate-limit mid-recording. HN demands reproducibility — "I tried it, doesn't work" in a top comment kills the post.

**Strategy:**
- Primary demo (Wave 3): deterministic `shark demo delegation-with-trace` with vault hop. Trust anchor.
- Polished video B (above): the YC + investor artifact.
- Secondary demos (below): real agents using SharkAuth in their actual workflows. Social proof for the founder/builder audience on Twitter, LinkedIn, Reddit follow-ups, MCP Discord.

**Distribution math:** 1 primary + 1 polished + 2-3 secondary = launch-week content for 7-10 different posts across HN, Twitter, LinkedIn, Reddit, Discord, MCP Discord. Stretches the launch into a 5-day campaign.

## Target real agents

Pick 2-3. Each gets its own 30-90s screencast and its own caption.

### Claude Code (Anthropic) — highest priority

**Why:** flagship Anthropic agent, recognizable, MCP-first, audience overlap is enormous.

**The shot:** Claude Code spinning up an MCP server that uses SharkAuth for OAuth. User logs in via shark magic-link, authorizes Claude Code as an agent, Claude Code makes a delegated call to a third-party API through SharkAuth's vault.

**Setup time:** ~90 min CC (configure MCP server, wire SharkAuth as the OAuth backend, record).

**Caption (Twitter):** "Claude Code authenticated through SharkAuth. DPoP-bound, audited, delegation chain visible. Watch the audit log update in real time as Claude Code calls a vaulted Gmail API."

### OpenClaw — high priority

**Why:** rising agent CLI, audience is exactly the "agent infra builders" we want as integrators. User is already familiar with it.

**The shot:** OpenClaw issuing an MCP request through SharkAuth. Show the OpenClaw config file pointing to a SharkAuth-protected MCP server. Run a command, watch SharkAuth audit log show the agent action with full chain.

**Setup time:** ~60 min CC.

**Caption:** "OpenClaw authenticated via SharkAuth. Same auth backend serves humans (you) and agents (your CLI). One trust chain, one audit log."

### Cursor or Cline (optional, third)

**Why:** broad VS-Code-extension audience overlap. Less MCP-native than Claude Code/OpenClaw but captures the IDE-agent crowd.

**The shot:** Cursor's agent feature making a call to a SharkAuth-protected backend. Show the OAuth handshake in Cursor's logs.

**Setup time:** ~60 min CC.

## Recording requirements

For each:
- 30-90 seconds, never longer
- 1080p, 30fps, mp4
- No personal handles, project paths, or API keys visible (blur if needed)
- Show: agent issuing request, SharkAuth dashboard updating live, audit log row appearing
- End with the SharkAuth dashboard showing the new audit event with the agent's actor_id

Save to `playbook/screencasts/` as `claude-code.mp4`, `openclaw.mp4`, etc.

## Rollout schedule (full launch-week campaign)

- **Sunday night:** record Video A (raw 30s demo), draft launch posts, send 10 personalized outreach DMs
- **Monday 9am ET:** primary launch (HN + Twitter + Discord + LinkedIn) using Video A. Stay on threads 9am-12pm responding to every comment.
- **Monday 12-2pm ET:** Reddit posts (r/golang, r/selfhosted, r/LocalLLaMA) — only AFTER HN front-page ranking confirmed
- **Monday afternoon-evening:** convert inbound DMs. Offer to do integration PRs personally for promising contacts.
- **Tuesday 10am ET:** Claude Code real-agent screencast on Twitter + reply on original HN thread with "1 day in: here's Claude Code using it"
- **Wednesday 10am ET:** OpenClaw real-agent screencast on Twitter + cross-post to MCP Discord
- **Wednesday night:** record Video B (polished 90s startup-launch video) using launch data + any named integrators in hand
- **Thursday 10am ET:** post Video B everywhere — Twitter pinned, LinkedIn launch post ("1 week in"), update README hero, embed in YC application
- **Friday:** retrospective post — "what I learned from launch week, links to all artifacts" + Cursor/Cline screencast if recorded
- **Friday-Saturday:** finalize YC application using Video B as application video. Submit Saturday.

This pattern keeps the launch alive 5 days instead of 1. Each new screencast brings a new wave of attention back to the repo. The polished Video B Thursday-Friday IS the YC application artifact.

## Twitter thread structure for each secondary demo

Standalone tweets, 4-6 per agent:

1. Open with the visual: "Watching <agent> authenticate via SharkAuth ↓"
2. Why this matters: "<one line on the differentiation visible in this clip>"
3. Code: "Here's the config — <X lines of Y>"
4. Audit log: "Every action is auditable per agent. Here's the trace from this clip."
5. Repo link + "DM me if you want to integrate this in your <thing>"

## Anti-patterns

- **Don't fake the recording.** If the integration breaks, fix it before recording. Doctored screencasts get caught and destroy credibility.
- **Don't release all secondary demos same day as launch.** Day-1 traffic should land on the deterministic demo. Day-2 onward is for variety.
- **Don't post the same screencast to 5 places.** Each platform gets one place. HN doesn't reward repost amplification.

## Definition of done for Wave 4.5

- 2-3 real-agent screencasts recorded and saved to `playbook/screencasts/`
- Each has its own Twitter caption + LinkedIn caption + Reddit-appropriate caption
- Rollout schedule confirmed
- All recordings pre-tested on a clean machine before publishing

## Skip condition

If Wave 1 + Wave 1.5 + Wave 2 + Wave 3 + Wave 4 takes you to Sunday night burnout, skip 4.5 entirely. The deterministic primary demo is sufficient for launch. Real-agent screencasts can ship in launch-week-2 with no loss of opportunity.
