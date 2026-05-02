# Cookbook: programmatic admin onboarding

Bootstrap a brand-new SharkAuth instance from zero — first admin key, an org, an app, an agent, a webhook — all from a script.

## Prerequisites

- A running `shark serve` instance on `https://auth.example.com`.
- A bootstrap admin key. SharkAuth emits one on first boot (CLI: `shark firstboot-key`). Persist it; the SDK does not yet ship a `firstboot-consume` helper (see `sdk/HANDOFF.md` P2).

## Script

```python
import os
from shark_auth import Client

BASE = os.environ["SHARK_AUTH_URL"]
BOOTSTRAP = os.environ["SHARK_BOOTSTRAP_KEY"]

c = Client(BASE, BOOTSTRAP)

# 1. Mint a scoped CI key.
ci_key = c.api_keys.create(
    name="ci-bot",
    scopes=["agents:write", "users:read", "webhooks:write", "audit:read"],
)
print("CI_KEY=", ci_key["key"])  # store in CI secret manager NOW

# 2. Create the production org.
org = c.organizations.create(
    name="Acme Co",
    slug="acme",
    metadata={"plan": "enterprise", "billing_id": "cus_abc"},
)

# 3. Register the application that agents will be scoped to.
app = c.apps.create(
    name="acme-inbox",
    integration_mode="custom",
    redirect_uris=["https://app.example.com/cb"],
)

# 4. Register the first agent.
agent = c.agents.register_agent(
    app_id=app["id"],
    name="acme-inbox-summarizer",
    scopes=["vault:read", "gmail:read"],
)
print("AGENT_CLIENT_ID=", agent["client_id"])
print("AGENT_SECRET=", agent["client_secret"])  # one-time

# 5. Subscribe a webhook for monitoring.
wh = c.webhooks.register(
    url="https://ops.example.com/shark-hooks",
    events=["agent.token_revoked", "agent.cascade_revoked", "user.deleted"],
    secret="whsec_" + os.urandom(24).hex(),
)
print("WEBHOOK_SECRET=", wh.get("secret"))

# 6. Sanity-check audit feed.
events = c.audit.list(limit=20)
print(len(events["events"]), "audit events emitted during onboarding")

c.close()
```

```typescript
import { SharkClient } from "@sharkauth/sdk";
import { randomBytes } from "node:crypto";

const c = new SharkClient({
  baseUrl: process.env.SHARK_AUTH_URL!,
  adminKey: process.env.SHARK_BOOTSTRAP_KEY!,
});

const ciKey = await c.apiKeys.create({
  name: "ci-bot",
  scopes: ["agents:write", "users:read", "webhooks:write", "audit:read"],
});
console.log("CI_KEY=", ciKey.key);

const org = await c.organizations.create(
  "Acme Co",
  "acme",
  { metadata: { plan: "enterprise", billing_id: "cus_abc" } }
);

const app = await c.apps.create({
  name: "acme-inbox",
  integration_mode: "custom",
  redirect_uris: ["https://app.example.com/cb"],
});

const agent = await c.agents.registerAgent({
  name: "acme-inbox-summarizer",
  scopes: ["vault:read", "gmail:read"],
  metadata: { app_id: app.id },
});
console.log("AGENT_CLIENT_ID=", agent.client_id);
console.log("AGENT_SECRET=", agent.client_secret);

const wh = await c.webhooks.register({
  url: "https://ops.example.com/shark-hooks",
  events: ["agent.token_revoked", "agent.cascade_revoked", "user.deleted"],
  secret: "whsec_" + randomBytes(24).toString("hex"),
});

const events = await c.audit.list({ limit: 20 });
console.log(events.events.length, "audit events emitted during onboarding");
```

## Idempotency

The script above will fail on the second run (orgs / apps / agents are unique by name or slug). Wrap each step:

```python
def get_or_create_org(c, name, slug):
    existing = next((o for o in c.organizations.list() if o["slug"] == slug), None)
    return existing or c.organizations.create(name=name, slug=slug)
```

## Storing secrets

The script prints three one-time secrets:

- `CI_KEY` — the API key for ongoing CI
- `AGENT_SECRET` — the OAuth client secret for the agent
- `WEBHOOK_SECRET` — used to verify incoming webhook signatures

Pipe to a secret manager (Vault, AWS Secrets Manager, GH Actions secrets) immediately. None of these can be re-read.

## Next steps

- [Webhooks](../webhooks.md) — implement the receiver and verify signatures
- [Delegation and agents](../delegation-and-agents.md) — wire up the agent's actual workload
- [Audit logs](../audit-logs.md) — set up retention + alerts on suspicious events
