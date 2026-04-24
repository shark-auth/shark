# Autoplan Synthesis — SharkAuth Launch Design Review

Generated 2026-04-24 by /office-hours autoplan pipeline.
Inputs: launch design doc + YC app + launch playbook in same directory.
Review agents: CEO/founder, eng-manager, DX-critic, adversarial cold-read.

---

## Final Verdict

**APPROVE WITH HARD FIXES + 5 TASTE DECISIONS.**

The plan is structurally sound and the codebase is genuinely ahead of the pitch. Four reviewers converge on the same shape: *honest artifact, shippable tech, weak narrative, aspirational calendar.* None of the four recommended RETHINK. All four flagged specific bugs in the plan that compound if left alone.

Critical path: **close the hard fixes before Day 3 or the HN launch tanks the YC application.** Taste decisions below need founder call before plan execution starts.

---

## Converging Signals (all 4 reviewers agreed)

1. **Wedge is still too wide.** "MCP-era OSS auth" is four features dressed as one. Reviewers independently pointed to RFC 8693 agent delegation as the true wedge — the one thing no competitor ships well. CEO lens: reframe YC title as "Stripe for agent delegation." Codex: only real durable wedge is *cost-of-disagreeing-with-the-vendor on agent semantics,* not deployment model.

2. **"3 design partners in 10 days" is aspirational.** CEO says 1-2 realistic. Codex says even if 3 land, probability most are hobbyists without budget is high. Plan should carry realistic number as the committed goal (1 quote-able partner + 5 telemetry installs) and treat 3 as the stretch.

3. **"No in-memory auth state" claim is false.** Eng found 4 process-local stores (DPoP replay cache, device rate map, per-IP rate limiter, per-API-key rate limiter). Design doc explicitly claims "stateless-safe." Either fix the gaps or strike the claim before launch.

4. **Narrative undersells the product.** Codex: "you undersell — pitch claims 3 things, codebase delivers ~40." CEO: "feature-matrix appendix is the killer artifact — include in YC app." DX: "agent-first quickstart is the link that should be in the HN post, not the README."

5. **v1.0 on Day 6 is premature.** Eng evidence: admin proxy API reshaped once in 6 months (v1 → v1.5). TS SDK has hand-written types that can drift from Go handlers. Recommendation: ship as `v0.9.0`, bump to v1.0 after 2 weeks of partner usage without breaking changes.

---

## Hard Fixes (not taste — just do, before Day 3)

Blocking issues. Every one is a concrete file or command.

### DX-blocking

| # | Fix | Evidence | Effort |
|---|---|---|---|
| 1 | Create `LICENSE` file at repo root (MIT text). README claims MIT but file doesn't exist. | `README.md:906` references file that does not exist | 2 min |
| 2 | Clean repo root. Move logs, dbs, internal docs, screenshots out of root. | 40+ `.md` files + `.log` + `.db` + `.jsonl` + 12 PNGs loose at root | 30 min |
| 3 | Unify SDK naming. Currently 3 conventions (`@sharkauth`, `@shark-auth`, `shark_auth`). | `sdk/typescript/README.md:1` vs `packages/shark-auth-react/package.json` vs `docs/hello-agent.md:117` | 1 hr |
| 4 | Ship binaries. README markets "$5 VPS deploy" but only install path requires Go compiler. Either goreleaser + `curl \| sh` or change README to honest "build from source for now." | `README.md:23`, `docs/hello-agent.md:30-38` | 4-8 hr (goreleaser) |

### Eng-blocking (security / correctness)

| # | Fix | Evidence | Effort |
|---|---|---|---|
| 5 | Atomic refresh-token rotation. Currently `GetActive → Revoke` is not transactional. Under load, two refreshes can race. | `internal/oauth/store.go:258-265` | 20 min (single `UPDATE ... WHERE revoked_at IS NULL RETURNING id`) |
| 6 | Strike "stateless-safe" claim OR move 4 in-memory stores to SQLite-backed tables. | `internal/oauth/dpop.go:48-78`, `internal/oauth/device.go:33-55`, `internal/api/middleware/ratelimit.go:46-84`, `internal/auth/apikey.go:121-162` | 30 min (strike) or 4 hr (fix) |
| 7 | Add `SCALE.md` at repo root naming the 4 in-memory stores + Q3 2026 Postgres + shared-cache roadmap. Pre-empts the HN comment "this isn't actually stateless." | Derived from fix #6 | 30 min |
| 8 | Document signing-key-at-rest model in security doc. AES-GCM'd with env secret, no KMS. Acceptable for OSS self-host if documented; security risk if oversold. | `internal/oauth/server.go:47,129-233` | 30 min |

### DX-high-leverage (do before Day 6)

| # | Fix | Evidence | Effort |
|---|---|---|---|
| 9 | Ship `shark doctor` diagnostic command. Checks config, DB writability, JWKS keys, port bind, base_url reachability, admin key presence. | `cmd/shark/cmd/root.go:90-97` registers 27 subcommands, no diagnostics | 3-4 hr |

### Plan-structural

| # | Fix | Evidence | Effort |
|---|---|---|---|
| 10 | Day 1 calendar is actually 2-3 days of work (goreleaser + release workflow + GHCR + PyPI). Expand to Day 1-2. OpenAPI spec moves to Day 3 with reduced scope (30 endpoints instead of 200). | Existing `.github/workflows/` has only 3 workflows; 200 route handlers in router.go | Calendar reshuffle, 0 min |
| 11 | Ship as `v0.9.0` on Day 6, not `v1.0.0`. YC judges product, not version string. | Admin API reshape history + unstable SDK types | 0 min (decision only) |
| 12 | Rewrite YC app "How many users?" section on Day 10 in past tense with real numbers, not targets. | YC app line 41 | Same-day update |
| 13 | Promote audit matrix to a linked appendix in YC app. | Design doc lines 66-160 | 15 min |

---

## Taste Decisions (founder must choose)

These are trade-offs where reviewers disagreed with the plan or with each other. Founder call.

### TD1 — Wedge framing

**Current plan:** "MCP-native agent auth — OSS self-host with OAuth 2.1 + agent delegation + proxy mode + all identity primitives."
**Reviewer alternative (CEO + Codex converge):** "Stripe for agent delegation. Add RFC 8693 delegation to your MCP server in 5 minutes. Everything else is included."

Options:
- A) Keep current wedge. Broader value prop. Risk: reads as "a better Auth0," unwinnable for solo 18yo.
- B) Narrow to pure delegation. Sharpest differentiation. Risk: audience is smaller; delegation-specifically is still new demand.
- C) Hybrid: Lead with delegation in YC title + HN post + demo; keep broader value prop in README + landing page.

### TD2 — HN launch day

**Current plan:** Wednesday 2026-05-01, 7am PT.
**Reviewer alternative (Codex):** Sunday 2026-05-04, 10am PT. Lower competing-launch density, audience skews indie/self-hoster, front page ride lasts into Monday.

Options:
- A) Keep Wednesday. Known high-traffic window. Recovery room for Day 8-9.
- B) Move to Sunday. Counterintuitive bet. Higher ICP match, but Day 10 = YC submission day — can't launch same day.
- C) Move to Day 5 (Tuesday). Gives 5-day recovery window before YC submit.

### TD3 — Pricing story for YC app

**Current plan:** "$29/mo hosted vs Auth0 $53+." Per-seat / per-MAU implied.
**Reviewer alternative (Codex):** "$0 self-host forever + $500/mo flat up to 1M agents." Flat pricing is the real wedge against per-MAU SaaS.

Options:
- A) Keep $29 per-MAU framing. Familiar to SaaS buyers.
- B) Flat $500/mo. Price-maker posture, harder to commoditize.
- C) Defer pricing commitment. YC app says "OSS free, hosted pricing finalized during batch."

### TD4 — Partner goal commitment

**Current plan:** 3 named design partners by Day 10.
**Reviewer alternative (CEO):** 1 quote-able partner + 5 telemetry-pinged installs.

Options:
- A) Keep 3-partner commitment. Aspirational target motivates cold-DM velocity.
- B) Lower commitment to 1 quote + 5 installs. Realistic, sets up honest YC app section.
- C) Track both. Public commitment (to self) is 3; YC app reports whatever actually shipped.

### TD5 — Version on launch day

**Current plan:** `v1.0.0` tagged Day 6.
**Reviewer alternative (Eng):** `v0.9.0` tagged Day 6, `v1.0.0` after 2 weeks of partner usage without breaking changes.

Options:
- A) Ship as v1.0.0. Marketing narrative is "version 1 is out."
- B) Ship as v0.9.0. Retains SemVer flexibility, honest about API maturity.
- C) Ship as v0.9.0 on Day 6, promote to v1.0.0 *during* YC interview prep if API has been stable 10 days.

---

## What To Amplify (reviewers agreed on strengths)

Bake these into every customer touchpoint.

1. **Audit matrix as appendix.** 38/54 capabilities with file-level evidence is a moat against pattern-match rejection. No other solo pre-launch applicant has this.
2. **Error catalog (docs/errors.md).** RFC 6749 §5.2 split + extended envelope = category-pro signaling. HN commenters will grep for this.
3. **`shark serve --dev` zero-config moment.** Single command, admin key to stdout, dev inbox. Genuinely tweet-worthy. Lead the HN post video with this.
4. **"Competition is the base of businesses" operator instinct.** Keep this voice in YC app answers.
5. **Honest-framing discipline.** PARTIAL-means-PARTIAL + waitlist-is-not-demand throughout. Reads as high-integrity to partners.
6. **Agent delegation demo as the differentiator.** Lead every customer touchpoint with the delegation-chain visualization. Codex's sandbox idea (sharkauth.com/try with live delegation tree) is a 48-hr stretch goal worth considering.

---

## Failure Modes (ranked by probability, reviewer-consensus)

1. **Zero-to-one design partners by Day 10 (60% probability per CEO lens).** Cold-DM success rate in 10 days is low. Mitigation: Approach C pivot to OSS-flagship narrative; use star count + install count + waitlist as traction signal.
2. **HN launch flops (35% probability — timing + competitive density).** Mitigation: launch earlier (Day 5 Tuesday) for recovery room; don't pin YC app traction to HN outcome.
3. **MCP category compresses (25% probability).** Anthropic ships auth primitives into MCP spec itself. Mitigation: stronger positioning as *reference open implementation* (like nginx to HTTP) rather than "platform-neutral always wins."
4. **Horizontal-scale gap exposed at launch (30% probability if fixes #6/#7 skipped).** Top HN comment becomes "this isn't actually stateless." Mitigation: ship SCALE.md Day 2.
5. **v1.0 SemVer trap (20% probability — plays out weeks later).** Partners on v1.0 hit breaking change, public loss of trust. Mitigation: ship as v0.9.x.
6. **Solo-founder burnout between Day 7 launch + Day 10 submission (15%).** 16-hour HN day + interview prep + partner calls with no co-founder. Mitigation: realistic calendar, cut optional items (Homebrew tap, Windows binary) ruthlessly.

---

## Next Steps (in order)

1. Founder makes taste decisions TD1-TD5 (15 min).
2. Founder commits to hard fixes #1-#13 as Day 0-1 work (replaces current Day 0 scope).
3. Revise design doc + YC app + launch playbook to absorb taste decisions + fixes (1 hour).
4. Execute revised Day 0 today: hard fixes #1-#3 (LICENSE, repo cleanup, SDK naming) before EOD.
5. Optional: before Day 7 launch, run one more narrow review on the *revised* YC app (post-taste-decisions). One codex pass, not full autoplan.

---

## Autoplan Pipeline Stats

- Reviewers: 4 (CEO/founder, eng-manager, DX-critic, adversarial cold-read).
- Subagent runtime: ~2 min (parallel).
- Evidence cited: 17 specific file:line references + 3 workflow files + 1 router with 200 handlers.
- Hard fixes identified: 13.
- Taste decisions requiring founder call: 5.
- Convergent signals (≥3 reviewers agreed): 5.
- Divergent signals (1 reviewer dissented): 4 (wedge narrowness, HN day, pricing, version).

End of synthesis.
