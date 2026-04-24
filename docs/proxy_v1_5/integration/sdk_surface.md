# Integration — SDK surface

Every v1.5 proxy admin endpoint with its target method name in TypeScript (camelCase) and Python (snake_case), plus a one-line signature sketch. Shapes reference the existing doc files — SDK authors should marshal payloads exactly as `contracts/rule_shape.md` / `contracts/require_grammar.md` describe.

## Proxy rules CRUD

Doc: `api/admin_proxy_rules_db.md`

| HTTP | TS method | Python method | Signature sketch |
|---|---|---|---|
| GET    /api/v1/admin/proxy/rules/db       | `listProxyRules(appID?: string)`                     | `list_proxy_rules(app_id: str \| None = None)`   | `Promise<ProxyRule[]>` / `list[ProxyRule]` |
| POST   /api/v1/admin/proxy/rules/db       | `createProxyRule(input: CreateProxyRuleInput)`       | `create_proxy_rule(input: CreateProxyRuleInput)` | `Promise<ProxyRule>` / `ProxyRule` |
| GET    /api/v1/admin/proxy/rules/db/{id}  | `getProxyRule(id: string)`                           | `get_proxy_rule(rule_id: str)`                   | `Promise<ProxyRule>` / `ProxyRule` |
| PATCH  /api/v1/admin/proxy/rules/db/{id}  | `updateProxyRule(id: string, patch: UpdateProxyRuleInput)` | `update_proxy_rule(rule_id: str, patch: UpdateProxyRuleInput)` | `Promise<ProxyRule>` / `ProxyRule` |
| DELETE /api/v1/admin/proxy/rules/db/{id}  | `deleteProxyRule(id: string)`                        | `delete_proxy_rule(rule_id: str)`                | `Promise<void>` / `None` |
| POST   /api/v1/admin/proxy/rules/import   | `importProxyRulesYAML(yaml: string)`                 | `import_proxy_rules_yaml(yaml: str)`             | `Promise<ImportResult>` / `ImportResult` |

## Proxy lifecycle

Doc: `api/admin_proxy_lifecycle.md`

| HTTP | TS method | Python method | Signature sketch |
|---|---|---|---|
| GET  /api/v1/admin/proxy/lifecycle | `getProxyLifecycleStatus()` | `get_proxy_lifecycle_status()` | `Promise<ProxyStatus>` / `ProxyStatus` |
| POST /api/v1/admin/proxy/start     | `startProxy()`              | `start_proxy()`                | `Promise<ProxyStatus>` / `ProxyStatus` |
| POST /api/v1/admin/proxy/stop      | `stopProxy()`               | `stop_proxy()`                 | `Promise<ProxyStatus>` / `ProxyStatus` |
| POST /api/v1/admin/proxy/reload    | `reloadProxy()`             | `reload_proxy()`               | `Promise<ProxyStatus>` / `ProxyStatus` |

## User tier

Doc: `api/admin_users_tier.md`

| HTTP | TS method | Python method | Signature sketch |
|---|---|---|---|
| PATCH /api/v1/admin/users/{id}/tier | `setUserTier(userID: string, tier: "free" \| "pro")` | `set_user_tier(user_id: str, tier: Literal["free", "pro"])` | `Promise<{ user: User; tier: string }>` / `dict` |

## Branding design tokens

Doc: `api/admin_branding_design_tokens.md`

| HTTP | TS method | Python method | Signature sketch |
|---|---|---|---|
| PATCH /api/v1/admin/branding/design-tokens | `setBrandingDesignTokens(tokens: Record<string, unknown>)` | `set_branding_design_tokens(tokens: dict)` | `Promise<{ branding: Branding; design_tokens: object }>` / `dict` |

## Paywall (read-only URL)

Doc: `api/paywall_route.md`

| HTTP | TS method | Python method | Signature sketch |
|---|---|---|---|
| GET /paywall/{app_slug} | `paywallURL(appSlug: string, tier: string, returnURL?: string)` | `paywall_url(app_slug: str, tier: str, return_url: str \| None = None)` | `string` / `str` (URL builder, not a call) |

## Shared types

All SDK methods share a small set of types. Define these once and re-use:

### TypeScript

```ts
export interface ProxyRule {
  id: string; app_id: string; name: string; pattern: string;
  methods: string[]; require: string; allow: string; scopes: string[];
  enabled: boolean; priority: number; tier_match: string; m2m: boolean;
  created_at: string; updated_at: string;
}

export interface CreateProxyRuleInput {
  app_id?: string; name: string; pattern: string;
  methods?: string[]; require?: string; allow?: string; scopes?: string[];
  enabled?: boolean; priority?: number; tier_match?: string; m2m?: boolean;
}

export type UpdateProxyRuleInput = Partial<CreateProxyRuleInput>;

export interface ProxyStatus {
  state: number; state_str: "stopped" | "running" | "reloading" | "unknown";
  listeners: number; rules_loaded: number;
  started_at: string; last_error: string;
}

export interface ImportResult {
  imported: number;
  errors: Array<{ index: string; name: string; error: string }>;
}
```

### Python

```python
from typing import TypedDict, Literal

class ProxyRule(TypedDict):
    id: str; app_id: str; name: str; pattern: str
    methods: list[str]; require: str; allow: str; scopes: list[str]
    enabled: bool; priority: int; tier_match: str; m2m: bool
    created_at: str; updated_at: str

class CreateProxyRuleInput(TypedDict, total=False):
    app_id: str; name: str; pattern: str
    methods: list[str]; require: str; allow: str; scopes: list[str]
    enabled: bool; priority: int; tier_match: str; m2m: bool

class ProxyStatus(TypedDict):
    state: int
    state_str: Literal["stopped", "running", "reloading", "unknown"]
    listeners: int; rules_loaded: int; started_at: str; last_error: str

class ImportResult(TypedDict):
    imported: int
    errors: list[dict]  # { index, name, error }
```

## Auth

Every admin-gated method takes the admin API key from the shared SDK client (e.g. `SharkClient({ adminKey })`). No per-method bearer argument.

## Error handling

Both SDKs should surface the server error envelope as a typed exception:

```ts
class SharkAPIError extends Error { code: string; status: number; }
```

```python
class SharkAPIError(Exception):
    code: str
    status: int
```

The `code` is the server's `error.code` (e.g. `invalid_proxy_rule`, `proxy_start_failed`), `status` is the HTTP status code. Callers can branch on either — preferring `code` for programmatic handling and `status` for generic UI mapping.
