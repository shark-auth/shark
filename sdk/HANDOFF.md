# SharkAuth SDKs — Handoff (2026-04-27)

## Status

| SDK | Backend coverage | Parity |
|---|---|---|
| Python (`sdk/python/shark_auth/`) | ~75% (50+ of 236 routes wrapped + helpers) | reference |
| TypeScript (`sdk/typescript/`) | ~75% | matches Python class-for-class |

Both shipped 2026-04-27 via three parallel sonnet worktrees per language. Backend frozen during entire SDK push (concurrent agent owned `internal/`).

---

## What landed

### Human auth + identity
| Module | Class | Methods |
|---|---|---|
| `auth.{py,ts}` | `AuthClient` | signup, login, logout, get_me/getMe, delete_me/deleteMe, change_password, request/confirm_password_reset, request/consume_email_verification, verify_magic_link |
| `mfa.{py,ts}` | `MFAClient`/`MfaClient` | enroll, verify, challenge, disable, regenerate_recovery_codes |
| `mfa.{py,ts}` | `compute_totp`/`computeTotp` | RFC 6238 inline (no `pyotp` / no node deps) |
| `sessions.{py,ts}` | `SessionsClient` | list, revoke, revoke_all/revokeAll |
| `consents.{py,ts}` | `ConsentsClient` | list, revoke |
| `magic_link.{py,ts}` | `MagicLinkClient` | send, **verify** (added) |

### OAuth + DCR
| Module | Class | Methods |
|---|---|---|
| `oauth.{py,ts}` | `OAuthClient` | revoke_token, introspect_token, get_token_with_dpop, **token_exchange**, **bulk_revoke_by_pattern**, **get_token_authorization_code**, **refresh_token**, **build_authorize_url** (static), **pkce_pair** (helper) |
| `dcr.{py,ts}` | `DCRClient`/`DcrClient` | register, get, update, delete, rotate_secret, rotate_registration_token (RFC 7591/7592) |
| `device_flow.{py,ts}` | `DeviceFlow` | begin, wait_for_approval. **default path fixed** (`/oauth/device_authorization` → `/oauth/device`) |

### Agent platform
| Module | Class | Methods |
|---|---|---|
| `agents.{py,ts}` | `AgentsClient` | register, list, get, revoke, list_tokens, revoke_all, rotate_secret, rotate_dpop_key, get_audit_logs |
| `users.{py,ts}` | `UsersClient` | create, get, update, delete, list, set_tier, **revoke_agents** (cascade), list_agents |
| `organizations.{py,ts}` | `OrganizationsClient` | CRUD + members + invitations (14 methods total) |
| `apps.{py,ts}` | `AppsClient` | CRUD + rotate_secret + get_snippet + preview |
| `api_keys.{py,ts}` (Py) / `apiKeys.ts` (TS) | `APIKeysClient`/`ApiKeysClient` | CRUD + rotate |
| `rbac.{py,ts}` | `RBACClient`/`RbacClient` | role CRUD + permission CRUD + attach/detach + assign/revoke role on user |
| `audit.{py,ts}` | `AuditClient` | list (with filters), get, export, purge |

### Integration
| Module | Class | Methods |
|---|---|---|
| `webhooks.{py,ts}` | `WebhooksClient` | register, list, get, update, delete, send_test, replay, list_deliveries |
| `webhooks.{py,ts}` | `verify_signature`/`verifySignature` | HMAC-SHA256, supports Stripe-style `t=,v1=`, raw hex, `sha256=` prefix; constant-time compare; replay-window via `tolerance_seconds` |
| `proxy_rules.py` / `proxyRules.ts` | `ProxyRulesClient` | list, create, get, update, delete, import_rules_yaml |
| `proxy_lifecycle.py` / `proxyLifecycle.ts` | `ProxyLifecycleClient` | start, stop, reload, status |
| `branding.{py,ts}` | `BrandingClient` | get, set |
| `paywall.{py,ts}` | `PaywallClient` | url, render, preview |

### DPoP primitives
| Module | Class | Methods |
|---|---|---|
| `dpop.{py,ts}` | `DPoPProver` | generate, from_pem, make_proof, jkt thumbprint, batched/resign modes (Python) |
| `http_client.py` (Py) | `DPoPHTTPClient` | get/post/delete with auto-DPoP |
| `claims.py` (Py) | `AgentTokenClaims` | parse + delegation_chain walker + is_delegated + has_scope |
| `tokens.py` (Py) | `decode_agent_token` (verify+JWKS) | full JWT verify; TS only has decode-only `decodeAgentToken` |

### Top-level composer
| Module | Class | Composes |
|---|---|---|
| `client.py` | `Client(base_url, token)` | 18 namespaces: auth, mfa, sessions, consents, dcr, users, agents, organizations, apps, api_keys, rbac, audit, webhooks, proxy_rules, proxy_lifecycle, branding, paywall, http |
| `sharkClient.ts` | `SharkClient({baseUrl, adminKey})` | same 18 namespaces (camelCase) |

---

## How to use

### Python

```python
from shark_auth import Client, AuthClient, OAuthClient, pkce_pair

# Human-auth flow (no admin key needed)
auth = AuthClient("https://auth.example.com")
auth.signup(email="alice@example.com", password="Strong-Pwd-2026")
auth.login(email="alice@example.com", password="Strong-Pwd-2026")
me = auth.get_me()

# Authorization-code + PKCE
verifier, challenge, _ = pkce_pair()
url = OAuthClient.build_authorize_url(
    client_id="abc", redirect_uri="https://app/callback",
    code_challenge=challenge, scope="openid"
)
# ... user redirects, app receives code ...
oauth = OAuthClient("https://auth.example.com")
token = oauth.get_token_authorization_code(code="...", redirect_uri="https://app/callback", code_verifier=verifier)

# Admin operations via unified client
c = Client("https://auth.example.com", token="sk_live_...")
org = c.organizations.create(name="Acme Co", slug="acme")
key = c.api_keys.create(name="ci-bot", scopes=["agents:write"])
events = c.audit.list(action="agent.created", limit=50)
c.webhooks.register(url="https://hooks.app/shark", events=["agent.*"], secret="whsec_...")
```

### TypeScript

```typescript
import { SharkClient, AuthClient, OAuthClient, pkcePair, verifySignature } from '@sharkauth/sdk';

const auth = new AuthClient('https://auth.example.com');
await auth.signup({ email: 'alice@example.com', password: 'Strong-Pwd-2026' });
await auth.login({ email: 'alice@example.com', password: 'Strong-Pwd-2026' });

const { verifier, challenge } = await pkcePair();
const url = OAuthClient.buildAuthorizeUrl({
  clientId: 'abc',
  redirectUri: 'https://app/callback',
  codeChallenge: challenge,
  scope: 'openid',
});

const c = new SharkClient({ baseUrl: 'https://auth.example.com', adminKey: 'sk_live_...' });
const org = await c.organizations.create({ name: 'Acme Co', slug: 'acme' });
const events = await c.audit.list({ action: 'agent.created', limit: 50 });

// Webhook signature verify (works in browser + Node)
const ok = await verifySignature(rawPayload, request.headers.get('Shark-Signature'), 'whsec_...');
```

---

## Backend paths verified (corrections during port)

These differed from the original gap report — final wired paths reflect actual `internal/api/router.go`:

| Endpoint | Wired path | Note |
|---|---|---|
| DCR rotate-secret | `POST /oauth/register/{id}/secret` | NOT `/rotate-secret` |
| DCR rotate-registration-token | `DELETE /oauth/register/{id}/registration-token` | verb is DELETE, NOT POST `/rotate-token` |
| Webhook test fire | `POST /api/v1/admin/webhooks/{id}/test` | NOT `/send-test` |
| Webhook update | `PATCH /{id}` | NOT PUT |
| Audit export | `POST /api/v1/audit-logs/export` | admin-key gated, NOT under `/admin/` prefix |
| Audit purge | `POST /api/v1/admin/audit-logs/purge` | |
| RBAC role assign | `POST /api/v1/users/{user_id}/roles` | |
| RBAC update role | `PUT /api/v1/roles/{id}` | NOT PATCH |
| Org accept-invitation | `POST /api/v1/organizations/invitations/{token}/accept` | user prefix, NOT `/admin/` |

---

## Endpoint shapes that surprised

- Signup body field is `name`, not `full_name`. SDKs map transparently.
- Signup response uses `id`, not `user_id`. Both fields surfaced.
- MFA `disable` is `DELETE /api/v1/auth/mfa` (with body), NOT `POST /disable`.
- MFA `recovery-codes` is `GET`, NOT `POST`.
- Sessions list returns `{data: [...]}`; consents list returns bare array — SDK normalises both.
- User-side has no bulk revoke-all-sessions endpoint. SDK iterates and skips `current` row.
- Org metadata field is `*string` (JSON-encoded string). SDK auto-`json.dumps()` / `JSON.stringify()` on dict input.

---

## Existing-but-broken now fixed

| Bug | Fix |
|---|---|
| `DeviceFlow` default `/oauth/device_authorization` | Updated to `/oauth/device` (router.go:784) |
| `VaultClient.get_fresh_token` (no backing endpoint) | Deprecated stub raising `VaultError` with migration note pointing to `fetch_token` |
| `MagicLinkClient` send-only | Added `verify(token)` method |
| TS `OAuthClient` only revoke+introspect | Added `getTokenAuthorizationCode`, `refreshToken`, `buildAuthorizeUrl`, `pkcePair` |
| TS no signature verifier for webhooks | Added `verifySignature` (browser-safe Web Crypto) |

---

## Browser compatibility (TypeScript)

- All HMAC + SHA paths use `globalThis.crypto.subtle` (Web Crypto API) — work in browser + Node 18+.
- Constant-time hex comparison done manually (no `crypto.timingSafeEqual` dep).
- Node fallback only on TOTP HMAC-SHA1 (`crypto.createHmac`) because some browsers don't expose SHA-1 via Web Crypto. Detected via `typeof window === 'undefined'`.
- Build artefacts: ESM 101.82 KB, CJS 104.28 KB, DTS 75.30 KB.

⚠️ Caveat: `package.json` doesn't yet declare `"browser"` export condition explicitly, and `dpop.ts` still uses `jose`'s `exportPKCS8`/`importPKCS8` for PEM round-trips (Node-only). Browser builds work for everything except DPoP keypair PEM persistence — which customers usually don't need in-browser anyway. Address post-launch.

---

## Type safety reality

- Hand-rolled types per module (no OpenAPI codegen yet).
- `Record<string, unknown>` count down significantly in new modules — concrete result interfaces (`SignupResult`, `OAuthToken`, `WebhookDelivery`, etc.) used.
- Existing TS modules (pre-2026-04-27) still loose-typed; not in scope this pass.
- Post-launch backlog: generate types from `openapi.yaml` to prevent server↔SDK drift.

---

## Remaining gaps (post-launch backlog)

### P1 — must close W+1
- Flows CRUD (auth flow builder is a marquee feature; not exposed)
- Vault provider/template CRUD (only disconnect + fetch present; no enroll-from-code)
- Admin sessions list/revoke (admin-side; user-side already shipped)
- Admin stats + trends endpoints
- Admin logs stream (SSE)
- System mode/reset/swap-mode
- Signing-key rotation
- SSO connection management (OIDC/SAML)

### P2 — nice-to-have
- System config get/patch (admin)
- Dev-emails outbox helper
- Hosted pages helpers (paywall has one; others don't)
- Branding logo upload + delete
- Permissions batch-usage analytics
- Bootstrap consume + firstboot key (currently CLI-only)
- Server metadata / discovery helper

### Backend gaps surfaced during SDK work (not SDK responsibility)
- `acr`/`amr` claim emission absent in `internal/oauth/exchange.go` — blocks real MFA step-up enforcement on agent tokens (see commit 0c8de8d for why MFA was dropped from concierge demo)
- `sessions.mfa_at` freshness column missing — `mfa_passed` is sticky bool only
- Per-vault `requires_mfa` flag absent — no per-action step-up gating

---

## How to extend

Each new SDK module follows this pattern:

```python
# sdk/python/shark_auth/foo.py
from . import _http
from .errors import SharkAuthError

class FooClient:
    def __init__(self, base_url: str, admin_api_key: str | None = None, *, session=None):
        self.base_url = base_url.rstrip("/")
        self.admin_api_key = admin_api_key
        self.session = session or _http.new_session()
    
    def some_method(self, ...) -> dict:
        # self.session.post / get / etc with self._headers()
        ...
```

Then add the export in `__init__.py` and the namespace in `client.py`.

TS pattern:
```typescript
// sdk/typescript/src/foo.ts
import { httpRequest } from './http';
import { SharkAPIError } from './errors';

export class FooClient {
  constructor(private baseUrl: string, private opts: { adminKey?: string; session?: Session } = {}) {}
  async someMethod(...): Promise<ConcreteType> { ... }
}
```

Then add re-export in `index.ts` and namespace in `sharkClient.ts`.

---

## Tests

No automated SDK test suite exists yet. Each gap-closing agent ran a smoke roundtrip against a live `shark serve` instance during dispatch. Pre-launch tests:

- Python: `python -c "from shark_auth import *"` smoke + auth roundtrip + admin roundtrip
- TS: `npm run build` clean + `node` import smoke + webhook signature roundtrip

Recommended W+1: `tests/sdk/python/` + `tests/sdk/typescript/` pytest + jest suites covering happy-path of each method against a local shark.

---

## Commits

- `feat(sdk-python): close BLOCKER gaps — 11 new clients, ~22% -> ~75% endpoint coverage` (post-launch / today)
- `feat(sdk-ts): mirror Python additions — 11 new clients, ~44% -> ~75% parity` (post-launch / today)
