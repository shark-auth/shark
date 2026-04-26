# Wave 1.8 — CLI / Terminal Output Polish

**Budget:** 4-6h CC · **Run after Wave 1.7, before Wave 2** · **Outcome: every terminal output a developer sees is screenshot-shareable**

## Why this wave exists

The first thing every developer sees when running `shark serve` is the terminal. Not the dashboard. The CLI output IS the first impression. Stripe's CLI, Vercel's CLI, Railway's CLI, Fly's CLI are iconic — developers screenshot them, post them on Twitter, the CLI itself becomes a viral moment.

Today (per memory): branded ASCII header + slog handler shipped Phase 7G. Probably still rough. Wave 1.8 polishes every terminal-facing surface to launch-grade.

User directive: "make all logs beautiful (the ones in the terminal)."

## What "beautiful CLI output" means concretely

Not "more colors." Specifically:

1. **Restraint.** Use color sparingly. Default to mono. Color only when it earns information density (errors red, success green, warnings yellow, dimmed gray for timestamps).
2. **Aligned columns** for any structured data (audit log tail, agent list, token info).
3. **Progressive boot sequence** — not a wall of debug logs. Stripe's CLI does this perfectly: ✓ checkmarks as each subsystem starts, single line per phase, final URL printed boldly.
4. **Heartbeat after boot** — single live status line: `shark · :8080 · 3 agents · 0 active · 247 audit events · ↑ 1m23s`
5. **Clean error formatting** — bold red label, indented body, suggestion line, link to docs. NOT a Go stack trace.
6. **Smart truncation** — long IDs shown as `agt_zQ7m...Lw2` with full ID copyable on `--verbose`.
7. **No noise** — debug logs OFF by default. Suppress framework chatter (gin/chi/fiber default logs).
8. **Demo command output** specifically tuned — `shark demo delegation-with-trace` should print a visually striking preview before opening the browser.

## Surfaces to polish

### Surface 1 — `shark serve` first-boot sequence (~1.5h)

**File:** `cmd/shark/cmd/serve.go`, `internal/server/firstboot.go`

**Current (per memory + audit):** branded ASCII header exists, then mixed slog output during boot. Progressive but probably unbalanced.

**Target output:**

```
   ██╗ █████╗ █████╗ ██╗  ██╗
   ██║██╔══██╗██╔══██╗██║ ██╔╝
   ██║███████║███████║█████╔╝ 
██╗██║██╔══██║██╔══██║██╔═██╗ 
╚████╔╝██║  ██║██║  ██║██║  ██╗
 ╚═══╝ ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝

shark v0.1.0 · agent-native auth

✓ database         sqlite at data/sharkauth.db
✓ migrations       21 applied · schema v00021
✓ secrets          loaded from data/.secrets
✓ jwt keys         3 active · ES256
✓ vault            ready · 0 connections
✓ smtp             dev-inbox at /admin/dev-email
✓ http server      listening on :8080

  Dashboard:  http://localhost:8080/admin
  Setup key:  data/admin.key.firstboot (one-time)

  shark · :8080 · 0 agents · 0 active · 0 events · ↑ 0s
```

**Implementation:**
- Boot phases run in series with a single ✓ line per phase. Use `→` while in-flight, `✓` on success, `✗` on failure.
- Phase function signature: `func bootPhase(name string, fn func() error) error` — handles the ✓ /✗ logic uniformly. New file `internal/server/bootlog.go`.
- Suppress framework default logs (chi router default logging, sqlite chatter) — reroute to slog at debug level only.
- Final two lines before heartbeat: dashboard URL + setup key path. Bold/highlighted.
- Heartbeat starts AFTER boot complete. Single line, updates in place every 5s using ANSI cursor save/restore. Stop heartbeat on Ctrl+C with one final state line.

### Surface 2 — Slog handler formatting (~1h)

**File:** `internal/log/slog_handler.go` (or wherever the existing branded handler lives)

**Current (per memory Phase 7G):** branded slog handler shipped. Probably formats levels and timestamps but may not handle structured fields cleanly.

**Target format:**

```
14:32:01  INFO   agent.registered           agent_id=agt_zQ7m client_id=cli_8kL...Lw2 user_id=usr_42
14:32:14  WARN   oauth.token.expired        agent_id=agt_5Lm9 grace_period=30s
14:32:18  ERROR  vault.token.refresh.failed provider=google_gmail user_id=usr_42 cause="invalid_grant"
                                            → Run `shark vault reconnect <provider>` for usr_42
                                            → Docs: https://shark.dev/docs/vault/reconnect
```

**Implementation:**
- Timestamp: `15:04:05` only (no date — only Go convention HH:MM:SS, dimmed gray)
- Level: 5-char fixed-width, color-coded (INFO blue, WARN yellow, ERROR red, DEBUG dim gray)
- Action: dotted namespace (e.g., `vault.token.refresh.failed`) — fixed width minimum 30 chars, padded right
- Fields: `key=value` pairs, dimmed gray, single space separated, smart truncation for long IDs (first 4 + last 3 chars + `...`)
- Errors: indented `→ ` lines with action suggestion + docs URL — only printed for ERROR level
- Multi-line errors collapse to one line by default; expand with `--verbose`
- Color suppressed when stdout is not a TTY (CI safe)

### Surface 3 — `shark agent register` and CLI command output (~45 min)

**File:** `cmd/shark/cmd/agent_register.go`, `cmd/shark/cmd/output.go` (helpers)

**Current:** prints client_id and secret on registration.

**Target output:**

```
$ shark agent register --name email-bot

✓ Agent registered

  Name             email-bot
  Agent ID         agt_zQ7mRsLw2
  Client ID        cli_8kL9_0aB3xN
  Client Secret    sk_live_xY9...Jq2  (shown once — store it now)

  → Generate a DPoP keypair:
      shark agent dpop generate --agent agt_zQ7mRsLw2

  → Set delegation policy:
      shark agent policy set agt_zQ7mRsLw2 --may-act <other-agent> --scope <scope>

  → Test with a token:
      shark token issue --agent agt_zQ7mRsLw2
```

**Implementation:**
- New helper `internal/cli/output/kv.go` — formats key/value pairs with aligned columns, dimmed labels.
- Suggestions block uses `→` arrow, indented 4 spaces, dimmed.
- Same pattern applied to: `shark vault provider create`, `shark mode dev`, `shark mode prod`, `shark reset`, `shark token issue`, `shark token revoke`.

### Surface 4 — `shark demo delegation-with-trace` console output (~45 min)

**File:** `cmd/shark/cmd/demo.go`, `internal/demo/output.go` (new helpers)

**Target output:**

```
$ shark demo delegation-with-trace

→ Spinning up shark in temp DB...                      ✓ 142ms
→ Registering 3 agents (user-proxy, email, followup)...  ✓ 89ms
→ Configuring may_act policies...                      ✓ 12ms
→ Provisioning vault entry (synthetic Gmail)...        ✓ 38ms
→ Generating DPoP keypairs (3 ECDSA P-256)...          ✓ 21ms

DELEGATION CHAIN

  user@demo  →  user-proxy  →  email-service  →  followup-service
                jkt:zQ7m...    jkt:5kL9...      jkt:aB3x...

→ Issuing token 1 (user-proxy, client_credentials)...   ✓ 31ms
→ Exchanging token 2 (email-service, RFC 8693)...       ✓ 24ms
→ Exchanging token 3 (followup-service, scope narrowed) ✓ 28ms
→ Vault retrieval (followup → google_gmail)...          ✓ 42ms

✓ Delegation chain executed end-to-end (411ms total)
✓ All cryptographic proofs verified
✓ Audit trail written (4 events)

  Report saved:    ./demo-report.html
  Opening browser...
```

**Implementation:**
- Each step shows `→ description...                   ✓ Xms` with right-aligned timing
- Failed steps show `✗ description ... ERROR: <one-line cause>` and stop
- Final block: 3 ✓ summary lines + report path + browser open
- Same `bootPhase` helper from Surface 1 reused

### Surface 5 — Error output and `--verbose` mode (~30 min)

Every error path in CLI commands and serve loop:
- One-line summary in default mode
- Full stack + structured fields in `--verbose` (enabled per-command and via env `SHARK_VERBOSE=1`)
- Actionable suggestions where possible (link to docs path or sibling command)

## Discipline directive

Per user direction: every NEW component or screen uses `/impeccable` workflow. CLI output polish is the terminal-side equivalent. Going forward: any new CLI command or new structured log event should follow these formatting rules from day one.

Add to `CONTRIBUTING.md` (new section if file exists, create the section if not): "CLI output style guide" pointing to this Wave 1.8 file as canonical.

## Definition of done for Wave 1.8

- `shark serve` boot sequence matches the target output (visual review with screenshot saved to `playbook/screenshots/cli-boot.png`)
- Slog handler formats consistent across all log levels
- All `shark` CLI commands use the kv-output helper for structured response
- `shark demo delegation-with-trace` console output matches target (this becomes part of the screencast — both terminal and HTML report visible in the 30s)
- Color suppressed when stdout is not a TTY (verify by piping to file: `shark serve > /tmp/log.txt` should produce ANSI-free output)
- `--verbose` flag works on all top-level commands
- Smoke suite GREEN
- Subjective test: would you screenshot this? If yes, ship. If no, polish another pass.

## Demo + launch impact

The Wave 3 screencast (30s viral artifact) prominently features:
1. Terminal: `shark serve` boot — Wave 1.8 makes this beautiful
2. Terminal: `shark demo delegation-with-trace` — Wave 1.8 makes this beautiful
3. Browser: dashboard moat exposure — Wave 1 makes this beautiful

If Wave 1.8 ships, the entire screencast is screenshot-shareable. Twitter and HN react to CLI craft strongly — Stripe, Fly, Railway built developer love partly through CLI polish.

## Cuts available within Wave 1.8 if budget breaks

If Sunday hits and Wave 1.8 isn't done, cut in this order:
1. Surface 5 (verbose mode polish) — defer to W18
2. Surface 3 (per-command CLI output) — keep current state for non-demo commands
3. Surface 4 (demo command output) — keep but skip the perfect alignment
4. Surface 2 (slog handler) — keep current Phase 7G baseline
5. Surface 1 (`shark serve` boot) — DO NOT cut. This is the first impression.

Surface 1 alone is ~1.5h and is the highest-leverage piece. If only one surface ships in 1.8, ship Surface 1.
