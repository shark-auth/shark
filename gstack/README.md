# gstack/ — Launch Session Artifacts (2026-04-24)

This directory captures the full office-hours + autoplan session run on 2026-04-24. Produced during a single Claude Code session that audited the codebase from ground truth (not docs), ran a 4-reviewer autoplan pipeline, and produced the launch strategy, YC P26 application draft, and cold-DM kit.

**Canonical source** is in `~/.gstack/projects/shark-auth-shark/` (user-home, visible to tooling across sessions). This directory is a mirror for team visibility and pushing to the public repo.

## Read order

| # | File | What it is | When to read |
|---|---|---|---|
| 01 | `01-launch-design.md` | Design doc: problem, wedge, premises, approaches, 54-capability audit matrix with file-level citations | First. Source of truth for "what ships today." |
| 02 | `02-yc-application.md` | Full YC P26 application draft. All form fields filled. Submit-ready after Day 10 number updates. | When prepping YC submission. |
| 03 | `03-launch-playbook.md` | Original 10-day calendar. Superseded by file 05's compressed calendar. | Skim. |
| 04 | `04-autoplan-synthesis.md` | 4-reviewer pipeline findings (CEO + eng + DX + adversarial). 13 hard fixes, 5 taste decisions. | When making product/plan decisions. |
| 05 | `05-revisions-cold-dm-kit.md` | **Operative doc.** Compressed calendar for 2026-04-29 launch + cold-DM template + YC app patches. | Daily during launch sprint. |
| 06 | `06-refresh-rotation-patch.md` | Security patch: atomic refresh-token rotation. Closes a concurrent-refresh race. 3 patches + test. | Before any production deploy. |

## Locked decisions

- **Wedge:** hybrid — RFC 8693 agent delegation leads YC title, HN post, demo. README + landing keep broader value prop.
- **Launch:** Tuesday 2026-04-29 (external accelerator deadline).
- **Pricing:** deferred. YC app says "OSS free + hosted pricing during batch."
- **Partner goal:** personal commitment 3; YC app reports actual.
- **Version:** v0.9.0 on launch day. v1.0.0 after 2 weeks clean API.

## Feature status (from audit, not docs)

54 capabilities audited → 38 SHIPPED, 6 PARTIAL, 3 STUB, 7 MISSING. DX score 7/10. Cloud-readiness 4/10 (SQLite-only is the terminal blocker for multi-region hosted tier). See file 01 for full matrix.

## Hard fixes shipped Day 0 (2026-04-24)

- `LICENSE` file at repo root (MIT).
- `SCALE.md` documenting 4 in-memory stores + Q3 2026 roadmap.
- Repo-root cleanup (logs, dbs, internal docs, screenshots moved).
- Atomic refresh-token rotation patch prepared (see file 06).

## Still to ship before launch

- Binary release pipeline (goreleaser + GHCR + PyPI) — Day 1.
- OpenAPI spec for 30 core endpoints — Day 2.
- `shark doctor` diagnostic CLI command — Day 2.
- SDK naming unified to `@sharkauth/*` — Day 2.
- 3 new doc pages (mcp-5-min, agent-delegation, self-host-vs-cloud) — Day 2.
- Main demo video (5 min) + YC app video (1 min) — Day 3.
- Landing page on sharkauth.com — Day 4.
- Discord server public — Day 4.
- `v0.9.0` tag + launch — Day 5 (2026-04-29).

## Not in scope for the 10-day window

- SAML IdP mode (we consume SAML today, don't provide it).
- SMS OTP (no Twilio integration).
- PostgreSQL support (Q3 2026, unlocks hosted cloud tier).
- RFC 9728 Protected Resource Metadata.
- Homebrew tap.

Each is documented with a deliberate roadmap commitment in the YC app. Don't get pulled into these during the sprint.
