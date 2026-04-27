# Custom vault provider limitations (v0.1)

**Status:** v0.1 ships textbook OAuth 2.0 auth-code only. Provider-specific quirks land in v0.2.

## What works in v0.1

`POST /api/v1/vault/providers` with `template == ""` accepts a custom provider definition:

```json
{
  "name": "my-saas",
  "display_name": "My SaaS",
  "auth_url": "https://my-saas.example/oauth/authorize",
  "token_url": "https://my-saas.example/oauth/token",
  "client_id": "abc",
  "client_secret": "xyz",
  "scopes": ["read", "write"],
  "icon_url": "https://my-saas.example/favicon.ico"
}
```

Backend persists to `storage.VaultProvider`. `BuildAuthURL` issues a vanilla `oauth2.Config.AuthCodeURL(state, AccessTypeOffline)`. Token exchange goes through `golang.org/x/oauth2` defaults. Refresh uses the standard `TokenSource` interface.

This works **only** when the provider:

1. Speaks RFC 6749 auth-code grant (no PKCE-required, no JWT-bearer, no client-credentials-only).
2. Returns a standard token response (`access_token`, `refresh_token`, `expires_in`, `token_type` at top level — no nesting).
3. Does not require post-exchange API calls before tokens are usable.
4. Does not require non-OAuth headers on token-endpoint requests.
5. Uses space-delimited scopes.
6. Issues non-rotating refresh tokens, OR rotates with a standard refresh-token-in-response flow.
7. Returns a structured error matching RFC 6749 `{error, error_description}`.

## What does NOT work in v0.1

Any provider with a quirk in the matrix below cannot be onboarded via the custom-provider form. They need code changes in `internal/vault/providers.go` + handler hooks.

| Quirk | Affected providers | Consequence in v0.1 | Fix complexity |
|---|---|---|---|
| Required extra authorize-URL params (`prompt=consent`, `audience=...`) | Linear, Jira, some Microsoft tenants | **FIXED v0.1** — `extra_auth_params` persisted on `VaultProvider`; templates copy defaults, manual providers accept via API body. Admin UI key/value editor deferred to v0.2 (backend complete). | DONE |
| Non-standard token response (nested `access_token`, `ok` flag, multi-token) | Slack v2 (`authed_user.access_token`) | Wrong token stored silently, OR empty token stored on `ok:false` 200 OK | MEDIUM — needs custom unmarshaler hook + provider-type discriminator |
| Post-exchange step required to make token usable | Atlassian (`accessible-resources` lookup for cloud ID) | Token retrieved but unusable; caller cannot construct API URLs | MEDIUM — needs `post_exchange_call` config + storage column for derived metadata |
| Required headers on every API call (not OAuth, but downstream) | Notion (`Notion-Version`) | Out of vault scope; consumers must add header themselves. Document only. | DOC ONLY |
| Rotating refresh tokens with no-overwrite race | Atlassian | Refresh succeeds but persistence failure permanently bricks connection | MEDIUM — needs atomic refresh+persist; add idempotent retry with old token grace window |
| No refresh tokens, long-lived access only | Notion | `isExpired(nil)` is correct (works), but `Vault.GetFreshToken` semantics confusing | DOC ONLY |
| Comma-delimited scope format on token endpoint | Slack | `golang.org/x/oauth2` joins with space; provider may accept either or fail | LOW — `scope_delimiter` config field |
| Tenant-specific token endpoint | Microsoft single-tenant | Hardcoded `/common/` 403s on single-tenant Entra apps | LOW — admin already supplies `token_url`; document the `/{tenant_id}/` swap |
| Non-Bearer token type | rare | `TokenType` stored but caller may ignore; default Bearer assumed | LOW — already stored, document |
| PKCE required (provider rejects non-PKCE) | some financial APIs | Not toggleable — always disabled | LOW — `pkce_required: bool` flag |
| Non-RFC error responses | Slack (HTTP 200 + `ok:false`) | Errors hidden behind apparent success | MEDIUM — provider-type discriminator |
| Grant types other than auth-code | enterprise SaaS using JWT bearer or client_credentials | Not supported | HIGH — separate flow path |

## v0.2 design sketch (informational, not a commitment)

Two non-exclusive options under consideration:

### Option A — schema extension

Add JSON columns to `storage.VaultProvider`:

```go
type VaultProvider struct {
    // existing fields...
    ExtraAuthParams    map[string]string // SHIPPED v0.1 — {"prompt": "consent", "audience": "api.atlassian.com"}
    TokenResponseShape string            // "rfc6749" | "slack_v2" | "atlassian_3lo"
    PostExchangeCall   *PostExchangeSpec // optional URL + token-substitution
    ScopeDelimiter     string            // " " (default) | ","
    PKCERequired       bool
    GrantType          string            // "authorization_code" (default) | "client_credentials" | "device_code"
    RequiredHeaders    map[string]string // documented for consumers; not auto-injected
}
```

**`ExtraAuthParams` SHIPPED in v0.1** (migration 00027, 2026-04-27). Backend persists per-provider; API accepts in POST + PATCH bodies; `BuildAuthURL` reads from storage first.
Admin UI key/value editor for `extra_auth_params` on the Config tab is deferred to v0.2 — backend fully supports it, UI lands later. Workaround: use the admin API directly.

Backend dispatches on `TokenResponseShape` to a switch of unmarshalers. Atlassian's `accessible-resources` becomes a generic `PostExchangeSpec` runner.

Covers ~70% of real providers. Solves Linear, Jira, Slack, Microsoft single-tenant, Notion (header docs), most enterprise SaaS. Does NOT solve fundamentally non-OAuth-2.0 providers (e.g., Atlassian server with custom token format).

Estimated effort: 15–20h.

### Option B — plugin/adapter hooks

Go plugins (`.so`) or WASM modules loaded at startup. Each adapter implements:

```go
type ProviderAdapter interface {
    BuildAuthURL(state string, cfg ProviderConfig) (string, error)
    ParseTokenResponse(body []byte) (Token, error)
    PostExchange(token Token, cfg ProviderConfig) (Token, error)
    RefreshToken(refreshToken string, cfg ProviderConfig) (Token, error)
    MapError(httpStatus int, body []byte) error
}
```

Built-in adapters live in `internal/vault/adapters/`. Custom adapters drop into `./data/adapters/`. Handles arbitrary providers including non-standard ones.

Estimated effort: 35–45h. Defer until v0.3.

## v0.1 user-facing communication

README must say what custom providers actually support. Recommended L109 framing:

> **Token vault** — encrypted storage for customer OAuth tokens. Verified: Google (Gmail/Drive/Calendar), GitHub. Experimental: Slack, Microsoft (multi-tenant only), Notion. Linear and Jira ship in v0.2 with full quirk handling. Custom OAuth 2.0 providers supported for textbook auth-code flows — non-standard token responses, post-exchange steps, and rotating-refresh quirks land in v0.2.

Admin UI form (`vault_manage.tsx` create wizard) needs an inline note when `template == ""`:

> Custom providers in v0.1 require a textbook OAuth 2.0 auth-code provider with standard token response. If your provider needs `prompt=consent`, custom token parsing, or post-exchange steps, file an issue or wait for v0.2.

## Tracking

- v0.2 implementation: see Option A sketch above. Owner: solo dev. Target: ~3 weeks post-launch.
- Real-client smoke tests for built-in providers: `playbook/16-launch-week-yc-play.md` references vault testing — gap identified by 2026-04-27 audit.
- Provider-specific quirks documented above are sourced from 2026-04-27 vault audit (`internal/vault/providers.go:38-44` TODO and audit findings).
