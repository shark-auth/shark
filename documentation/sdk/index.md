# SDKs

Official SharkAuth client libraries. Two languages, identical surface. Honest endpoint coverage: 56% of all SDK-callable backend routes, ~85% of the launch-critical agent-platform + OAuth core path. See [Coverage](#coverage) below.

| Language   | Package                | Import                           |
| ---------- | ---------------------- | -------------------------------- |
| Python     | `shark-auth`           | `from shark_auth import Client`  |
| TypeScript | `@sharkauth/sdk`       | `import { SharkClient } from '@sharkauth/sdk'` |

Both wrap the same HTTP API. Most methods exist in both SDKs under the camelCase / snake_case equivalent, but a few parity gaps remain (see table below).

## Sections

### Foundations

- [Getting started](./getting-started.md) — install, env vars, first call
- [Authentication](./authentication.md) — signup, login, sessions, password reset, magic links
- [MFA](./mfa.md) — TOTP enroll / verify / challenge / recovery codes
- [Sessions](./sessions.md) — list and revoke active user sessions
- [Errors and retries](./errors-and-retries.md) — exception hierarchy, retry strategy

### OAuth surface

- [OAuth clients](./oauth-clients.md) — token, revoke, introspect, refresh, PKCE, authorize URL
- [Dynamic client registration](./dcr.md) — RFC 7591/7592 register, get, update, rotate
- [Device flow](./device-flow.md) — RFC 8628 device authorization grant
- [Token exchange](./token-exchange.md) — RFC 8693 exchange + downscope

### Agent platform (flagship)

- [Delegation and agents](./delegation-and-agents.md) — register agents, may_act, `act` chain, DPoP-bound tokens
- [DPoP primitives](./dpop.md) — `DPoPProver`, key rotation, `cnf.jkt` binding
- [Vault](./vault.md) — broker third-party OAuth tokens (Linear / Slack / Gmail / Microsoft / custom)

### Admin

- [API keys](./api-keys.md) — admin key CRUD + rotate
- [Organizations](./organizations.md) — org CRUD + members + invitations
- [RBAC](./rbac.md) — roles, permissions, assignments
- [Audit logs](./audit-logs.md) — query, export, purge
- [Webhooks](./webhooks.md) — register, verify signatures, replay

### Cookbook

- [Agent acts on user's Gmail](./cookbook/agent-acts-on-gmail.md)
- [Multi-hop delegation chain](./cookbook/multi-hop-delegation.md)
- [Programmatic admin onboarding](./cookbook/programmatic-admin-onboarding.md)

## Versioning

Both SDKs ship at `0.1.0`. The TypeScript build emits ESM + CJS + DTS (~280 KB total).

## Coverage

### Honest accounting

The backend exposes 236 total routes, but not all are SDK-callable. Excluding `/healthz`, the SPA shell, `.well-known/*`, `/api/docs`, static assets, redirects, and catch-all handlers, the **meaningful denominator is 130 SDK-callable endpoints**.

Against that denominator:

| SDK        | Endpoints wrapped | Raw coverage |
| ---------- | ----------------- | ------------ |
| Python     | 73 / 130          | **56%**      |
| TypeScript | 71 / 130          | **55%**      |

If you exclude the ~40 endpoints that are admin-panel-only (driven exclusively by the dashboard UI and not part of any documented SDK use case), the closest favorable read is **73 / 90 = 81%** for Python.

### Coverage by area

Of the launch-critical path (agent platform plus OAuth core), coverage is **~85%**. That is the wedge. Everything an agent author or OAuth integrator needs on day one is wrapped. Admin-panel parity, flow builder authoring, and SSO connection management lag and are explicitly post-launch.

| Area                  | Python | TypeScript | Status   |
| --------------------- | ------ | ---------- | -------- |
| Auth (signup / login / sessions / password / magic link) | 100% | 100% | Y        |
| OAuth core (token, revoke, introspect, refresh, PKCE, DCR, device flow, token exchange) | ~95% | ~95% | Y        |
| Agents (register, list, rotate, revoke)                  | 100% | 100% | Y        |
| Delegation (`may_act`, `act` chain, DPoP-bound tokens)   | 100% | 100% | Y        |
| Vault (broker tokens, list, refresh)                     | ~70% | ~70% | Partial  |
| Audit (query, export, purge)                             | 100% | 100% | Y        |
| Webhooks (register, verify, replay)                      | 100% | 100% | Y        |
| Admin Apps (CRUD)                                        | ~80% | ~80% | Y        |
| RBAC (roles, permissions, assignments)                   | 100% | 100% | Y        |
| Orgs (CRUD, members, invitations)                        | 100% | 100% | Y        |
| Proxy (admin proxy config + policy)                      | ~60% | ~60% | Partial  |
| Email / Templates                                        | ~50% | ~50% | Partial  |
| SSO (connection management)                              | 0%   | 0%   | N        |
| Passkeys (WebAuthn registration / auth)                  | 0%   | 0%   | N        |
| Flow Builder (CRUD)                                      | 0%   | 0%   | N        |

### Roadmap to 90%+

The gap between today's 56% raw coverage and 90%+ is concentrated in four shippable groups, in this priority order:

1. **Passkeys** (WebAuthn registration + authentication endpoints).
2. **Vault user-connect + provider-CRUD** (provider templates, user connect / disconnect, provider-level admin).
3. **SSO connection management** (create / list / update / delete connections, IdP metadata).
4. **Flow Builder** (CRUD on flows + steps).

Tracking and the full post-launch backlog live in `sdk/HANDOFF.md`. Other notable not-yet-shipped items: admin SSE log stream and signing-key rotation.

## How the two SDKs relate

| Concern              | Python                       | TypeScript                         |
| -------------------- | ---------------------------- | ---------------------------------- |
| Top-level client     | `Client(base_url, token)`    | `new SharkClient({ baseUrl, adminKey })` |
| Naming               | `snake_case`                 | `camelCase`                        |
| HTTP backend         | `requests`                   | global `fetch`                     |
| Crypto               | `cryptography`, `PyJWT`      | `globalThis.crypto.subtle`, `jose` |
| Token exchange       | `OAuthClient.token_exchange` | `exchangeToken(...)` (free fn)     |
| Webhook signature    | `verify_signature`           | `verifySignature` (async)          |
| JWT verify           | `decode_agent_token` (full)  | Not yet exported — decode manually   |
| Browser-safe         | n/a                          | yes, except DPoP key PEM round-trip |
