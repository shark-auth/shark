# Demo 02 — N-Level Agent Delegation Chain

RFC 8693 token-exchange demo. 5 agents, 3-deep `act` chain, per-hop scope narrowing, surgical revocation.

See `../DEMO_02_DELEGATION_CHAIN.md` for the full plan, persona, compliance crosswalk.

## Quickstart

```bash
# 1. Start Shark (separate terminal)
shark serve

# 2. Install Python SDK
python3 -m venv .venv && source .venv/bin/activate
pip install -e ../../sdk/python
pip install requests pyjwt cryptography

# 3. Seed 5 agents + write .env
export SHARK_ADMIN_KEY=your_admin_key
bash seed.sh

# 4. Run the orchestrator (end-to-end)
python orchestrator/main.py

# 5. (3 days later) audit replay
python audit_replay.py --request-id $(cat .last_request_id)

# 6. Surgical revocation
python orchestrator/revoke.py email-agent --reason key_compromise
```

## File map

```
delegation/
├── README.md                        this file
├── seed.sh                          registers 5 agents, writes .env
├── .env.example                     credential template
├── lib/
│   ├── __init__.py
│   ├── exchange.py                  RFC 8693 wrapper (passes audience param SDK doesn't expose)
│   └── decode.py                    decode + render act chain
├── agents/
│   ├── __init__.py
│   ├── triage.py                    hop 1 — gets full user scope
│   ├── knowledge.py                 hop 2 — kb:read only
│   ├── email.py                     hop 2 — email:draft (then chains to gmail-tool)
│   ├── gmail_tool.py                hop 3 — email:send (3-deep act chain)
│   ├── crm.py                       hop 2 — crm:write
│   └── followup.py                  hop 2 — calendar:write
├── orchestrator/
│   ├── main.py                      runs full demo end-to-end
│   └── revoke.py                    revokes one agent + verifies blast radius
├── resources/
│   └── mock_resource.py             stub resource server — decodes + prints act chain
└── audit_replay.py                  queries oauth_tokens for the delegation timeline
```

## Honest gaps (from plan §11)

- `exchange.go` logs via `slog.Info("oauth.token.exchanged", ...)` — does **not** call `audit.Logger.Log()`. `audit_replay.py` falls back to querying the `oauth_tokens` table (`delegation_subject` + `delegation_actor` columns).
- Per-hop `cnf.jkt` works because each agent has its own DPoP keypair from `client_credentials` issuance, NOT because exchange.go re-binds at exchange time. Output is still correct.
- `may_act` enforcement is wired but the claim must be pre-seeded at subject-token issuance. `seed.sh` documents how to seed it via the admin agents API.
- Subject-token JTI is **not** auto-revoked after exchange — replay-attack UAT item is aspirational.
