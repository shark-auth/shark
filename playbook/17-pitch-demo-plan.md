# Pitch Demo — Blast Radius (4 beats)

> **Companion to** `15-concierge-demo-plan.md`. Does not replace it. Concierge stays as deep-dive 20-step. This is the **pitch room** demo — 90 seconds, no narration crutch, single OH-SHIT moment.

**Goal:** Convert in <2 min. One claim, one visual, one click. No DPoP talk, no cascade lecture, no 20-step parade.

**Audience:** investor, design-partner CTO, hallway track at conf. Has 90s. Wants the moat in one frame.

---

## The 4 beats

| # | Beat | Time | What screen shows | What you say |
|---|---|---|---|---|
| 1 | `./shark serve` | ~16s | terminal: migrations apply, root key prints, server up | nothing. let it run. close terminal when ready. |
| 2 | Open dashboard → canvas | ~5s | delegation canvas loads | nothing. silent navigation. |
| 3 | Tree on screen | ~10s | `María → Concierge → Flight Booker → Payment Processor` with scopes, timestamps, act-chain badges | nothing. let them read it. |
| 4 | Point + revoke | ~15s | hover Payment Processor node → click **Revoke** → node greys out, others stay green | "Este agente tiene credenciales de Stripe de María. Si se compromete via prompt injection — *click* — solo muere él. Los demás siguen vivos." |

**Total runtime:** ~45-60s of screen, ~15s of speech. Pad with silence.

---

## Why this works

- **Silence beats explanation.** Tree IS the pitch. Words narrate a screenshot they're already looking at.
- **One sentence carries the moat.** Blast radius = single node. Everyone else has org-wide kill switch or nothing.
- **No jargon.** No DPoP, no RFC 8693, no cascade theory. Visual carries it.
- **Fast setup is the second moat.** 16-second `serve` proves "self-host is real, not a 3-week ops nightmare."

---

## What we are NOT showing

- Magic-link signup flow
- DCR + DPoP token issuance
- Vault FieldEncryptor mechanics
- Cascade-revoke explanation (Maria → all)
- Bulk pattern revoke
- Audit log internals
- 20-step Maria Chen narrative

These live in `15-concierge-demo-plan.md` for technical deep-dives. **Do not mix audiences.**

---

## Pre-flight state (set up off-camera)

Before the pitch, run the existing concierge demo through step 15 (depth-3 chain established, vault populated, audit log full). Stop there. **Do NOT run steps 17-20** — that destroys the visual we need on the canvas.

```bash
python tools/agent_demo_concierge.py --until-step 15 --no-cleanup
```

(If `--until-step` flag does not exist yet, add it as part of this work — see Phase B.)

Resulting canvas state:
```
María (human, owner)
└─ Travel Concierge          [client_credentials + DPoP]
   ├─ Flight Booker           [act_chain depth 2, vault: Amadeus]
   │  └─ Payment Processor    [act_chain depth 3, vault: Stripe]   ← revoke target
   ├─ Hotel Booker            [act_chain depth 2, vault: Booking]
   ├─ Calendar Sync           [act_chain depth 2, vault: GCal]
   └─ Expense Filer           [act_chain depth 2, vault: Concur]
```

---

## What needs to exist in the dashboard

| Need | Likely status | Action |
|---|---|---|
| Canvas page renders delegation tree from audit log / agent graph | partial — verify | audit, screenshot current state |
| Node hover → side panel with scopes/timestamps/act_chain | unclear | inspect, file gap |
| Single-node revoke button on side panel | unclear | likely missing — implement |
| Revoked node visually distinct (greyed, strike, red border) | unclear | CSS pass |
| Sibling nodes remain visually green/live after target revoke | property of single-node revoke working correctly | verify |

**Action ordering:**
1. Audit dashboard canvas — what renders today, what is missing?
2. Gap-fill: single-node revoke UI + visual revoked state
3. Add `--until-step N` flag to `agent_demo_concierge.py`
4. Dry-run pitch end-to-end on real machine, screen-record, time it
5. If >90s, cut beats — most likely cut is beat-2 silence padding

---

## Phases

### Phase A — Audit (one sonnet subagent, parallel-safe)
- Open dashboard, navigate canvas, screenshot every state after running `agent_demo_concierge.py` to step 15
- Report: does tree render? hover work? revoke button exists? revoked-state styling?
- Output: `playbook/plans/2026-04-27-pitch-demo-audit.md` with screenshots + gap list

### Phase B — Gap fill (sonnet, sequential after A)
- Implement whichever of these are missing:
  - single-node revoke endpoint already exists (`POST /admin/agents/{id}/revoke`) — wire to UI button
  - revoked-state CSS (greyed node, red border, strike on label)
  - `--until-step N` arg in `agent_demo_concierge.py`
- Atomic commits per gap

### Phase C — Rehearsal (manual, Raul)
- Run pre-flight, screen-record full 4-beat pitch
- Trim to <90s
- Save as `demos/pitch-blast-radius.mp4`
- Iterate copy on beat 4 — Spanish + English variants

---

## Beat-4 copy variants

**Spanish (default — Raul's pitch):**
> "Este agente tiene las credenciales de Stripe de María. Si se compromete via prompt injection, hago esto— *click* —y solo muere él. Los demás siguen vivos."

**English:**
> "This agent holds María's Stripe credentials. If it gets prompt-injected, I do this— *click* —and only it dies. Everything else keeps running."

**Even shorter (conf hallway):**
> "Stripe creds live here. Compromised? *click.* Blast radius: one node."

---

## Risks

- **Canvas does not visually distinguish revoked nodes** → pitch falls flat. Phase A must verify before rehearsal.
- **Revoke takes >500ms to reflect on canvas** → kills the *click → instant grey* moment. May need optimistic UI update.
- **Tree layout shifts when node revokes** → distracting. Lock layout, only change node style.
- **Screen-record at wrong resolution** → labels unreadable on projector. Record at 1080p min, 1440p preferred.

---

## Out of scope

- Replacing concierge demo
- Adding new shark features
- Marketing copy beyond beat-4 line
- Sales-deck slides around the demo
