# SharkAuth Launch Sprint — Handoff

**Created:** 2026-04-24 ~end-of-Day-0
**Branch:** `main` (HEAD `c860d81`)
**Session ended at:** Day 0 of 10-day launch sprint
**Next milestone:** Day 1 (2026-04-25) — binary release pipeline

This document is the cross-session pickup point. Read it first when resuming work on the launch sprint.

---

## Where we are

**Day 0 of 10. Launch target: Tuesday 2026-04-29. YC submission target: 2026-05-04.**

The 10-day calendar lives in `gstack/05-revisions-cold-dm-kit.md`. External constraint: launch by 2026-04-29 to fit other accelerator deadline. Compresses calendar to 5 days of build before launch + 5 days of post-launch polish before YC submit.

Today's hands-on work that already shipped to disk + git:

| Done today | Status |
|---|---|
| LICENSE (MIT) | Committed `d7fcd35`, pushed |
| SCALE.md (in-memory store inventory) | Committed `d7fcd35`, pushed |
| Repo-root cleanup (moved 25 internal docs + 22 PNGs into `docs/internal/` and `docs/images/`) | Committed `0fa405c`, pushed |
| Atomic refresh-token rotation patch (closes concurrent-refresh race) | Committed `0fa405c` (bundled), pushed |
| `gstack/` launch artifacts (design doc + YC app + playbook + autoplan synthesis + cold-DM kit + this handoff) | Committed `c038421` + `c860d81`, pushed |
| `documentation/api_reference/` — OpenAPI 3.1 spec, 219 operations, 5 surface files + master + README | Committed `b619295`, pushed |
| `documentation/inner_docs/` — 319 per-file MDs covering 100% of cmd/, internal/, admin/src/, packages/, sdk/typescript/, sdk/python/ + ARCHITECTURE.md | Committed `b619295` + `c860d81`, pushed |

Working tree clean. All work pushed to `origin/main`.

---

## Locked decisions (do not re-litigate)

These were made via 4-reviewer autoplan pipeline. Reasoning lives in `gstack/04-autoplan-synthesis.md` and `~/.claude/projects/-home-raul-Desktop-projects-shark/memory/launch_decisions_2026_04_24.md`.

| ID | Decision | Why |
|----|---|---|
| **TD1** | Hybrid wedge: lead with RFC 8693 agent delegation in YC title + HN post + demo. Keep broader feature surface (SSO, passkeys, MFA, proxy, orgs) in README + landing. | Sharpest differentiation in conversion surfaces; broader surface aids mid-market discovery. |
| **TD2** | Launch Tuesday 2026-04-29 (Day 5). | Hard external constraint — other accelerator deadline. |
| **TD3** | Defer pricing in YC app. Say "OSS free + hosted pricing finalized during batch." | Avoids price-taker trap; keeps optionality. |
| **TD4** | Personal target 3 design partners; YC app reports actual numbers landed. | Aspirational drives velocity; honest framing in app. |
| **TD5** | Ship as `v0.9.0` on launch day. `v1.0.0` only after 2 weeks clean API. | Admin API reshaped once in 6 months; SDK types hand-written; SemVer freeze premature. |

---

## Tomorrow (Day 1 — 2026-04-25, Friday)

**T1 — Binary release pipeline (the day's main deliverable):**
1. Write `.goreleaser.yml`. Targets: darwin-amd64, darwin-arm64, linux-amd64, linux-arm64, windows-amd64. SBOM + cosign signing.
2. Write `.github/workflows/release.yml`. On `v*` tag → goreleaser builds + uploads release artifacts.
3. Write `.github/workflows/ghcr.yml`. On `v*` tag → buildx multi-arch → push `ghcr.io/shark-auth/sharkauth:<tag>`.
4. Write `.github/workflows/publish-python-sdk.yml`. On `py-sdk-v*` tag → twine upload to PyPI.
5. Tag `v0.9.0-rc.0` as dry run. Verify all three artifacts (binaries on Releases page, image on GHCR, package on PyPI) land cleanly.

**T2 — Demand sprint (parallel):**
- Send 10 more cold DMs (template in `gstack/05-revisions-cold-dm-kit.md`). 20 sent total by EOD.
- Schedule any partner calls returned from Day 0 DMs into Day 2 calendar slots.
- Draft sharkauth.com landing page copy (hero + feature matrix + install one-liner + waitlist signup).

**Day 1 exit criteria:** binaries publish on tag, GHCR image publishes on tag, PyPI workflow exists, 20+ cold DMs out, landing copy drafted.

---

## Outstanding work that did NOT ship today

These are flagged-but-not-done items. None are launch-blockers but most are launch-week cleanup:

### Code fixes (must ship before v0.9.0)

1. **`SharkProvider.tsx:86` stray closing paren** (caught by React-SDK doc agent). Likely tsc compile error. Run `cd packages/shark-auth-react && pnpm tsc --noEmit` to confirm. Estimated fix: 2 min.
2. **SDK naming unification.** Three different scopes today: `@sharkauth/node` (TS SDK), `@sharkauth/react` (React SDK), `shark_auth` (Python). Pick one (`@sharkauth/*` recommended), rename, republish. Estimated: 1 hour including republish workflow updates.
3. **PyPI publish automation missing.** `pyproject.toml` exists at `sdk/python/pyproject.toml` but no `publish-python-sdk.yml` workflow. Lands as part of Day 1 release-pipeline work above.
4. **OpenAPI spec — depth on platform/admin sections.** OAuth + auth sections were built by full subagents with rich schemas. Platform + admin sections were built in main thread under time pressure — coverage is complete (operationIds + summaries + x-source citations) but examples are sparse. Worth one parallel-agent pass post-launch to deepen the schemas + add realistic examples.
5. **`shark doctor` CLI command.** Diagnostic for first-boot failures (config, DB writability, JWKS keys, port bind, base_url reachability, admin key presence). Day 2 of original calendar; estimated 3-4 hours.

### Hands-on (only the user can do)

1. **Buy domain** if not yet owned. `sharkauth.com` is the working assumption.
2. **Cold DMs** — 10 today (Day 0). Template in `gstack/05-revisions-cold-dm-kit.md` § Cold-DM Kit.
3. **Co-founder matching post** — one paragraph on YC matching board if not already done.
4. **Mentor + peer review request** — line up 1 mentor + 1 peer founder for Day 4 YC-app feedback pass.

---

## Critical files (next-session bookmarks)

```
LICENSE                             MIT, 2026 Raul R. Gonzalez
SCALE.md                            4 in-memory stores documented + Q3 2026 roadmap

gstack/                             session artifacts (read in order 01→07)
  README.md                         index
  01-launch-design.md               design doc + 54-capability audit matrix
  02-yc-application.md              YC P26 application draft (all form fields)
  03-launch-playbook.md             original 10-day calendar (superseded by 05)
  04-autoplan-synthesis.md          4-reviewer findings (CEO + eng + DX + adversarial)
  05-revisions-cold-dm-kit.md       OPERATIVE doc — compressed calendar, YC patches, DM kit
  06-refresh-rotation-patch.md      atomic-rotation patch writeup (already shipped)
  07-handoff.md                     this file

documentation/
  api_reference/                    OpenAPI 3.1 spec — 219 operations across 5 surfaces
    openapi.yaml                    master (info, security schemes, tag groups)
    sections/                       per-surface files (oauth, auth, platform, admin, system)
    README.md                       viewing + regen guide
  inner_docs/                       319 per-file MDs mirroring code tree
    ARCHITECTURE.md                 high-level overview + data flows + scale boundaries
    cmd/                            25 MDs
    internal/                       150 MDs across 19 packages
    admin/src/                      82 MDs (React SPA)
    packages/                       28 MDs (@sharkauth/react)
    sdk/typescript/                 18 MDs (@sharkauth/node)
    sdk/python/                     16 MDs (shark-auth)

internal/oauth/store.go             RotateRefreshToken — atomic rotation shipped 2026-04-24
internal/storage/oauth_sqlite.go:191  RevokeActiveOAuthTokenByRequestID — new atomic UPDATE
```

---

## Verification commands

Before resuming work, sanity-check the repo state:

```bash
# Working tree clean?
git status

# On main, up to date with origin?
git log --oneline -5
git diff origin/main..main          # should be empty

# Build still clean?
go build ./...

# Tests still green?
go test ./internal/oauth/... ./internal/storage/... -count=1

# OpenAPI files parse?
for f in documentation/api_reference/openapi.yaml documentation/api_reference/sections/*.yaml; do
  python3 -c "import yaml; yaml.safe_load(open('$f'))" && echo "ok $f"
done
```

All five should pass with no errors.

---

## Risks (ordered by probability of biting in the next 90 days)

1. **Zero-to-one design partners by Day 10 (60%)** — cold-DM math says 30 DMs ≈ 1 partner; need 90+ DMs to realistically hit 3. Mitigation: warm intros (Insforge founder, anyone in the LinkedIn comment thread) beat cold by 10×. Pivot path: Approach C (OSS-flagship narrative — star count + install count via telemetry as the YC traction signal).
2. **HN launch flops on Day 5 (35%)** — competitive density, audience timing. Mitigation: Sunday or Day 6 fallback to Product Hunt + r/selfhosted; YC app traction does not depend solely on HN.
3. **Anthropic ships MCP-spec auth primitives (25%)** — wedge compresses. Mitigation: positioning as reference open implementation (nginx-to-HTTP analogy) rather than "platform-neutral always wins."
4. **Solo-founder burnout (15%)** — Day 5 launch + Day 10 submit + interview prep + partner calls with no co-founder. Mitigation: cut optional items (Homebrew, Windows binary, RFC 9728) ruthlessly.

Running themes: every risk above is mitigated by either landing real partners or pivoting the framing — not by adding code.

---

## Recent commits (this session)

```
c860d81 docs(inner_docs): close coverage gaps + add SDK documentation
b619295 docs: add OpenAPI 3.1 reference + per-file inner_docs + architecture overview
c038421 docs(gstack): add launch session artifacts
0fa405c chore: repo-root cleanup — move internal docs and screenshots
d7fcd35 docs: add LICENSE (MIT) and SCALE.md
```

All pushed to `origin/main`.

---

## Memory pointers

The user-home auto-memory at `~/.claude/projects/-home-raul-Desktop-projects-shark/memory/` carries cross-session state. Relevant entries:

- `MEMORY.md` — index, scan first
- `launch_decisions_2026_04_24.md` — TD1-TD5 lock + reasoning
- `proxy_v1_5_complete.md` — pre-sprint codebase state baseline (HEAD `7856e0f`)
- `user_raul.md` — founder profile (solo, 18, Monterrey, Go-fluent, building Auth0/Clerk competitor)

When resuming, check the launch decisions memory first to confirm the TD1-TD5 set is still current, then re-read this handoff.

---

## Next-session opening prompt suggestion

If you want a clean cold-start, paste this into the next session:

> Resuming SharkAuth launch sprint Day 1 (2026-04-25). Read gstack/07-handoff.md for context. Today's deliverable is the binary release pipeline (.goreleaser.yml + release workflow + GHCR + PyPI). Locked decisions in launch_decisions_2026_04_24.md memory entry. Confirm working tree clean, then start with .goreleaser.yml for darwin-amd64/arm64 + linux-amd64/arm64 + windows-amd64 with SBOM + cosign signing.
