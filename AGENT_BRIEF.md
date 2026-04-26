# W18 Recovery Brief — Post-Rogue-Agent

**Date:** 2026-04-26
**Branch:** main
**Context:** Previous W17 agent shipped major regressions. This brief tells subagents exactly what to fix, where, and what "done" looks like. Each section is self-contained — a subagent can pick up just one and ship it.

---

## Ground rules for all subagents

- **Worktree isolation:** if dispatched in parallel, MUST use `isolation: "worktree"`. Smoke/dist/DB collide otherwise.
- **No pytest from subagents:** orchestrator runs pytest only. It kills shark processes machine-wide.
- **Verify your impl files commit:** previous agents shipped tests without impl. After your last commit, run `git diff HEAD~1 --stat` and confirm both sides present.
- **Build pipeline (Windows):** `make -f Makefile.windows` OR `cd admin && npm run build && cd .. && go build -trimpath -ldflags="-s -w" -o shark.exe ./cmd/shark`
- **Style contract:** `.impeccable.md v3` — strict monochrome, square corners, editable cells, no toggles for jwt/cookie mode.

---

## TASK 1 — Rebuild Identity Hub via /impeccable craft

### Why
Commit `080471d` nuked `admin/src/components/identity_hub.tsx` and `flow_builder.tsx`. User wants Identity Hub back (Flow Builder stays nuked). Original spec preserved at git ref `080471d^`.

### Original spec (recover sections, NOT the literal file)
Per `documentation/inner_docs/admin/src/components/identity_hub.tsx.md` at `080471d^`:

- **Authentication methods** — password (min length, reset TTL), magic link (toggle, token TTL), passkeys (RP name, RP ID, user verification level), social providers (Google/GitHub/Apple/Discord rows; click row → drawer with client_id/client_secret/callback URL + copy)
- **Sessions & Tokens** — Cookie session lifetime, JWT access lifetime, JWT issuer, JWT audience, signing keys list (active marker + Rotate button + per-key inspect drawer). Cookie + JWT BOTH always on; NO mode toggle anywhere.
- **Active sessions** — `GET /api/v1/admin/sessions?limit=50` (user_email, auth_method, mfa chip, date). Per-row Revoke (`DELETE /admin/sessions/{id}`), bulk Revoke all (`DELETE /admin/sessions`), Purge expired (`POST /admin/sessions/purge-expired`).
- **MFA** — enforcement select (off/optional/required), TOTP issuer, recovery code count
- **SSO connections** — `GET /sso/connections`. Per-row Inspect drawer + Delete (`DELETE /sso/connections/{id}`). Refresh button. Create-via-API hint when empty.
- **OAuth Server** — toggle, issuer URL, access/refresh token lifetimes, DPoP requirement, device code approval queue (`GET /admin/oauth/device-codes`; Approve/Deny per pending code).

### Files to touch
- CREATE: `admin/src/components/identity_hub.tsx` — function `IdentityHub()` (default + named export). Read original via `git show 080471d^:admin/src/components/identity_hub.tsx` for reference, but rewrite per current `.impeccable.md v3`.
- CREATE: `documentation/inner_docs/admin/src/components/identity_hub.tsx.md`
- EDIT: `admin/src/App.tsx` — import `IdentityHub`, add `auth: IdentityHub` to Page route map.
- EDIT: `admin/src/components/layout.tsx` (Sidebar) — add nav item linking to `/admin/auth` with key `auth`. Original was placed in IDENTITY group.
- BUILD: `cd admin && npm run build`

### Done = 
- Visit `/admin/auth` shows full hub with all 6 sections.
- `npm run build` succeeds.
- No console errors.
- All 6 sections render, each save round-trips through `PUT /api/v1/admin/config` or its dedicated endpoint.

### How to invoke
This subagent should call `Skill` with `frontend-design:frontend-design` (a.k.a. `/impeccable craft`) to ensure visual quality matches reference pages (`users.tsx`, `agents_manage.tsx`, `sessions.tsx`).

---

## TASK 2 — Fix dev_email tab redirect bug

### Why
`admin/src/App.tsx` contains:
```js
React.useEffect(() => {
  if (adminConfig !== null && emailProvider !== '' && emailProvider !== 'dev' && page === 'dev-email') {
    setPage('overview');
  }
}, [adminConfig, emailProvider, page]);
```
This kicks the user out of the Dev Email page when `email.provider != 'dev'`. User wants the tab always reachable; provider-flip is a separate concern. Also `dev_email.tsx` text references `--dev` flag, which is deprecated (use `shark mode dev` instead).

### Files to touch
- EDIT `admin/src/App.tsx` — DELETE the auto-redirect useEffect block above.
- EDIT `admin/src/components/dev_email.tsx` — `grep -n "\\-\\-dev\\|--dev" admin/src/components/dev_email.tsx` to find every ref. Replace each:
  - `--dev` flag mentions → "Set provider to `dev` in Settings → Email Delivery, or run `shark mode dev`"
  - Any "activate via terminal" hint → drop or rewrite to `shark mode dev` / `shark reset dev`
- EDIT `admin/src/components/sidebar.tsx` (or wherever Dev Email tab is gated) — ensure tab is always visible.

### Done = 
- Visiting `/admin/dev-email` while `email.provider=resend` does NOT redirect.
- Page renders with banner: "Provider currently set to <X>. Switch to `dev` in Settings → Email Delivery to capture outbound mail here."
- No `--dev` string remains in `admin/src/components/dev_email.tsx` or `admin/src/App.tsx`.
- `npm run build` succeeds.

---

## TASK 3 — Fix Settings Danger Zone overlap/scroll

### Why
`admin/src/components/settings.tsx` line 739 — Danger Zone block is positioned outside the inner scroll container at line 488 (`<div style={{ flex: 1, overflowY: 'auto', paddingBottom: 80 }}>`). User says danger zone is static while rest scrolls and they overlap.

### Files to touch
- EDIT `admin/src/components/settings.tsx`:
  - Move the Danger Zone JSX block (~line 739 onward, the section labeled "Danger Zone") INSIDE the scroll container that ends near the closing of the section list.
  - Confirm: parent flex layout `display: flex; height: 100%; overflow: hidden` (line 432) → child main column gets one scroll container that holds ALL sections including Danger Zone.
  - If Danger Zone needs to be at the bottom always, place it last inside the same scroll container — don't make it sticky.

### Done = 
- Scroll the settings page top-to-bottom — every section flows naturally.
- Danger Zone appears as the last section, scrolls with the rest, no overlap with sections above.
- Left rail anchor scroll for Danger Zone (if there's a nav item) still works.
- `npm run build` succeeds.

---

## TASK 4 — ASCII glyph on every boot + admin key wide banner

### Why
`internal/cli/branding.go` has a tiny 5-line ASCII glyph and `PrintHeader(out)` is called only by `cmd/shark/cmd/version.go:21`. `serve.go` does NOT call it on startup. Result: glyph never appears on `shark serve`. Also `firstboot.go` returns the full admin key but only writes it to `data/admin.key.firstboot` (file) — terminal output is masked prefix+suffix only.

### Files to touch

**A. Print glyph on every boot:**
- EDIT `cmd/shark/cmd/serve.go` — call `cli.PrintHeader(os.Stdout)` at the start of `RunE` (before any server bring-up).
- VERIFY: glyph + tagline + version prints first thing.

**B. Improve glyph (optional but desired):**
- EDIT `internal/cli/branding.go` — replace `sharkGlyphASCII` const with a larger, recognizable shark. Hand-craft (~12-15 lines, ~40 chars wide). Reference: search "shark ascii art" on `https://www.asciiart.eu/animals/fish` patterns. Stay monochrome, use `_/\<>` characters.

**C. Wide admin key banner on first boot:**
- EDIT `cmd/shark/cmd/serve.go` (or wherever `RunFirstBoot` result is consumed in `internal/server/server.go`):
  - When `FirstBootResult != nil` (first boot path), print to stdout:
    ```
    ════════════════════════════════════════════════════════════════════════════════
                  ⚠  ADMIN API KEY — YOU WILL ONLY SEE THIS ONCE  ⚠
    ════════════════════════════════════════════════════════════════════════════════
    
        sk_live_<full_key_here>
    
        Setup URL:    http://localhost:8080/admin/setup?token=<setup_token>
        Saved to:     data/admin.key.firstboot  (perms 0600 — delete after use)
    
    ════════════════════════════════════════════════════════════════════════════════
    ```
  - Use `cli.IsColorEnabled` for the banner (cyan border in color, plain ascii in NO_COLOR).
  - On NON-first-boot: print nothing about the key.

### Done = 
- `rm -rf data/ && ./shark serve --no-prompt` → glyph prints, then BIG banner with full `sk_live_*` key.
- `./shark serve --no-prompt` second time (state already bootstrapped) → glyph prints, NO key banner.
- `cli_test.go TestPrintHeader_NoColor` still passes.

---

## TASK 5 — First-boot dashboard flow: wide key banner + /get-started + dismissable tutorial tab

### Why
User wants: on first browser visit after first-boot, dashboard shows a full-width banner with the admin key (one-time view), then lands on `/get-started`. While the walkthrough isn't completed, a small horizontal tab on top of `/overview` reminds them to finish, with a "nvm" (dismiss) button.

### Files to touch

**A. Detect first boot in admin UI:**
- The setup page (`admin/src/components/setup.tsx`) already shows the admin key once after magic-link signin. Extend or replace this with a full-width banner inside the dashboard chrome (post-login):
- EDIT `admin/src/App.tsx`:
  - On login from `Setup` flow, set `localStorage.setItem('shark_first_boot_pending_key', '<full_key>')`.
  - On dashboard mount, if that localStorage key exists AND not yet dismissed, render a full-width banner above `<TopBar />`:
    ```
    ┌─────────────────────────────────────────────────────────────────────┐
    │ ⚠ ADMIN API KEY — YOU WILL ONLY SEE THIS ONCE                       │
    │ sk_live_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx        [Copy] [Dismiss]   │
    └─────────────────────────────────────────────────────────────────────┘
    ```
  - On dismiss → `localStorage.removeItem('shark_first_boot_pending_key')` + setPage('get-started').

**B. Land on /get-started after dismiss:**
- After dismiss, `setPage('get-started')`. Already a route in App.tsx.

**C. Tutorial tab on /overview:**
- EDIT `admin/src/components/overview.tsx`:
  - At the top of the component (above existing content), if `localStorage.getItem('shark_walkthrough_seen') !== '1'`:
    ```jsx
    <div className="row" style={{ /* horizontal tab style */ }}>
      <span>Finish setup tour →</span>
      <button onClick={() => setPage('get-started')}>Open</button>
      <button onClick={() => { localStorage.setItem('shark_walkthrough_seen','1'); /* re-render */ }}>nvm</button>
    </div>
    ```
  - Use minimal, monochrome styling matching overview chrome.
  - Hide tab once `shark_walkthrough_seen === '1'`.

### Done = 
- Wipe browser localStorage + `data/` → restart server → magic-link in → dashboard shows full-width key banner.
- Dismiss → lands on /get-started.
- Click Overview → small "Finish setup tour" tab visible on top.
- Click "nvm" → tab disappears, persists across reloads.

---

## TASK 6 — Smoke regression recovery (CONFIRMED ROOT CAUSE)

### Baseline (2026-04-26)
150 tests collected. Run timed out at 300s after ~73 tests. Heavy `F`/`E` cluster matches the 8 files below.

### Root cause — deprecated flags + yaml in 8 test files
Each file does `subprocess.Popen([bin, "serve", "--dev", "--config", cfg_path], ...)` against a yaml file it writes to tmp_path. Both `--dev` and `--config` are removed in W17.

Affected files + line numbers:
- `test_proxy_dpop.py:35,125,148,153`
- `test_proxy_header_spoof_strip.py:22,101,124,129`
- `test_proxy_lifecycle.py:24,93,116,121`
- `test_proxy_paywall_render.py:22,50,69,74`
- `test_proxy_rules_idempotency.py:28,56,75,80`
- `test_proxy_tier_gate.py:32,119,142,148`
- `test_vault_proxy_flows.py:4` (yaml import — verify usage)
- `test_w15_advanced.py:11,105,144,149`

Plus: `test_proxy_yaml_deprecation.py` is obsolete — yaml infrastructure was hard-removed in Phase H, so the deprecation WARNING source it asserts (`internal/server/server.go ~line 108`) no longer exists. **Delete this file.**

### Fix pattern (apply to each of the 8 files)
1. Remove `import yaml`.
2. Remove the `cfg = {...}` dict + `yaml.dump(cfg, f)` block.
3. Change spawn line from:
   ```python
   [bin_abs, "serve", "--dev", "--config", cfg_path]
   ```
   to:
   ```python
   [bin_abs, "serve", "--no-prompt", "--proxy-upstream", "http://localhost:NNNN"]
   ```
   (where NNNN is the upstream port the test set up; pass other config via admin API after boot).
4. After server is healthy, parse the admin key from `data/admin.key.firstboot` (use the same helper conftest already uses, or call `_read_admin_key()`).
5. Push proxy rules via `POST /api/v1/admin/proxy/rules` instead of yaml `proxy.rules:` block.
6. For tests that previously set `server.dev_mode: true` via yaml: hit `PATCH /api/v1/admin/config` with `{"server":{"dev_mode":true}}` (or `{"email":{"provider":"dev"}}` if it's only the dev email route they need).
7. Each test should clean its own `data/` dir BEFORE spawning (each writes to per-test tmp_path now? — verify; if tests share `data/`, add a pytest-fixture cleanup).

### Done = 
- All 8 files no longer reference `import yaml`, `--config`, `--dev`, `*.yaml` config files.
- `test_proxy_yaml_deprecation.py` deleted.
- `python -m pytest --tb=line` from orchestrator: 150 tests, ≤5 pre-existing unrelated failures, no W17-deprecation regressions.
- All tests use admin API to push runtime config.

---

## TASK 7 — Investigate proxy non-caught bugs + CLI/CRUD parity

### Why
User: "Find non-caught bugs in proxy" + "investigate cli parity and CRUD on all important components and backend."

### Process
Dispatch `general-purpose` subagent with model `sonnet` running `/investigate` skill flow. Read-only. Output report at `INVESTIGATE_REPORT.md` covering:

1. **Proxy bugs:** read `internal/proxy/{proxy.go, rules.go, headers.go, circuit.go, lifecycle.go, listener.go}` + tests. Hunt:
   - Race conditions on rule reload (`lifecycle.go`)
   - Header injection / hop-by-hop strip gaps (`headers.go`)
   - Circuit breaker thresholds vs config bounds
   - LRU eviction correctness under concurrent access (`lru.go`)
   - Path matching ambiguity in `rules.go`
   - DPoP enforcement consistency (`proxy_dpop_test.go`)

2. **CLI/CRUD parity:** for each resource (user, app, agent, key, branding, vault, sso, webhooks, rbac, organizations, consents, sessions, audit), check:
   - Admin API has full CRUD?
   - CLI has matching subcommand for each verb?
   - Dashboard has matching UI?
   - Report gaps as a table.

3. Output: `INVESTIGATE_REPORT.md` — section per finding, with file:line refs and severity (P0/P1/P2).

---

## TASK 8 — Revise HANDOFF_W17.md → HANDOFF_W18.md

### Why
Stale handoff doesn't reflect rogue-agent damage or recovery plan.

### Process
Replace top sections of HANDOFF_W17.md (or create HANDOFF_W18.md) with:
- W17 status: shipped but with regressions noted in this brief
- W18 in-flight tasks: list 1-7 above with per-task subagent + status
- Known damage list: identity_hub nuked, dev_email redirect bug, danger zone overlap, glyph not printing, key not visible
- Smoke status: TBD pending baseline run

---

## File reference quick map

| Concern | File |
|---|---|
| App routes | `admin/src/App.tsx` |
| Identity Hub (recover) | `admin/src/components/identity_hub.tsx` (DELETED, restore from `git show 080471d^:`) |
| Sidebar nav | `admin/src/components/layout.tsx` |
| Settings + Danger Zone | `admin/src/components/settings.tsx` (line ~739) |
| Dev Email | `admin/src/components/dev_email.tsx` |
| Overview (tutorial tab) | `admin/src/components/overview.tsx` |
| Get Started | `admin/src/components/get_started.tsx` |
| First-boot detection | `admin/src/components/setup.tsx`, `App.tsx` setup branch |
| ASCII glyph | `internal/cli/branding.go` |
| First boot Go | `internal/server/firstboot.go`, `internal/server/server.go` |
| Serve cmd | `cmd/shark/cmd/serve.go` |
| Mode/reset | `cmd/shark/cmd/mode.go`, `cmd/shark/cmd/reset.go` |
| Proxy | `internal/proxy/*.go` |
| Smoke conftest | `tests/smoke/conftest.py` |

## Non-goals (do NOT)

- Resurrect `flow_builder.tsx` — user confirmed nuked.
- Resurrect `--dev` flag — deprecated, use `shark mode dev`.
- Resurrect `dev_inbox.tsx` — replaced by `dev_email.tsx`.
- Add yaml/koanf — fully removed in W17.
