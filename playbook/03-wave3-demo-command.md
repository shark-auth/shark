# Wave 3 — `shark demo delegation-with-trace`

**Budget:** 5-8h CC · **Outcome:** viral 30-second screencast artifact

## What it does

One command. Spins up shark in a temp DB, registers 3 agents, runs a delegation chain with DPoP at each hop, generates a self-contained HTML report with the cryptographic proofs visible. Auto-opens in browser.

Goal: a HN reader sees the screencast, says "wait, that's actually new." A YC partner sees the same screencast, says "this is differentiated."

## CLI surface

```bash
shark demo delegation-with-trace
# Optional flags:
#   --output <path>   Override report destination (default: ./demo-report.html)
#   --no-open         Don't auto-open the browser
#   --keep            Keep the temp DB for inspection (default: cleanup on exit)
```

Exit code 0 on success. Prints the report path to stdout. Auto-opens via existing `cmd/shark/cmd/serve.go` browser-open helper.

## File layout

```
cmd/shark/cmd/
  demo.go                 # cobra command registration
internal/demo/
  delegation.go           # the orchestration logic
  template.go             # embedded HTML template + render
  template/
    report.html.tmpl      # go template, embedded via //go:embed
    style.css             # monochrome/square per .impeccable.md v3
    inline.js             # minimal JS for collapse/copy buttons
```

## Orchestration steps (delegation.go)

1. **Spin up shark in-process** (or as subprocess) on a random local port with a temp SQLite DB. Reuse `internal/server/firstboot.go` to bootstrap admin + setup token. Disable email provider (no SMTP needed).
2. **Register a synthetic human user** `demo-user@sharkauth.dev` with magic-link bypass for the demo.
3. **Register 3 agents** via admin API, all with `created_by = demo-user`:
   - `user-proxy` (top of chain — acts as the user's primary delegate)
   - `email-service` (mid — needs `act` claim for delegation)
   - `followup-service` (bottom — only acts on a sub-token from email-service, fetches from vault)
4. **Configure `may_act` policies:**
   - `user-proxy` may delegate to `email-service` (scope: `email:send`, `email:read`, `vault:read`)
   - `email-service` may delegate to `followup-service` (scope: `email:read`, `vault:read`)
5. **Provision a vault entry** — a synthetic Gmail OAuth connection for `demo-user`. Fake provider credentials, fake refresh token, fake access token. Vault treats it as real because the encryption + audit path is identical.
6. **Generate DPoP keypairs** for each agent (use existing DPoP package). Capture each `jkt`.
7. **Issue tokens, hop by hop:**
   - Token 1: `user-proxy` requests `client_credentials` with DPoP, `sub=demo-user@sharkauth.dev` → access_token bound to `user-proxy.jkt`, `act` is empty (root)
   - Token 2: `email-service` calls `/oauth/token` with `grant_type=token-exchange`, `subject_token=token_1`, DPoP proof signed by `email-service.jkt`. Server validates `may_act`, returns token with `act` chain `[user-proxy]`, scope narrowed, `cnf.jkt = email-service.jkt`. `sub` still `demo-user`.
   - Token 3: `followup-service` exchanges Token 2. Returns token with `act` chain `[user-proxy, email-service]`, scope narrowed to `email:read vault:read`, `cnf.jkt = followup.jkt`. `sub` still `demo-user`.
8. **Vault retrieval (the killer hop):** `followup-service` calls `GET /api/v1/vault/google_gmail/token` with `Authorization: DPoP <token_3>` + DPoP proof. Server:
   - Validates DPoP proof binds to followup-service.jkt
   - Validates token has `vault:read` scope
   - Validates `tok.UserID == demo-user`
   - Returns the (synthetic) Gmail access token
   - Writes audit event `vault.token.retrieved` with `actor_id=followup-service`, `target=connection-id`, metadata including the full `act_chain` JSON
9. **Capture each token** (decoded claims), each DPoP proof JWT, each audit event written, plus the vault retrieval result.
10. **Render the report** using the embedded template + captured data.
11. **Cleanup** temp DB unless `--keep` set.

## Report layout (report.html.tmpl)

Self-contained HTML. No external CDN. Inline CSS + JS. Embedded into the binary via Go `//go:embed`.

```
┌──────────────────────────────────────────────────────────────────────────┐
│  SharkAuth · Delegation Trace · 2026-04-26 02:55:00                     │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│   [user] → [user-proxy] ───→ [email-service] ───→ [followup-service]    │
│             (jkt: zQ7m...)    (jkt: 5kL9...)       (jkt: aB3x...)       │
│                                                                          │
├──────────────────────────────────────────────────────────────────────────┤
│  TOKENS                                                                  │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐         │
│  │ Token 1         │  │ Token 2         │  │ Token 3         │         │
│  │ sub: user-proxy │  │ sub: email-svc  │  │ sub: followup   │         │
│  │ act: —          │  │ act: [user-pxy] │  │ act: [user-pxy, │         │
│  │                 │  │                 │  │       email-svc]│         │
│  │ scope: email:*  │  │ scope: email:*  │  │ scope: email:rd │         │
│  │ cnf.jkt: zQ7m...│  │ cnf.jkt: 5kL9...│  │ cnf.jkt: aB3x...│         │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘         │
│  (each card collapsible — full decoded JWT below)                       │
├──────────────────────────────────────────────────────────────────────────┤
│  DPoP PROOFS                                                             │
│  Token 1 proof: htm=POST htu=/oauth/token jti=... iat=... [verify ✓]    │
│  Token 2 proof: htm=POST htu=/oauth/token jti=... ath=... [verify ✓]    │
│  Token 3 proof: htm=POST htu=/oauth/token jti=... ath=... [verify ✓]    │
├──────────────────────────────────────────────────────────────────────────┤
│  VAULT RETRIEVAL (the closed loop)                                       │
│  followup-service called GET /api/v1/vault/google_gmail/token            │
│  ✓ DPoP proof valid (jkt: aB3x...)                                       │
│  ✓ Scope vault:read present                                              │
│  ✓ User binding: demo-user@sharkauth.dev                                 │
│  ✓ Audit event: vault.token.retrieved                                    │
│    actor: followup-service                                               │
│    act_chain: [user-proxy, email-service]                                │
│    target: gmail_connection_for_demo-user                                │
│    metadata: { provider_id, provider_name: "google_gmail" }              │
├──────────────────────────────────────────────────────────────────────────┤
│  AUDIT TRAIL                                                             │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │ time          actor          action                target        │   │
│  │ 02:55:01.142  user-proxy     oauth.token.issued    —            │   │
│  │ 02:55:01.234  email-service  oauth.token.exchanged user-proxy   │   │
│  │ 02:55:01.345  followup-svc   oauth.token.exchanged email-svc    │   │
│  │ 02:55:01.421  followup-svc   vault.token.retrieved gmail-conn   │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────────────────┘
```

**Why the vault hop matters for the demo:** without it, the report shows "agents can delegate to each other." Cool but abstract. With it, the report shows the closed loop: a third-hop agent fetched a real (encrypted-at-rest) credential from the vault and used it, with cryptographic proof that ties the action back to the original human user. No other auth product can show this end-to-end in 30 seconds.

Style: monochrome, square corners, generous whitespace. Reuse existing dashboard tokens from `admin/src/styles/`. No animations beyond collapse/expand. No external font (system font stack).

## Implementation notes

- **Embed the template:** `//go:embed template/*` in `internal/demo/template.go`. Single binary, no asset dir to ship.
- **No new deps:** use `text/template` from stdlib, no chart libs. The "graph" is just text + arrows.
- **Speed target:** total run < 60 seconds on cold cache. Most time is server startup (~2s) + 3 token exchanges (~50ms each).
- **Verify smoke green** after merge: the demo command must not affect production code paths; `cmd/shark/cmd/demo.go` is additive only.

## Acceptance

- `shark demo delegation-with-trace` runs to completion in <60s
- Report file generated at `./demo-report.html`
- Report opens in default browser
- Visiting the report on a clean machine (no shark running) renders fully — fully self-contained
- All 3 token cards show real decoded claims with `act` chains visible
- All 3 DPoP proofs show signature verification status
- Audit table shows 3 events, timestamps in correct order
- Smoke suite GREEN (375 PASS) after merge
- 30-second screencast captures: terminal command → browser opens → user scrolls through report → ends with audit table

## Fallback (if energy collapses)

Drop the HTML template entirely. Replace with structured stdout output:

```bash
$ shark demo delegation-with-trace --plain
[1/3] Registering agents...                    ✓
[2/3] Configuring may_act policies...          ✓
[3/3] Running delegation chain...

  user → user-proxy → email-service → followup-service

Token 1: scope=email:* cnf.jkt=zQ7m... act=[]
Token 2: scope=email:* cnf.jkt=5kL9... act=[user-proxy]
Token 3: scope=email:read cnf.jkt=aB3x... act=[user-proxy, email-service]

DPoP proofs: 3/3 verified ✓
Audit events: 3 written

Run with --html to generate the visual report (W18+).
```

This still demonstrates the moat. Loses the screenshot virality but ships in 2-3h instead of 5-8h. Ship the HTML version W+1 if needed.

## Definition of done for Wave 3

- New command `shark demo delegation-with-trace` lives in `cmd/shark/cmd/demo.go`
- Embedded HTML template renders correctly on first run
- Report is fully self-contained (works opened from disk on a fresh machine)
- 30-second screencast recorded and saved to `playbook/screencast.mp4` (or wherever)
- Smoke suite GREEN
- README links to the demo command in the "Try it" section
