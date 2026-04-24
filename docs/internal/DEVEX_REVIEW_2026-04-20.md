# SharkAuth DevEx Review — 2026-04-20

**Method:** 8 parallel read-only Explore agents ran devex-review methodology across heterogeneous components. Source + live (gstack browse) + competitor benchmarks. Persona-first.

**Overall:** 6.2/10 weighted average. Mechanics solid, DX/polish/docs weak. **5 pre-launch blockers identified.**

---

## Master Scorecard

| # | Component | Persona | Overall | Worst dim | Best dim |
|---|-----------|---------|---------|-----------|----------|
| C1 | First-run onboarding | Self-hoster (YC) | **7.5/10** | Recovery hints (6) | Magical moment (9) |
| C2 | Admin dashboard | Admin operator | **6.3/10** | DX measurement (3) | Dev environment (8) |
| C3 | End-user auth | Real human | **7.0/10** | Documentation (5) | Server-rendered UX (9) |
| C4 | OAuth 2.1 server | App integrator | **7.2/10** | Documentation (4) | Safety defaults (9) |
| C5 | **Agent auth (moat)** | YC agent builder | **4.2/10 DX** / 9/10 code | Doc accuracy (2), Showcase (1), SDK (3) | Spec coverage (9) |
| C6 | Reverse proxy | Self-hoster wrapping | **7.1/10** | Transparent port W15 (2) | Circuit breaker UX (9) |
| C7 | Auth flows builder | Power admin | **5.4/10** | Save-test loop (4) | Debug visibility (7) |
| C8 | CLI | CI/power user | **5.0/10** | Scripting/JSON (2) | Completion (7) |

---

## Pre-launch critical (block ship)

| Rank | Finding | Component | File:line | Fix size |
|------|---------|-----------|-----------|----------|
| **P0.1** | `AGENT_AUTH.md:4` claims "not yet implemented" but code ships 95% | C5 | `AGENT_AUTH.md:4-6,28,607+` | 2h doc rewrite |
| **P0.2** | Flow builder conditional `then/else` branches silently lost on save | C7 | `flow_handlers.go:331` + `flow_builder.tsx:293-299` response deserialize | 2h |
| **P0.3** | Orgs "New" button has no onClick | C2 | `organizations.tsx:57` | 1h (existing W01) |
| **P0.4** | W15 multi-listener not shipped → single-port architecture breaks "one binary" pitch | C6 | new `internal/proxy/listener.go` + config shape | 4-6h (existing W15) |
| **P0.5** | CLI zero `--json` flags → blocks CI/CD/Terraform/Pulumi | C8 | all `cmd/shark/cmd/*.go` | 3-5d |

---

## Per-component synthesis

### C1 — First-run onboarding (7.5/10)

**TTHW:** ~4min. Clerk 2min, Supabase 3min, Auth0 15min. Shark 2nd place achievable.

**Friction log (10):**
1. Post-init msg "Do not use for production" dents confidence on first output (`cmd/shark/cmd/init.go:80`)
2. Two competing auth paths printed (raw key + bootstrap URL) — user must pick, 50% pick wrong one (`internal/server/server.go:231-242`)
3. Bootstrap token 10min silent expiry, no recovery hint on login error
4. `login.tsx:143-192` "Where is my key?" hint collapsed by default
5. No email config warning on first login (testing tier still active)
6. GetStarted uses `sessionStorage` not `localStorage` → hard-refresh loops back (`get_started.tsx:12-14`)
7. Proxy wizard forced as first experience (pushy for non-proxy users)
8. No visual confirmation of bootstrap success (silent redirect)
9. Migration logs mixed with ready signal, no banner
10. Bootstrap URL printed as plain text, no terminal hyperlink escape

**Top 3 fixes:**
- **F1.1** Single auth path: print only bootstrap URL, hide raw key unless bootstrap failed. `server.go:231`
- **F1.2** Readiness banner after migrations. `server.go:148`
- **F1.3** Auto-expand "Where is my key?" hint when `?bootstrap=` param detected but consume failed. `login.tsx:11-14`

---

### C2 — Admin dashboard (6.3/10)

**Keyboard shortcuts:** `?`, `/`, `n`, `r`, `g<key>` work. `Cmd+K` opens but doesn't route to search/nav.

**Page-by-page:**
- Overview ✓, Users ✓ (CreateUserSlideover works), Sessions ✓, Agents ✓, Audit ✓ (post-migration 00016), Session Debugger ✓
- Organizations 🔴 — `organizations.tsx:57` `<button>New</button>` has zero onClick, TeachEmptyState only refers to CLI
- Settings ⚠️ — read-only display, no PATCH, no dirty-state tracking
- Flow Builder ⚠️ — MVP, TODOs in-source: drag-reorder, conditional canvas fork, keyboard nav
- Webhooks ⚠️ — create+list+retry work, secret reveal has no copy-on-click button
- Dev Inbox ✓ (dev-mode only, graceful 404 when disabled)
- Empty shells (Tokens/API Explorer/Event Schemas/OIDC/Impersonation/Migrations/Branding) — intentional phase-gated stubs

**Top 3 fixes:**
- **F2.1** Orgs create slide-over (W01 already tracked)
- **F2.2** `Cmd+K` → page navigation (not just visual overlay)
- **F2.3** Webhook secret copy-on-click + unsaved-changes warning on Settings

---

### C3 — End-user auth (7.0/10)

**Security posture:** Excellent — Argon2id, constant-time, lockout, RFC-compliant CSRF on OAuth, magic-link rate limit, no enumeration.

**DX gaps:**
- No structured error envelope. `{error, message}` only. Missing `code`, `docs_url`, `details`. Integrators can't programmatically distinguish `weak_password` (too short vs dictionary vs common). Clerk/Auth0 ship RFC 7807 problem+detail.
- MFA enroll has stale state bug: user starts enroll → closes tab → next enroll blocked by `mfa_already_enabled` despite never verifying. Need `mfa_pending_verification` flag. `mfa_handlers.go:88-89`
- Passkey challenge 5min TTL expires silently, error returns generic `authentication_failed` with no "register new passkey" hint.
- Magic link rate-limit in-memory only (breaks multi-instance), no `Retry-After` header.
- Consent/device-flow HTML pages hardcode "Powered by SharkAuth" footer — no white-label hook.

**Top 3 fixes:**
- **F3.1** Standard `ErrorResponse` struct across all `/auth/**` with `code`, `docs_url`, `details`
- **F3.2** MFA pending-verification flag + allow re-enroll when only pending
- **F3.3** Passkey `expiresAt` in begin-response + distinct error codes on finish (`challenge_expired`, `no_matching_credential`)

---

### C4 — OAuth 2.1 server (7.2/10)

**Spec coverage:** RFC 6749 (code+PKCE+client_credentials+refresh) ✓, 7636 PKCE enforced S256-only ✓, 8414 metadata complete ✓, 7591 DCR ✓, 7662 introspect ✓, 7009 revoke ✓, 8628 device flow ✓, 9449 DPoP ✓, 8693 token exchange ✓, 8707 resource indicator ✓, 7517 JWKS ✓. **Missing: OIDC discovery (`/.well-known/openid-configuration`), `form_post` response mode, `error_uri` in errors, PAR (RFC 9126).**

**Error-shape inconsistency:**
- introspect: `{error, error_description}` (RFC-compliant)
- authorize login_required: `{error, login_url}` (Shark extension)
- OAuth callbacks: `{error, message}` (non-standard)

**Secret rotation:** Not exposed — integrator leaks secret → re-register client orphaning old.

**Top 3 fixes:**
- **F4.1** Standardize on `{error, error_description, error_uri}` across all `/oauth/*` endpoints (new `internal/oauth/errors.go` helper)
- **F4.2** Add `response_mode=form_post` rendering + advertise in metadata
- **F4.3** Ship `POST /oauth/register/{client_id}/secret` rotation endpoint + `DELETE` for registration access token rotation

---

### C5 — Agent auth stack (4.2/10 DX, 9/10 code) — THE MOAT

**Code reality:** DPoP full (JWK validation, JTI replay cache, thumbprint verify, ath), Device Flow full (user-code format, slow_down, approval UI), Token Exchange full (act claim, scope narrowing, may_act), Token Vault full (encrypted at-rest, auto-refresh, provider templates for Google/Slack/GitHub), Agent CRUD full, Resource Indicators full. **95% complete.**

**Doc reality:** `AGENT_AUTH.md:4` says "**Status: Design spec — not yet implemented**". Competitive table at line 28 shows Token Vault as "No (v2)" when it's shipping. Every HN/Reddit reader who checks the code-vs-doc drift will bounce.

**SDK reality:** Zero. Agent builders hand-roll DPoP proof JWT with cryptography library:
- Manual JWK x/y coordinate base64url encoding
- Manual `ath` = base64url(sha256(token))
- Manual jti collision check
- Manual clock-skew handling

**Walkthrough reality:** Zero. No "Hello Agent" doc. Builders must grep source to infer API.

**Competitive edge (real, undersold):** Only OSS option shipping DPoP + device flow + token exchange + vault in a single binary with SQLite. Auth0/Okta/Stytch are closed SaaS, charge 50%+ premium.

**Top 3 fixes:**
- **F5.1** Rewrite `AGENT_AUTH.md` header to "Status: MVP shipping (Waves A-E complete, 95% feature parity). Phase 6 next." Update Token Vault row. Add "Implementation Status" section. **Single highest-ROI doc fix in repo.** 2h.
- **F5.2** Ship `shark-sdk-py` (+ `-ts`, `-go`) with `DPoPProver`, `DeviceFlow.wait_for_approval()`, `VaultClient.get_fresh_token()`, `decode_agent_token()`. 500 LOC each package. 16h/pkg.
- **F5.3** "Hello Agent" walkthrough (`docs/hello-agent.md` + `examples/hello_agent.sh`) — 15-min end-to-end. Optional 10-min demo video. 4h doc + 6h video.

---

### C6 — Reverse proxy (7.1/10)

**Mechanics solid:** default-deny, glob path matching, circuit breaker with LRU + negative cache, identity header injection, panic recovery, strip-incoming safe-by-default.

**Config ergonomics:** YAML readable, `require:authenticated` vs `allow:anonymous` disambiguated, `scopes` AND-constraint explicit, timeout in seconds. Weaknesses: method filter not in example, scope semantics not in example, rule ordering not documented inline.

**Dashboard parity:** DB-backed rules have full CRUD, YAML rules read-only until restart. "Reload YAML" button is P5.1 TODO. Precedence unclear to new users.

**Observability:** Logs only deny decisions, not successful injections. Self-hoster debugging "why does upstream see wrong user_id" is blind. No `X-Shark-Injection-ID` trace header.

**W15 impact:** Single-port architecture means self-hoster must hit `:8080` instead of app's natural port (`:3000`). Breaks "install and protect in 60s" narrative. Workaround: Caddy in front = 3 processes defeats "one binary" pitch. Standalone mode has dedicated port but JWT verify unimplemented — `require:authenticated` always denies.

**Top 3 fixes:**
- **F6.1** Ship W15 multi-listener (already tracked)
- **F6.2** Injection audit trail with `--log-proxy-injections` opt-in + `X-Shark-Injection-ID` response header
- **F6.3** "Reload YAML" button + precedence banner ("DB rules override YAML")

---

### C7 — Auth flows builder (5.4/10) — USER-REPORTED BUG LOCATED

**Step catalog:**
- Wired (8): `require_email_verification`, `require_mfa_enrollment`, `require_mfa_challenge`, `require_password_strength`, `redirect`, `webhook`, `assign_role`, `add_to_org`
- Conditional: wired + 🔴 silent-save bug
- Deferred (3): `set_metadata`, `custom_check`, `delay` (wired:false, "v0.2" chip)

**Bug: conditional `then`/`else` branches lost on save round-trip.** Frontend POSTs `{then:[...], else:[...]}`. Backend stores fine. On re-fetch, response round-trip drops nested steps in `setFlow(updated)` state sync. Root cause: response handling in `flow_builder.tsx:293-299` OR backend response at `flow_handlers.go:331` not re-hydrating nested steps from DB.

**Top 3 fixes:**
- **F7.1** Fix conditional save bug. Add `TestUpdateFlow_ConditionalBranchesPreserved` integration test. Console warn on frontend if draft branches missing after save.
- **F7.2** Ship "Require MFA if risk high" template in palette quick-templates section
- **F7.3** Parse error responses on save failure — show actual `{error, message}` in toast instead of generic "Failed to save"

---

### C8 — CLI (5.0/10)

**Command tree:** `app {create,list,show,update,rotate-secret,delete}`, `init`, `serve`, `keys generate-jwt`, `proxy`, `health`, `version`, `completion`. **16 commands.**

**Missing (expected in production auth CLI):** `users`, `tokens`, `organizations`, `webhooks`, `audit`/`logs`, `config`, `migrate`, `backup`, `doctor`, `admin`.

**Flag-consistency disaster:** `--config` scattered (app/serve/keys yes; init no). Zero `--json`. Zero `--verbose`/`--debug`. Zero `--output` format selector.

**Help text quality:** Purposes clear, long-form explanations good (proxy, init, keys), **zero usage examples anywhere**, no exit code documentation.

**Scripting:** `app list` is tabwriter (unparseable). `app create` prints secret in fixed k:v format (requires brittle regex). No streaming / filtering / field selection. Competitors (stripe, gh, supabase, op, fly) all ship `--json` universally.

**Error UX:** Clear messages, SilenceUsage prevents wall-of-text, but no "did you mean" hints, no error categorization, no error-code documentation.

**Top 3 fixes:**
- **F8.1** Add `--json` to all output commands (`app list/show/create`, `keys generate-jwt`, `health`). 3-5d.
- **F8.2** Global persistent flags: `--verbose`, `--config` (inherit into all subcommands). 2d.
- **F8.3** Ship `shark doctor` (config + DB + port + secret + email checks) + add Examples section to every help text. 1-2d.

---

## Cross-component patterns

1. **@ts-nocheck everywhere** — 42 frontend files, all disable TS. Masked 9/24 P6.6 bugs. Tygo post-launch (W14).
2. **Response envelope drift** — admin endpoints use `{data}`, `{users}`, bare arrays, `{items}` inconsistently. Needs standardization → `{data, has_more?, next_cursor?}`.
3. **Docs lag code by weeks** — AGENT_AUTH.md most egregious; API.md missing recent endpoints; no hello-world walkthroughs anywhere.
4. **No SDK anywhere** — Phase 7. Every integrator hand-rolls.
5. **Silent catches** — still 7 across App.tsx/audit.tsx/flow_builder.tsx (W09/W10).
6. **No docs_url on errors** — integrator can't self-serve diagnostics.

---

## Launch-critical-path recommendation

| Ship before April 27 | Defer to Phase 7 | Post-launch polish |
|----------------------|------------------|---------------------|
| F5.1 doc rewrite (2h) | F5.2 SDK-py (16h) | F1.1-F1.3 onboarding polish |
| F7.1 conditional save bug (2h) | F4.2 form_post | F2.2 cmd+k routing |
| W01 orgs create (1h) | F6.2 injection audit | F3.2 MFA pending flag |
| F5.3 hello-agent walkthrough (4h) | F6.3 YAML reload button | F8.2 global flags |
| W15 multi-listener (4-6h) | F4.3 secret rotation | F2.3 webhook copy |
| F8.1 CLI --json (3d) | | |
| F3.1 structured error envelope (2d) | | |
| F4.1 standardize OAuth errors (3h) | | |

**Launch-critical total: ~6 working days solo.** Realistic for April 27 if pure-focus from April 22.

---

## Methodology note

Eight parallel Explore subagents, each persona-scoped, each ran: persona narrative → friction log → TTHW estimate → competitor benchmark → scorecard → top 3 fixes. Read-only. One agent used gstack live browse (C2); others code+API+docs only. Aggregation: this master scorecard + synthesis above.

**Audit date:** 2026-04-20 night session (Phase 6.7 extension).
**Branch:** `claude/admin-vendor-assets-fix`.
