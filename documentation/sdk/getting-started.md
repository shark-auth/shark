# Getting started

Install the SDK, point it at your SharkAuth server, make a call. Two minutes.

## Install

```bash
# Python (3.9+)
pip install shark-auth

# TypeScript / Node 18+
npm install @sharkauth/sdk
```

The TypeScript build is ESM-first with CJS fallback. It runs in browsers (Web Crypto) and Node 18+.

## Environment

The SDKs do not read environment variables themselves — pass everything explicitly. Conventional names if you wire your own loader:

| Variable               | Used for                                |
| ---------------------- | --------------------------------------- |
| `SHARK_AUTH_URL`       | Base URL of your SharkAuth instance     |
| `SHARK_ADMIN_KEY`      | `sk_live_...` admin key                 |
| `SHARK_AGENT_CLIENT_ID`| `shark_agent_...` for an agent          |
| `SHARK_AGENT_SECRET`   | Agent client secret                     |

## First call — admin client

```python
import os
from shark_auth import Client

c = Client(
    base_url=os.environ["SHARK_AUTH_URL"],
    token=os.environ["SHARK_ADMIN_KEY"],
)

users = c.users.list_users()
print(len(users), "users")
```

```typescript
import { SharkClient } from "@sharkauth/sdk";

const c = new SharkClient({
  baseUrl: process.env.SHARK_AUTH_URL!,
  adminKey: process.env.SHARK_ADMIN_KEY!,
});

const users = await c.users.listUsers();
console.log(users.total, "users");
```

## First call — public auth

No admin key needed for end-user auth. The Python `AuthClient` keeps a `requests.Session` so the cookie planted by `/login` flows through. The TS `AuthClient` does the same with a small built-in cookie shim for Node and `credentials: "include"` for the browser.

```python
from shark_auth import AuthClient

auth = AuthClient("https://auth.example.com")
auth.signup(email="alice@example.com", password="Strong-Pwd-2026", full_name="Alice")
auth.login("alice@example.com", "Strong-Pwd-2026")
me = auth.get_me()
```

```typescript
import { AuthClient } from "@sharkauth/sdk";

const auth = new AuthClient("https://auth.example.com");
await auth.signup("alice@example.com", "Strong-Pwd-2026", { name: "Alice" });
await auth.login("alice@example.com", "Strong-Pwd-2026");
const me = await auth.getMe();
```

## What's in `Client` / `SharkClient`

The composer client exposes 18 namespaces. Same shape in both languages:

```
auth, mfa, sessions, consents, dcr,
users, agents, organizations, apps, api_keys, rbac, audit,
webhooks, proxy_rules, proxy_lifecycle, branding, paywall, http
```

Each namespace is also importable directly if you want to skip the composer:

```python
from shark_auth import AgentsClient, AuditClient
```

```typescript
import { AgentsClient, AuditClient } from "@sharkauth/sdk";
```

## Two auth modes

| Mode             | Python                    | TypeScript                 | When                         |
| ---------------- | ------------------------- | -------------------------- | ---------------------------- |
| Admin API key    | `Client(url, "sk_live_")` | `new SharkClient({...})`   | Server-side automation       |
| User session     | `AuthClient(url)`         | `new AuthClient(url)`      | Acting as an end user        |
| Agent + DPoP     | `OAuthClient` + `DPoPProver` | `OAuthClient` + `DPoPProver` | Machine-to-machine, MCP, delegation |

For the agent path, see [Delegation and agents](./delegation-and-agents.md).

## Next

- [Authentication](./authentication.md) — full human-auth surface
- [Delegation and agents](./delegation-and-agents.md) — flagship: agents acting on users' behalf
- [Errors and retries](./errors-and-retries.md) — what to catch
