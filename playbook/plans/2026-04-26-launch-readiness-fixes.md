# Launch Readiness Fixes — 2026-04-26

**Goal:** clear all P0 blockers + key P1s identified by 4-axis re-audit so SharkAuth ships Monday for YC + OSS launch without hostile-reviewer bounce.

**Wedge framing reminder:** "auth for products that give customers their own agents." Audience = teams shipping full-stack apps where their own users get agents (NOT MCP server builders directly — they're a tertiary audience). Five-layer revocation moat is the lead.

**Scope cuts confirmed by founder:**
- PyPI publish deferred — dogfood Python SDK first, ship to PyPI W+1 after internal validation
- README is rewrite, not surgical edit — aim for YC + OSS traction, demos as `**demo: X**` placeholders
- **F1 setup-token DB persistence DEFERRED** — accept restart-during-setup risk for Monday launch, schedule W+1 (still 5-min window in practice for hostile-reviewer demo)

**Founder decisions on plan questions:**
1. F1: deferred (above)
2. F3 demo trigger: implementer's call — pick whichever ships cleanest
3. F7 human nodes: initials-only avatar (monochrome circle with 1-2 letters)
4. F9 README: keep current ASCII logo
5. Demo placeholders: `**demo: X**` markdown bold

**MANDATORY for every fix that adds new logic:** ship pytest smoke coverage in `tests/smoke/`. New behavior without a corresponding `test_*.py` will be rejected at merge. Per-fix test requirements listed below; subagents must include impl + test in same diff (per HANDOFF D4).

**STANDING RULE — opportunistic button wiring:** while editing any frontend surface, if subagent encounters an unwired button (`onClick={() => {}}`, no `onClick`, dead handler, TODO marker), wire it to its obvious destination. Examples: list-row "view details" → open drawer for that row; "create" button → open empty create drawer; "export" → trigger backend export endpoint. Do NOT scope-creep into rebuilding entire features — only wire buttons whose intent is unambiguous from surrounding context. Report wired buttons in the diff summary.

**State at plan creation:**
- Backend audit pipeline: GREEN (verified)
- Frontend monochrome surfaces: GREEN (verified, 3 small bugs)
- Onboarding UX: 3 P0s remain
- DX distribution: shark doctor missing, docs gap, README weak
- Delegation chains: canvas wired correctly, but RFC 8693 nested act-claim never flattened to flat array → empty render

---

## Critical Path (sequential dependencies)

```
[F2 login hint]    ───┐
[F3 get-started]   ───┤
[F4 serve banner]  ───┤──→ smoke green ──→ Monday ship
[F5 shark doctor]  ───┤
[F6 delegation fix]───┤
[F7 chain canvas /impeccable]  ─ depends on F6 (data must work first)
[F8 quickstart doc reframe]  ─ independent
[F9 README YC/OSS rewrite] ───┤
[F10 Scalar+OpenAPI+stale docs]  ─ independent
```

All fixes touch DIFFERENT files except F6+F7 (both touch `delegation_chains.tsx` — keep sequential). Otherwise full parallelization possible.

---

## F1 — Setup Token DB Persistence — DEFERRED to W+1

**Status:** founder accepts restart-during-setup risk for Monday launch. In-memory `setupState` package var stays as-is. Document the limitation in `POST_LAUNCH_BUGS.md` with a one-line note: "first-boot setup token lost on process restart; manually delete `data/sharkauth.db` to re-trigger." Schedule full DB-backed fix W+1.

**No work this round.**

---

## F2 — Login Page Hint Default-Open + Recovery Context (P0, ~30min)

**Problem:** `admin/src/components/login.tsx` has `hintOpen` state defaulting to `false`. First-time user sees blank `sk_live_…` input with zero context.

**Fix shape:**
1. Default `hintOpen=true` when `localStorage.getItem('shark.admin.lastLogin')` is null (first visit)
2. Hint text expanded to: `Find your admin key in the terminal output where you ran shark serve, or read it from data/admin.key.firstboot. Run shark admin-key show to print it again.`
3. Error message on invalid key: append `Lost your key? Run shark admin-key regenerate to get a new one.` (depends on F1 shipping the regenerate command)
4. After successful login, set `localStorage.setItem('shark.admin.lastLogin', Date.now())` so hint stays collapsed on return visits

**Files:** `admin/src/components/login.tsx` only.

**Subagent dispatch:** sonnet (no worktree needed — single file edit, can run in main).

---

## F3 — Get Started Rewrite: Drop Install, Focus First Agent (P0, ~2h)

**Problem:** `admin/src/components/get_started.tsx:39` shows `curl -fsSL https://sharkauth.dev/install.sh | sh` — hallucinated URL. Also irrelevant: user is already running shark to view this page.

**Fix shape:**
1. **Delete** Section 1 (install task) entirely. User reading this is already past install.
2. **New Section 1: Create your first human user**
   - CTA button: navigate to `/admin/users` open-create-drawer state
   - CLI mirror: `shark user create --email demo@example.com`
   - Auto-checkoff: poll `GET /api/v1/users?count=1`, mark done when ≥1 user exists
3. **New Section 2: Register your first agent**
   - CTA button: navigate to `/admin/agents` open-create-drawer
   - CLI mirror: `shark agent register --name my-first-agent`
   - Auto-checkoff: poll `GET /api/v1/agents?count=1`
4. **New Section 3: See the wedge — run a delegation chain**
   - Paragraph: "shark agents can act-on-behalf-of other agents (RFC 8693 token exchange). Watch a 3-hop chain in 8 seconds."
   - Single button: `Run delegation demo` → POSTs to a new endpoint `POST /admin/demo/delegation-chain` (or shells out via existing `shark demo delegation-with-trace --base-url http://localhost:8080`)
   - On success: link to `/admin/delegation-chains` to see the resulting canvas (depends on F6+F7)
5. **New Section 4: Ship to your stack**
   - Two cards: "Python SDK (dogfood mode)" with git-install snippet + caveat, "TypeScript SDK" coming-soon badge
   - Link to `docs/mcp-5-min.md` (renamed per F8) as reference doc
6. Remove all `pip install shark-auth` references, replace with `pip install git+https://github.com/<repo>/shark#subdirectory=sdk/python` + `# PyPI release coming after dogfood validation`

**Files:** `admin/src/components/get_started.tsx` (rewrite tasks array + auto-check probes)

**Subagent dispatch:** sonnet, no worktree (single file). Use /impeccable craft adherence — match existing get-started DNA.

---

## F4 — `shark serve` Re-Run Banner (P1, ~30min)

**Problem:** Second `shark serve` on same data dir prints generic startup, no signal that admin is already configured.

**Fix shape (no F1 dependency):**
1. In `internal/server/firstboot.go` after boot completes, check `admin_users` table count
2. If count > 0: print `✓ admin configured · Dashboard: http://localhost:8080/admin · Sign in with your admin key`
3. If count == 0 AND `data/admin.key.firstboot` file exists on disk: print `⚠ setup pending · Dashboard: http://localhost:8080/admin · Key in: data/admin.key.firstboot`
4. If count == 0 AND no key file: trigger first-boot flow (write key, print full first-boot banner)

**Files:** `internal/server/firstboot.go`, small helper in `internal/cli/branding.go`

**Subagent dispatch:** sonnet, no worktree. Independent — schedule in Wave A.

---

## F5 — `shark doctor` Implementation (P0, ~3h)

**Problem:** README + playbook reference `shark doctor`, file does not exist.

**Fix shape:** new `cmd/shark/cmd/doctor.go` with checks:
1. **config:** load DB-backed config, report version + last-modified
2. **db writability:** open DB, attempt write to a transient test row, report path + size
3. **migrations:** report applied count + schema version, flag pending
4. **jwt keys:** count active signing keys, report alg + age, warn if all keys >90 days
5. **port bind:** attempt to bind configured port, on EADDRINUSE report PID via `lsof`/`netstat` (cross-platform best-effort)
6. **base_url reachability:** HTTP GET `<base_url>/api/v1/health` from within process, report 200/error
7. **admin key:** check `data/admin.key.firstboot` exists OR `admin_users` count > 0, exit non-zero if neither
8. **smtp (optional):** if SMTP configured, do TLS handshake against host (no send)
9. **vault:** check encryption key derivation works (decrypt+re-encrypt a test value)

**Output format:** match Wave 1.8 polished CLI: `→` while running, `✓ check_name    detail` on pass, `✗ check_name    actionable error` on fail. Final line: `N/M checks passed`.

**Exit codes:** `0` all pass, `1` any fail, `2` config not loadable.

**Files:** new `cmd/shark/cmd/doctor.go`, register in `cmd/shark/cmd/root.go`, helpers possibly in new `internal/diag/diag.go`

**Subagent dispatch:** sonnet, isolation:"worktree" (touches root.go which other waves may also touch). Add 5 smoke tests in `tests/smoke/test_doctor_command.py`.

---

## F6 — Delegation Chains: flattenActClaim Helper (P0, ~1h)

**Problem (per delegation chains research subagent):** Backend `internal/oauth/exchange.go:335` `buildActClaim` correctly emits RFC 8693 nested map: `{"sub": "a2", "act": {"sub": "a1"}}`. Frontend `delegation_chains.tsx:74-77` checks `Array.isArray(meta.act_chain)` — false for nested object — so chain renders as empty array.

**Fix shape:**
1. Add `flattenActClaim` helper at top of `delegation_chains.tsx`:
   ```ts
   function flattenActClaim(nested: any): Array<{sub: string; jkt?: string; label?: string}> {
     const hops: Array<{sub: string; jkt?: string}> = [];
     let cur = nested;
     while (cur && typeof cur === 'object' && cur.sub) {
       hops.push({ sub: cur.sub, jkt: cur.jkt });
       cur = cur.act;
     }
     return hops;
   }
   ```
2. In `normalizeEntry`, replace bare `Array.isArray(meta.act_chain) ? meta.act_chain : []` with three-way check: array → use as-is; object with `.sub` → flatten; else → empty
3. Apply identical helper in `admin/src/components/agents_manage.tsx:1543`
4. Add smoke test `tests/smoke/test_delegation_chain_render.py` — runs `python tools/agent_demo_tester.py`, then GETs audit log, asserts at least one event has chain length ≥ 2 after flatten

**Files:** `admin/src/components/delegation_chains.tsx`, `admin/src/components/agents_manage.tsx`, new smoke test

**Subagent dispatch:** sonnet, no worktree (small surgical edit).

---

## F7 — Delegation Chains Canvas /impeccable Pass (P1, ~2-3h)

**Depends on F6 shipping first** (canvas needs real data flowing).

**Problem:** Even with data, canvas nodes look basic, edges don't visually convey act-on-behalf-of relationship clearly. User wants: pick a past chain → canvas renders as auditable history with pretty nodes + meaningful wires.

**Fix shape:**
1. Use `/impeccable craft` skill on `admin/src/components/delegation_canvas.tsx` (or wherever react-flow node types live). Anchor on `users.tsx` DNA (gold standard).
2. **Node design** (3 types):
   - **Human node:** square 64×64, hairline border, subtle avatar glyph + name label below, monochrome
   - **Agent node:** square 64×64, hairline border, agent name in mono font, small `act-as` count badge bottom-right when chain length > 1
   - **Service node** (optional): hexagon for resource server / audience target
3. **Edge design:** directional arrow `subject → actor`, label edge with `via token_exchange · <timestamp>` mono caption. Active edge (most recent hop) bolder than historical hops.
4. **Layout:** left-to-right chain, oldest hop leftmost, newest rightmost. ReactFlow `dagre` or simple manual x-spacing.
5. **Interaction:** click a node → side drawer (right, 380px hairline border per DNA) shows agent details + linked audit events
6. **Header:** chain summary `3-hop chain · started 2m ago · alice@corp → research-agent → tool-agent` with `← back to chains list` link

**Files:** `admin/src/components/delegation_canvas.tsx`, `admin/src/components/delegation_chains.tsx` (header + chain selector)

**Subagent dispatch:** invoke `/impeccable` craft skill directly. Pass design context from `.impeccable.md` v3 + reference `users.tsx`. Sonnet, no worktree.

---

## F8 — `docs/mcp-5-min.md` Reframe (P1, ~1.5h)

**Problem:** Original framing "drop SharkAuth in front of MCP server in 5 min" mistargets audience. Real wedge audience = teams shipping full-stack apps where their users get agents (think: Lovable, Replit-like products, vertical AI SaaS that ships agents per customer).

**Fix shape:** rename file `docs/agent-platform-quickstart.md` (keep mcp-5-min.md as a redirect note pointing to new doc).

**New doc structure:**
1. **Who this is for** (60s): "You're building a product where each customer gets their own agents — research assistants, coding tools, customer-support bots. SharkAuth is the auth layer that makes those agents safe to ship."
2. **The 3-minute mental model:** human → your app → customer's agent → external resource. Shark sits at every boundary, issues scoped tokens, logs every act-on-behalf-of hop, lets you revoke any token at any layer.
3. **5-minute integration:**
   - Step 1: install shark, run shark serve
   - Step 2: register your app via Applications surface
   - Step 3: when customer signs up in YOUR product, call SharkAuth API to provision agent identity for them
   - Step 4: your agent runtime requests token from SharkAuth (DPoP-bound) before each external call
   - Step 5: when customer cancels, single `bulk_revoke_by_pattern` call kills all their agents' tokens
4. **The wedge: five-layer revocation** — visual diagram (use `**diagram showing 5 revocation layers**` placeholder) + one-paragraph each layer
5. **Code examples** — Python SDK calls in 3 snippets (provision, get-token, revoke)
6. **Comparison table:** SharkAuth vs Auth0 vs Clerk vs Stytch vs build-your-own — agent-native columns

**Files:** new `docs/agent-platform-quickstart.md`, redirect stub at `docs/mcp-5-min.md`

**Subagent dispatch:** sonnet (writing-heavy task, no worktree). Reference `playbook/07-yc-application-strategy.md` LOCKED 55-word for voice.

---

## F9 — README Rewrite for YC + OSS Traction (P1, ~3h)

**Problem:** Current README leads with "Auth0 charges $23/mo" framing, buries agent moat, mixes runtimes in one-liner.

**Fix shape:** full rewrite. Structure:

1. **Hero (above the fold):**
   - Logo / tagline: `SharkAuth · auth for products that give customers their own agents`
   - 1-line: `Self-hosted, single-binary, OAuth 2.1 + RFC 8693 + DPoP. Five-layer revocation built in.`
   - Three badges: license (Apache-2.0), build status, latest release
   - **demo showing the 5-second cold-start to first agent token** placeholder

2. **Why SharkAuth (60s read):** the wedge — full-stack app teams shipping agents per customer. 3 bullets on what's painful without it (vendor lock-in, no agent-native primitives, no act-on-behalf-of audit chain).

3. **Quickstart (90s):**
   ```bash
   # download binary (releases page link)
   ./shark serve
   # open http://localhost:8080/admin
   # paste admin key from terminal output
   ```
   No `pip install`, no `npm install` in quickstart.

4. **The five layers** — short table, each layer one row + one-line description:
   - L1 individual token revoke
   - L2 agent-wide revoke
   - L3 user-cascade revoke
   - L4 bulk pattern revoke (W+1)
   - L5 vault disconnect cascade (W+1)

5. **Architecture diagram** placeholder: `**diagram showing human → app → agent → resource with shark at every boundary**`

6. **Use cases** — 4 cards:
   - AI SaaS shipping agents per customer
   - Internal AI platform team (compliance audit trails)
   - MCP server developers (mention as tertiary)
   - Self-hosters wanting Auth0-replacement with agent support

7. **Roadmap (transparent):**
   - v0.1 (now): identity hub + token vault + audit + auth flow builder + delegation chains
   - v0.2 (W18-W19): proxy + auth flow builder + TS SDK + bulk-pattern revoke + vault cascade
   - v0.3 (W20+): hosted tier + Claude Code skill + MCP wrapper

8. **Community + license:** Apache-2.0, Discord link placeholder, GitHub Discussions, contribution guide link

9. **YC reviewer hook (footer):** one paragraph: "Why now: every product is becoming an agent platform. Auth was already a differentiator. Agent auth is a moat."

**Drop:** all PyPI references, all install.sh references, all marketing-flavored "Auth0 charges $23/mo" framing.

**Demo placeholders to embed:** `**demo: 8-second cold-start cast**`, `**demo: delegation chain canvas**`, `**demo: bulk revoke**`. Founder fills in later.

**Files:** `README.md` full rewrite. Keep CHANGELOG.md, LICENSE, etc untouched.

**Subagent dispatch:** sonnet (long-form writing). Reference playbook/07-yc-application-strategy.md, playbook/00-design.md, playbook/04-wave4-launch.md for voice consistency.

---

## F10 — Scalar UI + OpenAPI Spec + Stale Docs Sweep (P1, ~3-4h)

**Problem:**
- `docs/openapi.yaml` MISSING — `/api/docs` Scalar UI route exists in router but serves nothing (probably 404 or 500)
- Existing `docs/` pages have stale references: `shark init` (yaml deprecated W17), `pip install shark-auth` (PyPI not shipped), install.sh (hallucinated), wrong CLI command names per CLI/docs mismatch surfaced in DX audit
- Scalar UI not actually verified working in current build

**Fix shape:**

**Part A — OpenAPI 3.1 spec** (~2h):
1. Generate or hand-write `docs/openapi.yaml` covering ~30 endpoints scoped to launch surface (per playbook 06-references.md):
   - `/oauth/*` (authorize, token, register/DCR, jwks, userinfo, revoke, introspect, .well-known)
   - `/api/v1/users/*` (CRUD + revoke-agents)
   - `/api/v1/agents/*` (register, CRUD, rotate-secret, rotate-dpop-key, revoke-tokens, policies)
   - `/api/v1/applications/*` (CRUD, rotate-secret)
   - `/api/v1/audit-logs/*` (list, export)
   - `/api/v1/vault/*` (list, store, retrieve, rotate, disconnect)
   - `/api/v1/auth-flows/*` (list, create, update, delete, run)
   - `/admin/setup`, `/admin/bootstrap/status`
   - `/api/v1/health`, `/api/v1/me/*`
2. Document request/response schemas with examples per endpoint
3. Tag groups: `oauth`, `agents`, `users`, `applications`, `audit`, `vault`, `flows`, `admin`
4. Security schemes: `bearerAuth` (admin key), `oauth2` (DCR), `dpopBound` (custom)
5. Skip cut surfaces: `/proxy/*`, `/api/v1/policies/bulk-revoke-pattern`, vault cascade endpoints

**Part B — Scalar UI route verification** (~30min):
1. Verify `/api/docs` route in `internal/api/router.go` correctly serves Scalar HTML pointing at `/api/openapi.yaml`
2. Verify `/api/openapi.yaml` route serves the spec file (likely needs `go:embed` or filesystem read)
3. Open `http://localhost:8080/api/docs` after `shark serve`, confirm Scalar renders all endpoints + try-it-out works against running shark

**Part C — Stale docs sweep** (~1h):
1. Grep all `docs/**/*.md` for: `shark init`, `pip install shark-auth`, `sharkauth.dev/install.sh`, `sharkauth.yaml`, references to deprecated YAML config
2. Replace with current equivalents:
   - `shark init` → remove (yaml config dead per W17 — DB-backed now)
   - `pip install shark-auth` → `pip install git+https://github.com/<repo>/shark#subdirectory=sdk/python` + caveat
   - `sharkauth.dev/install.sh` → `# download binary from GitHub releases (link)`
   - `sharkauth.yaml` references → "config managed via dashboard (Settings) or `shark admin config` CLI"
3. Reconcile `docs/hello-agent.md` against what actually works after fixes (especially after F3 get-started rewrite)
4. Remove or annotate any doc page referencing cut features (proxy, bulk-pattern, vault cascade) with `**v0.2 (W18-W19)**` badge

**Files:**
- new `docs/openapi.yaml` (~30 endpoints, ~600 lines)
- `internal/api/router.go` (verify Scalar route + openapi.yaml serve)
- possibly new `internal/api/docs_handler.go` if not already exists
- sweep edits across `docs/auth.md`, `docs/errors.md`, `docs/hello-agent.md`, `docs/postman-guide.md`, `docs/guides/*.md`
- new smoke test `tests/smoke/test_openapi_scalar.py` — assert `/api/docs` 200 + contains `Scalar` keyword + `/api/openapi.yaml` 200 + valid YAML parse

**Subagent dispatch:** sonnet, isolation:"worktree" (touches router.go which other waves may also touch). Long task — give it 4h budget. Dispatch in Wave A.

---

## Parallelization Plan (revised — F1 deferred, F10 added)

**Wave A (parallel, worktrees for ones touching shared files):**
- F5 shark doctor (worktree — touches root.go, ~3h)
- F10 Scalar+OpenAPI+stale docs (worktree — touches router.go + many docs/, ~3-4h)
- F9 README rewrite (no worktree — only README.md, ~3h)
- F8 quickstart doc reframe (no worktree — only docs/, ~1.5h)
- F4 serve banner (no worktree — only firstboot.go + branding.go, ~30min)

**Wave B (parallel, after Wave A merge so smoke baseline known):**
- F2 login hint (no worktree, ~30min)
- F3 get-started rewrite (no worktree — only get_started.tsx, ~2h)
- F6 delegation flatten helper (no worktree — 2 small TS edits, ~1h)

**Wave C (sequential, after F6):**
- F7 delegation canvas /impeccable craft (~2-3h)

**Critical chain bottleneck:** F6 flatten gates F7 canvas. Otherwise wide parallel.

**Total wall-clock with parallelization:** ~5-7h (vs ~17h sequential).

**Worktree collision matrix:**
| Fix | router.go | root.go | firstboot.go | docs/ | admin/ |
|---|---|---|---|---|---|
| F2 |  |  |  |  | login.tsx |
| F3 |  |  |  |  | get_started.tsx |
| F4 |  |  | ✓ |  |  |
| F5 |  | ✓ |  |  |  |
| F6 |  |  |  |  | delegation_chains.tsx + agents_manage.tsx |
| F7 |  |  |  |  | delegation_canvas.tsx |
| F8 |  |  |  | new file |  |
| F9 |  |  |  |  |  |
| F10 | ✓ |  |  | many | possibly nav docs link |

Only F5 and F10 collide on top-level Go files → both worktreed. Frontend fixes don't overlap on same components.

---

## Mandatory Pytest Coverage Per Fix

Every fix that ships new logic MUST add a `tests/smoke/test_*.py` file. Naming convention: `test_<fix_id>_<short_desc>.py`. Subagents are NOT allowed to merge impl-only diffs.

| Fix | Test file | Coverage |
|---|---|---|
| F2 login hint | `test_f2_login_hint.py` | Default-open state on first visit, hint copy contains key file path, error message mentions regenerate |
| F3 get-started rewrite | `test_f3_get_started_first_agent.py` | Section 1/2/3/4 render, auto-checkoff probes hit correct endpoints, demo trigger 200s, install section absent |
| F4 serve banner | `test_f4_serve_rerun_banner.py` | `admin_users count > 0` → "configured" banner; key file exists + zero users → "setup pending"; clean state → first-boot |
| F5 shark doctor | `test_f5_doctor_command.py` | All 9 checks present, exit codes (0/1/2), `--json` output shape, port-in-use produces actionable error |
| F6 delegation flatten | `test_f6_delegation_chain_render.py` | Run `tools/agent_demo_tester.py`, GET audit log, assert at least one event has chain length ≥ 2 after flatten; both array + nested-object inputs handled |
| F7 canvas /impeccable | `test_f7_delegation_canvas.py` | Canvas data endpoint returns nodes+edges with correct shape; node types (human/agent/service); edge labels include timestamp |
| F8 quickstart doc | `test_f8_quickstart_doc.py` | File exists at `docs/agent-platform-quickstart.md`, contains 5-min integration section, references SDK methods that actually exist |
| F9 README rewrite | `test_f9_readme.py` | No `pip install shark-auth` (without git+), no `sharkauth.dev/install.sh`, no `Auth0 charges` framing, contains "five-layer revocation" |
| F10 OpenAPI + Scalar | `test_f10_openapi_scalar.py` | `/api/docs` 200 + Scalar HTML, `/api/openapi.yaml` 200 + parses as valid OpenAPI 3.1, all listed endpoints documented, no proxy endpoints in spec |

**Smoke target after all merges:** ≥385 PASS (current 375 + ~10 new tests).

---

## Verification After Each Wave

1. Run `./smoke.sh` from main session (NOT subagent) — must stay above current PASS baseline + new tests
2. `git diff --stat` per merged PR — verify impl + test + docs all present (HANDOFF D4)
3. Manual: `shark serve` → open dashboard → walk get-started → run delegation demo → verify canvas renders chain with flatten fix
4. Manual: `shark doctor` against fresh DB → all checks pass
5. Manual: open `/api/docs` → Scalar UI loads, try-it-out works against running shark
6. Manual: re-run `shark serve` on existing data dir → verify F4 banner shows "admin configured"

---

## Out of Scope (deferred W+1 or later)

- PyPI publish (dogfood Python SDK first, ship after internal validation)
- **F1 setup-token DB persistence** (founder accepts restart-during-setup risk for Monday)
- TypeScript SDK agent-native methods (CUT 4 in playbook/08)
- Proxy 12 known bugs (already coming-soon gated)
- Bulk-pattern revoke + vault cascade (CUT 2)
- Harness self-register skill + MCP wrapper (CUT 7, W19+)
- 4 fixture-only smoke failures (deferred per founder directive)
- Audit `err.Error()` raw leaks in admin_system_handlers (P1 from backend audit — schedule W+1)

---

## Founder Decisions — RESOLVED

1. **F1 setup token:** DEFERRED to W+1
2. **F3 get-started demo trigger:** implementer's call — pick whichever ships cleanest
3. **F7 human nodes:** initials-only avatar (monochrome circle, 1-2 letters)
4. **F9 README ASCII logo:** KEEP current
5. **Demo placeholders format:** `**demo: X**` markdown bold

Ready to dispatch Wave A.
