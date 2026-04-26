# shark demo

Run interactive SharkAuth demo flows against a live server.

## Commands

### `shark demo delegation-with-trace`

Registers 3 synthetic agents, configures `may_act` policies, and runs a 3-hop
DPoP-bound token-exchange delegation chain, printing each token's `scope`,
`cnf.jkt`, and `act` chain to stdout.

**Requires a running `shark serve` instance.**

```
Usage:
  shark demo delegation-with-trace [flags]

Flags:
  --admin-key string   admin API key (or SHARK_ADMIN_KEY env)
  --base-url string    shark server base URL (default "http://localhost:8080")
  --html string        (W+1) write HTML report to this path; empty = plain stdout
  --plain              force plain stdout output (default true)
```

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
Vault retrieval: skipped (provision a vault entry to enable)

Run with --html to generate the visual report (W+1).
```

## Try it

```bash
# Terminal 1
shark serve

# Terminal 2
shark demo delegation-with-trace --admin-key sk_live_...
```

The demo registers 3 agents, runs a 3-hop DPoP-bound delegation chain, and
prints the resulting tokens with their `act` claims so you can see the lineage
end-to-end.

## Environment variables

| Variable          | Description                        |
|-------------------|------------------------------------|
| `SHARK_ADMIN_KEY` | Fallback admin key if `--admin-key` is not passed |
