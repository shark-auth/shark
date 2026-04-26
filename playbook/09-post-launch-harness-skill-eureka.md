# 09 — Post-Launch Eureka: Harness Self-Register Skill + MCP

**Logged:** 2026-04-26
**Status:** POST-LAUNCH ONLY (W19+). Monday launch unchanged.
**Wedge:** UNCHANGED — "auth for products that give customers their own agents."

This file is a **secondary lever**, not a positioning pivot. The five-layer revocation pitch (see `07-yc-application-strategy.md` LOCKED 55-word) remains the lead in every external surface.

---

## What this file is NOT

- Not a 55-word rewrite. The 55-word stays locked.
- Not a Monday launch asset. See `08-launch-scope-cuts.md` CUT 7.
- Not a primary pitch to platform CEOs (Cursor, Replit, Lovable). They buy on five-layer revocation, not on harness skills. Skill is a secondary objection-neutralizer used INSIDE deals, never to open them.
- Not a new product line. Skill + MCP wrapper is a 4-5 day W19+ ship, not a strategic axis.

---

## The Eureka (one paragraph)

Originally framed shark as failing harness integration ("hard to get Cursor / Replit / Claude Code to adopt"). Wrong frame. The DCR self-register endpoint (`POST /oauth/register`, RFC 7591) already ships at `internal/api/router.go:755`. CIMD validation, DPoP keypair, audience binding (RFC 8707), and token exchange (RFC 8693) all ship today. Demo #1 (`demos/DEMO_01_MCP_COLD_START.md`) already documents the 8-second cold-start flow end-to-end.

The missing piece is **wrapper distribution**: a Claude Code skill + a `shark-mcp` server that any harness installs. Subagents inside that harness self-register against shark with **zero harness-team buy-in**. We don't need Cursor's eng team to integrate. The customer (Cursor's user) installs the wrapper; Cursor's binary is unchanged.

---

## Existing primitives (no new backend work)

| Primitive | Where it ships | Reference |
|---|---|---|
| DCR self-register (RFC 7591) | `internal/api/router.go:755` | `POST /oauth/register` |
| CIMD validation | same handler chain | `client_id_metadata_document_uri` |
| DPoP keypair on client (RFC 9449) | client-side ECDSA P-256 | `tests/smoke/test_agent_flow_dpop.py` |
| DCR smoke coverage | ships | `tests/smoke/test_agent_flow_dcr.py` |
| Audience binding (RFC 8707) | ships | demo flow step 6 |
| Token exchange (RFC 8693) | ships | Wave 2 Method 2, used by demo |
| Cold-start architecture diagram | ships | `demos/DEMO_01_MCP_COLD_START.md` |
| Marketplace backbone narrative | ships | `demos/DEMO_01_AGENT_MARKETPLACE.md` |

Backend is ready. Wrappers are the gap.

---

## The two wrappers (what gets built W19+)

### Wrapper A: shark Claude Code skill

- Lives at `skills/shark-cc/SKILL.md` in this monorepo (single source of truth, no second repo)
- Triggers: user says "register this agent" / "set up shark for this project" / `/shark-register`
- Reads project context (language, framework, existing `.env`)
- Calls `POST /oauth/register` against `$SHARK_URL` (default `http://localhost:8080`, fallback prompt)
- Writes `SHARK_CLIENT_ID` + DPoP keypair into project-local `.env` (gitignored)
- Outputs the 10-line SDK snippet from Wave 2 ready to paste into the user's code
- Optional: scaffolds a minimal `shark` middleware file in user's framework of choice (FastAPI, Express, Gin)

### Wrapper B: shark-mcp server

- Subcommand on existing `shark` binary: `shark serve --mcp` (no second binary to ship)
- Exposes shark's agent CRUD + token mint + revoke as MCP tools (stdio + SSE transport)
- Any MCP-compatible client — Claude Desktop, Cursor, Cline, OpenClaw, Continue.dev — calls it directly
- Self-registers itself on first call via DCR; no pre-shared secret
- Tools surfaced (initial set): `register_agent`, `mint_token`, `revoke_agent`, `list_my_agents`, `get_audit_log`

Both wrappers reuse the existing DCR endpoint. Neither requires a backend change.

---

## Why this matters (in priority order)

### 1. SALES OBJECTION NEUTRALIZER (primary use)

Platform eng-team objection on a sales call: "how much work for us to integrate?"

**Answer:** "Two paths. Either ship a 50-line `shark` middleware in your binary (1-2 dev days), OR your end-users install our Claude Code skill / MCP and your subagents self-register via DCR with zero binary change on your side. You pick the model that fits your release cycle."

This UNBLOCKS deals that would have died on integration cost. Most platform teams will pick the middleware path eventually — but having the skill path means the deal doesn't stall while their eng team prioritizes.

### 2. SECONDARY YC PUSHBACK ANSWER

YC interview question: "Doesn't Claude Code already do this?"

**Primary answer (unchanged, see 07):** orchestration vs auth, different problem, Claude Code is the agent technology our platform customers ship.

**Secondary answer (new, use only if pushed twice):** "Claude Code is also our distribution channel. We ship a skill that lights shark up inside any harness — Claude Code, Cursor, Cline, OpenClaw. Harnesses become our user-acquisition surface, not our competition. Every harness install is a foothold inside a platform team's existing developer flow."

### 3. POST-LAUNCH GROWTH SURFACE

Skill marketplace + MCP registry are install-count discoverable. Becomes a launch follow-up tweet ("now in your Claude Desktop") at W19/W20. Separate news cycle from Monday launch — wins a second wave of HN attention without diluting the first.

---

## When to use vs when NOT to use

**USE for:**
- Sales calls with platform eng leads (objection neutralizer)
- HN follow-up post 2-3 weeks after Monday launch
- Skill marketplace + MCP registry submissions
- Cold DM to harness power-users on Anthropic MCP Discord, Cursor Discord, Cline Discord
- Demo at meetups where the audience is harness users (not platform CEOs)

**DO NOT use for:**
- The 55-word pitch (stays locked on five-layer revocation)
- Cold outreach to platform CEOs / founders (they buy security, not skills)
- YC application long-form lead paragraph
- Launch-day HN title or Twitter thread (would dilute the security-moat framing)
- Wave 4 launch posts in `04-wave4-launch.md`

---

## Distribution math (rough, honest bands)

| Channel | Time-to-first-100-installs | Conversion to paying platform |
|---|---|---|
| Claude Code skill marketplace | 2-4 weeks post-listing | Low (signals only) |
| MCP registry (Anthropic + Smithery) | 1-2 weeks post-listing | Low-mid |
| Direct DM to harness power-user → ref to skill | per-deal | Mid (if eng lead replies) |

Skill installs ≠ revenue. They are objection-neutralizer EVIDENCE ("100+ harnesses already self-register against shark") for sales conversations and YC follow-up applications. Treat install count as a credibility metric, not a north-star.

---

## Build sketch (W19+ scope, NOT pre-launch)

| Wave | Scope | Budget |
|---|---|---|
| W19a | shark CC skill (`skills/shark-cc/SKILL.md` + bash setup + project scaffolding) | 1 day CC |
| W19b | `shark serve --mcp` subcommand (Go, MCP stdio + SSE, 5 initial tools) | 2-3 days CC |
| W19c | Skill marketplace + MCP registry submission packets | 0.5 day human |
| W19d | Demo screencast: skill install → agent self-registers → first DPoP token in 30s | 0.5 day |

**Total:** ~4-5 days CC + 1 day human, post-Monday-launch. Single news cycle for both wrappers.

---

## Acceptance criteria

- `/shark-register` slash command in Claude Code creates a shark agent in **<10 seconds** end-to-end (cold start, no pre-existing config)
- `shark serve --mcp` invoked by Claude Desktop registers + mints first DPoP-bound token in **<15 seconds**
- Smoke test `tests/smoke/test_skill_register.py` covers happy path (DCR registration → token mint → audit log entry visible)
- README "Use shark inside Claude Code" section ships with W19a
- README "Use shark inside any MCP client" section ships with W19b
- Demo screencast renders cleanly at 1080p, ≤45 seconds, no voiceover (matching Video A discipline from `07-yc-application-strategy.md`)

---

## Anti-patterns

- **DO NOT** pitch shark as "the skill for Claude Code" externally. That's a feature, not a moat. Five-layer revocation is the moat.
- **DO NOT** let skill marketplace install count become the headline metric. It does not predict paying platforms.
- **DO NOT** build a proprietary-protocol skill before the MCP registry version. MCP is the open standard; the CC skill is a secondary wrapper. Ship both, but treat MCP as primary.
- **DO NOT** add wrappers for Cursor, Cline, OpenClaw individually. The MCP server covers all of them transitively. Resist N-platform-specific wrappers.
- **DO NOT** ship before Monday launch. Discipline matters — see CUT 7 in `08-launch-scope-cuts.md`.

---

## Cross-references

- Cold-start technical flow: `demos/DEMO_01_MCP_COLD_START.md`
- Marketplace backbone narrative: `demos/DEMO_01_AGENT_MARKETPLACE.md`
- DCR endpoint definition: `internal/api/router.go:755-760`
- DCR smoke coverage: `tests/smoke/test_agent_flow_dcr.py`
- Locked 55-word + YC pushback: `07-yc-application-strategy.md`
- Customer category appendix (platform list): `07-yc-application-strategy.md`
- Distribution plan baseline: `00-design.md`
- Launch scope discipline: `08-launch-scope-cuts.md` CUT 7

---

## Open questions

1. **Self-hosted shark URL discovery.** mDNS/bonjour, env var, or interactive prompt? Recommendation: `SHARK_URL` env first, fallback to `http://localhost:8080`, fallback to interactive prompt. No mDNS in v1 (adds dependency).
2. **Skill-issued client_id storage.** Project-local `.env` or `~/.shark/credentials`? Recommendation: project-local, gitignored. Multi-project users get clean separation.
3. **`shark-mcp` vs `shark serve --mcp`?** Recommendation: subcommand on existing binary. Avoids second-binary distribution overhead.
4. **Does the skill repo live in shark monorepo or separately?** Recommendation: monorepo under `skills/shark-cc/`. Single source of truth, single CI, single release tag.
5. **MCP transport priority — stdio first, SSE second, or both at W19b?** Recommendation: stdio first (simpler, covers Claude Desktop + Cline). SSE in W19b.5 if time permits, otherwise W20.
6. **License of the skill files.** Same as shark (MIT/Apache, whichever the repo uses). Confirm before submission to skill marketplace.
