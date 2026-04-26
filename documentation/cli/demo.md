# shark demo

Run interactive SharkAuth demo flows against a live server.

## Commands

### `shark demo delegation-with-trace`

Registers 3 synthetic agents, configures `may_act` policies, runs a 3-hop
DPoP-bound token-exchange delegation chain, then performs a **vault hop** (hop 4)
where `followup-service` fetches a Gmail OAuth token from the vault using the
delegated token. Generates a self-contained HTML report and auto-opens it in
the browser.

**Requires a running `shark serve` instance.**

```
Usage:
  shark demo delegation-with-trace [flags]

Flags:
  --admin-key string   admin API key (or SHARK_ADMIN_KEY env)
  --base-url string    shark server base URL (default "http://localhost:8080")
  --output string      write HTML report to this path (default "./demo-report.html")
  --no-open            do not auto-open the report in a browser
  --keep               keep temp DB for inspection after run
  --plain              force plain stdout output only (skip HTML)
```

#### Orchestration steps

1. Register 3 synthetic agents: `user-proxy`, `email-service`, `followup-service`
2. Configure `may_act` policies (DPoP-bound, scoped)
3. Run 3-hop token exchange — each hop narrows scope, extends `act` chain
4. **Vault hop:** `followup-service` calls `GET /api/v1/vault/google_gmail/token`
   with `Authorization: DPoP <token_3>` — server validates DPoP binding, scope,
   and user ownership, then returns the (synthetic) Gmail access token
5. Render a self-contained HTML report with decoded claims, DPoP proofs, and
   an audit trail

#### Example output

```
[1/3] Registering agents...
  ✓ agent 1: demo-user-proxy (id=ag_... jkt=zQ7m...aB3x)
  ✓ agent 2: demo-email-service (id=ag_... jkt=5kL9...cD2w)
  ✓ agent 3: demo-followup-service (id=ag_... jkt=aB3x...eF1z)
[2/3] Configuring may_act policies...
  ✓ user-proxy → email-service (email:*, vault:read)
  ✓ email-service → followup-service (email:read, vault:read)
[3/3] Running delegation chain...

  user → user-proxy → email-service → followup-service

Token 1: scope=email:* cnf.jkt=zQ7m...aB3x act=[]
Token 2: scope=email:* cnf.jkt=5kL9...cD2w act=[demo-user-proxy]
Token 3: scope=email:read cnf.jkt=aB3x...eF1z act=[demo-user-proxy, demo-email-service]

DPoP proofs: 3/3 verified ✓
Audit events: 3 written
[4/4] Fetching from vault (google_gmail)... ✓ access_token retrieved

Report written to ./demo-report.html
```

#### HTML Report

The generated report is fully self-contained (no CDN, no external assets).
It includes:

- **Delegation breadcrumb** — visual chain from `[user]` through each agent with JKT fingerprints
- **Token cards** — decoded claims (`sub`, `scope`, `cnf.jkt`, `act` chain) with collapsible JWT view
- **Vault retrieval card** — provider, fetched_at, last-4 chars of access token
- **Audit trail table** — action, actor, target, timestamp for all 4 events

The report can be opened from disk on any machine — no shark server needed to view it.

## Try it

```bash
# Terminal 1
shark serve

# Terminal 2
shark demo delegation-with-trace --admin-key sk_live_...

# Custom output path, no browser open
shark demo delegation-with-trace --admin-key sk_live_... --output /tmp/trace.html --no-open
```

## Why the vault hop matters

Without the vault hop, the demo shows "agents can delegate to each other" —
correct but abstract. With it, the report shows the closed loop: a third-hop
agent fetched an encrypted-at-rest credential from the vault with cryptographic
proof tying the action back to the original human user. The `act` chain in the
vault audit event is the receipt.

## Environment variables

| Variable          | Description                        |
|-------------------|------------------------------------|
| `SHARK_ADMIN_KEY` | Fallback admin key if `--admin-key` is not passed |
| `SHARK_BASE_URL`  | Fallback base URL (used by smoke tests) |
